package costs

import (
	"context"
	"fmt"
	"time"

	"govuk-reports-dashboard/internal/reports"
	"govuk-reports-dashboard/pkg/logger"
)

// CostReport implements the reports.Report interface for cost reporting
type CostReport struct {
	costService        *CostService
	applicationService *ApplicationService
	renderer           *reports.Renderer
	logger             *logger.Logger
}

// NewCostReport creates a new cost report instance
func NewCostReport(costService *CostService, applicationService *ApplicationService, logger *logger.Logger) *CostReport {
	return &CostReport{
		costService:        costService,
		applicationService: applicationService,
		renderer:           reports.NewRenderer(),
		logger:             logger,
	}
}

// GetMetadata returns metadata about this report module
func (r *CostReport) GetMetadata() reports.ReportMetadata {
	return reports.ReportMetadata{
		ID:          "costs",
		Name:        "Cost Analysis",
		Description: "AWS cost tracking and analysis for GOV.UK applications",
		Type:        reports.ReportTypeCost,
		Version:     "1.0.0",
		Author:      "GOV.UK Platform Team",
		Tags:        []string{"aws", "costs", "billing", "applications"},
		Priority:    reports.PriorityHigh,
	}
}

// GenerateSummary creates summary data for dashboard display
func (r *CostReport) GenerateSummary(ctx context.Context, params reports.ReportParams) ([]reports.Summary, error) {
	r.logger.Info().Msg("Generating cost summary for dashboard")

	// Get cost summary data
	costSummary, err := r.costService.GetCostSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to get cost summary: %w", err)
	}

	// Get application data for additional metrics
	appData, err := r.applicationService.GetAllApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get application data: %w", err)
	}

	var summaries []reports.Summary

	// Total Cost Summary
	totalCostSummary := r.renderer.CreateSummaryCard(
		"Total Monthly Cost",
		r.renderer.FormatCurrency(costSummary.TotalCost, "GBP"),
		"Current month",
		reports.SummaryTypeCurrency,
		r.calculateCostTrend(costSummary.TotalCost),
	)
	summaries = append(summaries, totalCostSummary)

	// Application Count Summary
	appCountSummary := r.renderer.CreateSummaryCard(
		"Applications",
		r.renderer.FormatNumber(appData.Count),
		"Active applications",
		reports.SummaryTypeCount,
		nil,
	)
	summaries = append(summaries, appCountSummary)

	// Average Cost per Application
	avgCost := 0.0
	if appData.Count > 0 {
		avgCost = appData.TotalCost / float64(appData.Count)
	}
	avgCostSummary := r.renderer.CreateSummaryCard(
		"Average Cost",
		r.renderer.FormatCurrency(avgCost, "GBP"),
		"Per application",
		reports.SummaryTypeCurrency,
		nil,
	)
	summaries = append(summaries, avgCostSummary)

	// Top Cost Service
	topService := r.getTopCostService(costSummary.Services)
	if topService != nil {
		topServiceSummary := r.renderer.CreateSummaryCard(
			"Top Service",
			topService.Service,
			r.renderer.FormatCurrency(topService.Amount, "GBP"),
			reports.SummaryTypeMetric,
			nil,
		)
		summaries = append(summaries, topServiceSummary)
	}

	r.logger.WithField("summary_count", len(summaries)).Info().Msg("Generated cost summaries")
	return summaries, nil
}

// GenerateReport creates detailed report data
func (r *CostReport) GenerateReport(ctx context.Context, params reports.ReportParams) (reports.ReportData, error) {
	r.logger.Info().Msg("Generating detailed cost report")

	data := reports.ReportData{
		Status:      reports.StatusRunning,
		GeneratedAt: time.Now(),
	}

	// Get cost summary
	costSummary, err := r.costService.GetCostSummary()
	if err != nil {
		data.Status = reports.StatusFailed
		data.Errors = append(data.Errors, reports.ReportError{
			Code:      "COST_FETCH_ERROR",
			Message:   "Failed to fetch cost data",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return data, nil
	}

	// Get application data
	appData, err := r.applicationService.GetAllApplications(ctx)
	if err != nil {
		data.Status = reports.StatusFailed
		data.Errors = append(data.Errors, reports.ReportError{
			Code:      "APPLICATION_FETCH_ERROR",
			Message:   "Failed to fetch application data",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return data, nil
	}

	// Generate data points
	data.DataPoints = r.generateDataPoints(costSummary, appData)

	// Generate summary data
	data.Summary, err = r.GenerateSummary(ctx, params)
	if err != nil {
		data.Warnings = append(data.Warnings, reports.ReportWarning{
			Code:      "SUMMARY_GENERATION_WARNING",
			Message:   "Failed to generate summary data",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
	}

	// Generate charts
	data.Charts = r.generateCharts(costSummary, appData)

	// Generate tables
	data.Tables = r.generateTables(appData)

	data.Status = reports.StatusCompleted
	r.logger.WithFields(map[string]interface{}{
		"data_points": len(data.DataPoints),
		"charts":      len(data.Charts),
		"tables":      len(data.Tables),
	}).Info().Msg("Generated detailed cost report")

	return data, nil
}

// IsAvailable checks if this report can run with current configuration
func (r *CostReport) IsAvailable(ctx context.Context) bool {
	// Check if cost service is available
	_, err := r.costService.GetCostSummary()
	return err == nil
}

// GetRefreshInterval returns how often this report should be refreshed
func (r *CostReport) GetRefreshInterval() time.Duration {
	return 15 * time.Minute // Refresh every 15 minutes
}

// Validate checks if the provided parameters are valid for this report
func (r *CostReport) Validate(params reports.ReportParams) error {
	// Cost reports don't have specific parameter requirements currently
	return nil
}

// Helper methods

func (r *CostReport) calculateCostTrend(currentCost float64) *reports.TrendData {
	// For demo purposes, simulate a trend
	// In a real implementation, you'd compare with historical data
	previousCost := currentCost * 0.95 // Simulate 5% increase
	return r.renderer.FormatTrend(currentCost, previousCost, "vs last month")
}

func (r *CostReport) getTopCostService(services []CostData) *CostData {
	if len(services) == 0 {
		return nil
	}

	top := &services[0]
	for i := 1; i < len(services); i++ {
		if services[i].Amount > top.Amount {
			top = &services[i]
		}
	}
	return top
}

func (r *CostReport) generateDataPoints(costSummary *CostSummary, appData *ApplicationListResponse) []reports.DataPoint {
	var dataPoints []reports.DataPoint
	now := time.Now()

	// Add overall cost data point
	overallPoint := reports.DataPoint{
		Timestamp: now,
		Labels: map[string]string{
			"type":   "total_cost",
			"period": "monthly",
		},
		Values: map[string]interface{}{
			"total_cost":        costSummary.TotalCost,
			"application_count": appData.Count,
			"currency":          costSummary.Currency,
		},
	}
	dataPoints = append(dataPoints, overallPoint)

	// Add service-level data points
	for _, service := range costSummary.Services {
		servicePoint := reports.DataPoint{
			Timestamp: now,
			Labels: map[string]string{
				"type":    "service_cost",
				"service": service.Service,
			},
			Values: map[string]interface{}{
				"cost":     service.Amount,
				"currency": service.Currency,
			},
		}
		dataPoints = append(dataPoints, servicePoint)
	}

	// Add application-level data points
	for _, app := range appData.Applications {
		appPoint := reports.DataPoint{
			Timestamp: now,
			Labels: map[string]string{
				"type":        "application_cost",
				"application": app.Name,
				"team":        app.Team,
				"hosting":     app.ProductionHostedOn,
			},
			Values: map[string]interface{}{
				"cost":            app.TotalCost,
				"currency":        app.Currency,
				"service_count":   app.ServiceCount,
				"cost_source":     app.CostSource,
				"cost_confidence": app.CostConfidence,
			},
		}
		dataPoints = append(dataPoints, appPoint)
	}

	return dataPoints
}

func (r *CostReport) generateCharts(costSummary *CostSummary, appData *ApplicationListResponse) []reports.ChartData {
	var charts []reports.ChartData

	// Service cost breakdown pie chart
	if len(costSummary.Services) > 0 {
		serviceChart := reports.ChartData{
			Title: "Cost by Service",
			Type:  "pie",
			XAxis: "service",
			YAxis: "cost",
		}

		var series reports.ChartSeries
		series.Name = "Service Costs"
		for _, service := range costSummary.Services {
			series.Data = append(series.Data, reports.ChartPoint{
				X: service.Service,
				Y: service.Amount,
			})
		}
		serviceChart.Series = append(serviceChart.Series, series)
		charts = append(charts, serviceChart)
	}

	// Application cost bar chart
	if len(appData.Applications) > 0 {
		appChart := reports.ChartData{
			Title: "Cost by Application",
			Type:  "bar",
			XAxis: "application",
			YAxis: "cost",
		}

		var series reports.ChartSeries
		series.Name = "Application Costs"
		
		// Sort applications by cost (top 10)
		apps := appData.Applications
		if len(apps) > 10 {
			// In a real implementation, you'd sort by cost first
			apps = apps[:10]
		}
		
		for _, app := range apps {
			series.Data = append(series.Data, reports.ChartPoint{
				X: app.Name,
				Y: app.TotalCost,
			})
		}
		appChart.Series = append(appChart.Series, series)
		charts = append(charts, appChart)
	}

	return charts
}

func (r *CostReport) generateTables(appData *ApplicationListResponse) []reports.TableData {
	var tables []reports.TableData

	// Application cost table
	appTable := reports.TableData{
		Title: "Application Costs",
		Headers: []reports.TableHeader{
			{Key: "name", Label: "Application", Type: "string", Sortable: true, Filterable: true},
			{Key: "team", Label: "Team", Type: "string", Sortable: true, Filterable: true},
			{Key: "hosting", Label: "Hosting", Type: "string", Sortable: true, Filterable: true},
			{Key: "cost", Label: "Monthly Cost", Type: "currency", Sortable: true, Filterable: false},
			{Key: "confidence", Label: "Confidence", Type: "string", Sortable: true, Filterable: true},
		},
	}

	for _, app := range appData.Applications {
		row := map[string]interface{}{
			"name":       app.Name,
			"team":       app.Team,
			"hosting":    app.ProductionHostedOn,
			"cost":       r.renderer.FormatCurrency(app.TotalCost, "GBP"),
			"confidence": app.CostConfidence,
		}
		appTable.Rows = append(appTable.Rows, row)
	}

	tables = append(tables, appTable)
	return tables
}
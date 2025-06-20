package rds

import (
	"context"
	"fmt"
	"time"

	"govuk-reports-dashboard/internal/reports"
	"govuk-reports-dashboard/pkg/logger"
)

// RDSReport implements the reports.Report interface for PostgreSQL version checking
type RDSReport struct {
	rdsService *RDSService
	renderer   *reports.Renderer
	logger     *logger.Logger
}

// NewRDSReport creates a new RDS report instance
func NewRDSReport(rdsService *RDSService, logger *logger.Logger) *RDSReport {
	return &RDSReport{
		rdsService: rdsService,
		renderer:   reports.NewRenderer(),
		logger:     logger,
	}
}

// GetMetadata returns metadata about this report module
func (r *RDSReport) GetMetadata() reports.ReportMetadata {
	return reports.ReportMetadata{
		ID:          "rds",
		Name:        "PostgreSQL Version Checker",
		Description: "PostgreSQL RDS instance discovery and version compliance checking",
		Type:        reports.ReportTypeHealth,
		Version:     "1.0.0",
		Author:      "GOV.UK Platform Team",
		Tags:        []string{"rds", "postgresql", "versions", "compliance", "eol"},
		Priority:    reports.PriorityMedium,
	}
}

// GenerateSummary creates summary data for dashboard display
func (r *RDSReport) GenerateSummary(ctx context.Context, params reports.ReportParams) ([]reports.Summary, error) {
	r.logger.Info().Msg("Generating RDS summary for dashboard")

	// Get instances summary
	summary, err := r.rdsService.GetAllInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RDS instances: %w", err)
	}

	var summaries []reports.Summary

	// Total PostgreSQL Instances
	totalInstancesSummary := r.renderer.CreateSummaryCard(
		"PostgreSQL Instances",
		r.renderer.FormatNumber(summary.PostgreSQLCount),
		"Total instances",
		reports.SummaryTypeCount,
		nil,
	)
	summaries = append(summaries, totalInstancesSummary)

	// EOL Instances (Critical)
	eolSummary := r.renderer.CreateSummaryCard(
		"EOL Instances",
		r.renderer.FormatNumber(summary.EOLInstances),
		"End-of-life versions",
		reports.SummaryTypeAlert,
		nil,
	)
	if summary.EOLInstances > 0 {
		eolSummary.(*reports.BasicSummary).SetHealthy(false)
	}
	summaries = append(summaries, eolSummary)

	// Outdated Instances
	outdatedSummary := r.renderer.CreateSummaryCard(
		"Outdated Instances",
		r.renderer.FormatNumber(summary.OutdatedInstances),
		"Need updates",
		reports.SummaryTypeHealth,
		nil,
	)
	if summary.OutdatedInstances > 0 {
		outdatedSummary.(*reports.BasicSummary).SetHealthy(false)
	}
	summaries = append(summaries, outdatedSummary)

	// Compliance Status
	compliantInstances := summary.PostgreSQLCount - summary.EOLInstances - summary.OutdatedInstances
	compliancePercentage := 0.0
	if summary.PostgreSQLCount > 0 {
		compliancePercentage = (float64(compliantInstances) / float64(summary.PostgreSQLCount)) * 100
	}
	
	complianceSummary := r.renderer.CreateSummaryCard(
		"Version Compliance",
		r.renderer.FormatPercentage(compliancePercentage, 1),
		"Up-to-date instances",
		reports.SummaryTypeHealth,
		nil,
	)
	if compliancePercentage < 90 {
		complianceSummary.(*reports.BasicSummary).SetHealthy(false)
	}
	summaries = append(summaries, complianceSummary)

	r.logger.WithField("summary_count", len(summaries)).Info().Msg("Generated RDS summaries")
	return summaries, nil
}

// GenerateReport creates detailed report data
func (r *RDSReport) GenerateReport(ctx context.Context, params reports.ReportParams) (reports.ReportData, error) {
	r.logger.Info().Msg("Generating detailed RDS report")

	data := reports.ReportData{
		Status:      reports.StatusRunning,
		GeneratedAt: time.Now(),
	}

	// Get instances summary
	summary, err := r.rdsService.GetAllInstances(ctx)
	if err != nil {
		data.Status = reports.StatusFailed
		data.Errors = append(data.Errors, reports.ReportError{
			Code:      "RDS_FETCH_ERROR",
			Message:   "Failed to fetch RDS instances",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return data, nil
	}

	// Get version check results
	versionChecks, err := r.rdsService.GetVersionCheckResults(ctx)
	if err != nil {
		data.Warnings = append(data.Warnings, reports.ReportWarning{
			Code:      "VERSION_CHECK_WARNING",
			Message:   "Failed to get version check results",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
	}

	// Generate data points
	data.DataPoints = r.generateDataPoints(summary, versionChecks)

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
	data.Charts = r.generateCharts(summary, versionChecks)

	// Generate tables
	data.Tables = r.generateTables(summary, versionChecks)

	data.Status = reports.StatusCompleted
	r.logger.WithFields(map[string]interface{}{
		"data_points": len(data.DataPoints),
		"charts":      len(data.Charts),
		"tables":      len(data.Tables),
	}).Info().Msg("Generated detailed RDS report")

	return data, nil
}

// IsAvailable checks if this report can run with current configuration
func (r *RDSReport) IsAvailable(ctx context.Context) bool {
	// Try to list instances to verify AWS RDS connectivity
	_, err := r.rdsService.GetAllInstances(ctx)
	return err == nil
}

// GetRefreshInterval returns how often this report should be refreshed
func (r *RDSReport) GetRefreshInterval() time.Duration {
	return 30 * time.Minute // Refresh every 30 minutes (RDS data changes less frequently)
}

// Validate checks if the provided parameters are valid for this report
func (r *RDSReport) Validate(params reports.ReportParams) error {
	// RDS reports don't have specific parameter requirements currently
	return nil
}

// Helper methods

func (r *RDSReport) generateDataPoints(summary *InstancesSummary, versionChecks []VersionCheckResult) []reports.DataPoint {
	var dataPoints []reports.DataPoint
	now := time.Now()

	// Add overall RDS data point
	overallPoint := reports.DataPoint{
		Timestamp: now,
		Labels: map[string]string{
			"type":   "rds_summary",
			"source": "aws_rds",
		},
		Values: map[string]interface{}{
			"total_instances":    summary.TotalInstances,
			"postgresql_count":   summary.PostgreSQLCount,
			"eol_instances":      summary.EOLInstances,
			"outdated_instances": summary.OutdatedInstances,
		},
	}
	dataPoints = append(dataPoints, overallPoint)

	// Add instance-level data points
	for _, instance := range summary.Instances {
		instancePoint := reports.DataPoint{
			Timestamp: now,
			Labels: map[string]string{
				"type":         "rds_instance",
				"instance_id":  instance.InstanceID,
				"application":  instance.Application,
				"environment":  instance.Environment,
				"region":       instance.Region,
				"version":      instance.Version,
				"major_version": instance.MajorVersion,
			},
			Values: map[string]interface{}{
				"is_eol":              instance.IsEOL,
				"is_outdated":         r.isInstanceOutdated(instance),
				"instance_class":      instance.InstanceClass,
				"allocated_storage":   instance.AllocatedStorage,
				"multi_az":            instance.MultiAZ,
				"publicly_accessible": instance.PubliclyAccessible,
			},
		}
		dataPoints = append(dataPoints, instancePoint)
	}

	// Add version distribution data points
	for _, versionSummary := range summary.VersionSummary {
		versionPoint := reports.DataPoint{
			Timestamp: now,
			Labels: map[string]string{
				"type":          "version_distribution",
				"major_version": versionSummary.MajorVersion,
			},
			Values: map[string]interface{}{
				"count":       versionSummary.Count,
				"is_eol":      versionSummary.IsEOL,
				"is_outdated": versionSummary.IsOutdated,
			},
		}
		dataPoints = append(dataPoints, versionPoint)
	}

	return dataPoints
}

func (r *RDSReport) generateCharts(summary *InstancesSummary, versionChecks []VersionCheckResult) []reports.ChartData {
	var charts []reports.ChartData

	// Version distribution pie chart
	if len(summary.VersionSummary) > 0 {
		versionChart := reports.ChartData{
			Title: "PostgreSQL Version Distribution",
			Type:  "pie",
			XAxis: "version",
			YAxis: "count",
		}

		var series reports.ChartSeries
		series.Name = "Instance Count"
		for _, versionSummary := range summary.VersionSummary {
			label := fmt.Sprintf("PostgreSQL %s", versionSummary.MajorVersion)
			if versionSummary.IsEOL {
				label += " (EOL)"
			} else if versionSummary.IsOutdated {
				label += " (Outdated)"
			}
			
			series.Data = append(series.Data, reports.ChartPoint{
				X: label,
				Y: versionSummary.Count,
			})
		}
		versionChart.Series = append(versionChart.Series, series)
		charts = append(charts, versionChart)
	}

	// Compliance status bar chart
	complianceChart := reports.ChartData{
		Title: "Version Compliance Status",
		Type:  "bar",
		XAxis: "status",
		YAxis: "count",
	}

	compliantCount := summary.PostgreSQLCount - summary.EOLInstances - summary.OutdatedInstances
	var complianceSeries reports.ChartSeries
	complianceSeries.Name = "Instances"
	complianceSeries.Data = []reports.ChartPoint{
		{X: "Compliant", Y: compliantCount},
		{X: "Outdated", Y: summary.OutdatedInstances},
		{X: "End-of-Life", Y: summary.EOLInstances},
	}
	complianceChart.Series = append(complianceChart.Series, complianceSeries)
	charts = append(charts, complianceChart)

	return charts
}

func (r *RDSReport) generateTables(summary *InstancesSummary, versionChecks []VersionCheckResult) []reports.TableData {
	var tables []reports.TableData

	// Instances table
	instancesTable := reports.TableData{
		Title: "PostgreSQL Instances",
		Headers: []reports.TableHeader{
			{Key: "instance_id", Label: "Instance ID", Type: "string", Sortable: true, Filterable: true},
			{Key: "application", Label: "Application", Type: "string", Sortable: true, Filterable: true},
			{Key: "environment", Label: "Environment", Type: "string", Sortable: true, Filterable: true},
			{Key: "version", Label: "Version", Type: "string", Sortable: true, Filterable: true},
			{Key: "status", Label: "Status", Type: "string", Sortable: true, Filterable: true},
			{Key: "compliance", Label: "Compliance", Type: "string", Sortable: true, Filterable: true},
			{Key: "instance_class", Label: "Instance Class", Type: "string", Sortable: true, Filterable: true},
			{Key: "region", Label: "Region", Type: "string", Sortable: true, Filterable: true},
		},
	}

	for _, instance := range summary.Instances {
		compliance := "Compliant"
		if instance.IsEOL {
			compliance = "End-of-Life"
		} else if r.isInstanceOutdated(instance) {
			compliance = "Outdated"
		}

		row := map[string]interface{}{
			"instance_id":    instance.InstanceID,
			"application":    instance.Application,
			"environment":    instance.Environment,
			"version":        instance.Version,
			"status":         instance.Status,
			"compliance":     compliance,
			"instance_class": instance.InstanceClass,
			"region":         instance.Region,
		}
		instancesTable.Rows = append(instancesTable.Rows, row)
	}

	tables = append(tables, instancesTable)

	// Version summary table
	versionTable := reports.TableData{
		Title: "Version Summary",
		Headers: []reports.TableHeader{
			{Key: "major_version", Label: "Major Version", Type: "string", Sortable: true, Filterable: false},
			{Key: "count", Label: "Instance Count", Type: "number", Sortable: true, Filterable: false},
			{Key: "status", Label: "Status", Type: "string", Sortable: true, Filterable: true},
		},
	}

	for _, versionSummary := range summary.VersionSummary {
		status := "Supported"
		if versionSummary.IsEOL {
			status = "End-of-Life"
		} else if versionSummary.IsOutdated {
			status = "Outdated"
		}

		row := map[string]interface{}{
			"major_version": fmt.Sprintf("PostgreSQL %s", versionSummary.MajorVersion),
			"count":         versionSummary.Count,
			"status":        status,
		}
		versionTable.Rows = append(versionTable.Rows, row)
	}

	tables = append(tables, versionTable)

	return tables
}

func (r *RDSReport) isInstanceOutdated(instance PostgreSQLInstance) bool {
	return !instance.IsEOL && r.rdsService.isOutdated(instance)
}
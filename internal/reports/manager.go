package reports

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"govuk-reports-dashboard/pkg/logger"
)

// Manager handles registration and execution of report modules
type Manager struct {
	reports map[string]Report
	cache   *ReportCache
	logger  *logger.Logger
	mu      sync.RWMutex
}

// NewManager creates a new report manager
func NewManager(logger *logger.Logger) *Manager {
	return &Manager{
		reports: make(map[string]Report),
		cache:   NewReportCache(),
		logger:  logger,
	}
}

// Register adds a new report module to the manager
func (m *Manager) Register(report Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadata := report.GetMetadata()
	if metadata.ID == "" {
		return fmt.Errorf("report ID cannot be empty")
	}

	if _, exists := m.reports[metadata.ID]; exists {
		return fmt.Errorf("report with ID %s is already registered", metadata.ID)
	}

	m.reports[metadata.ID] = report
	m.logger.WithField("report_id", metadata.ID).Info().Msg("Report module registered")
	
	return nil
}

// Unregister removes a report module from the manager
func (m *Manager) Unregister(reportID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.reports[reportID]; !exists {
		return fmt.Errorf("report with ID %s is not registered", reportID)
	}

	delete(m.reports, reportID)
	m.cache.Invalidate(reportID)
	m.logger.WithField("report_id", reportID).Info().Msg("Report module unregistered")
	
	return nil
}

// GetReport retrieves a specific report by ID
func (m *Manager) GetReport(reportID string) (Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report, exists := m.reports[reportID]
	if !exists {
		return nil, fmt.Errorf("report with ID %s not found", reportID)
	}

	return report, nil
}

// ListReports returns all registered reports
func (m *Manager) ListReports() []ReportMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []ReportMetadata
	for _, report := range m.reports {
		reports = append(reports, report.GetMetadata())
	}

	// Sort by priority (highest first), then by name
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Priority != reports[j].Priority {
			return reports[i].Priority > reports[j].Priority
		}
		return reports[i].Name < reports[j].Name
	})

	return reports
}

// GetAvailableReports returns only reports that are currently available
func (m *Manager) GetAvailableReports(ctx context.Context) []ReportMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var available []ReportMetadata
	for _, report := range m.reports {
		if report.IsAvailable(ctx) {
			available = append(available, report.GetMetadata())
		}
	}

	// Sort by priority (highest first), then by name
	sort.Slice(available, func(i, j int) bool {
		if available[i].Priority != available[j].Priority {
			return available[i].Priority > available[j].Priority
		}
		return available[i].Name < available[j].Name
	})

	return available
}

// GenerateSummary generates summary data for all available reports
func (m *Manager) GenerateSummary(ctx context.Context, params ReportParams) ([]Summary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allSummaries []Summary
	var errors []string

	for _, report := range m.reports {
		if !report.IsAvailable(ctx) {
			continue
		}

		metadata := report.GetMetadata()
		
		// Check cache first
		if !params.ForceRefresh && params.UseCache {
			if cached := m.cache.GetSummary(metadata.ID, params); cached != nil {
				allSummaries = append(allSummaries, cached...)
				continue
			}
		}

		// Generate fresh summary
		summaries, err := report.GenerateSummary(ctx, params)
		if err != nil {
			m.logger.WithFields(map[string]interface{}{
				"report_id": metadata.ID,
				"error":     err.Error(),
			}).Error().Msg("Failed to generate summary")
			errors = append(errors, fmt.Sprintf("%s: %v", metadata.Name, err))
			continue
		}

		// Cache the result
		if params.UseCache {
			m.cache.SetSummary(metadata.ID, params, summaries, report.GetRefreshInterval())
		}

		allSummaries = append(allSummaries, summaries...)
	}

	if len(errors) > 0 && len(allSummaries) == 0 {
		return nil, fmt.Errorf("all reports failed: %v", errors)
	}

	return allSummaries, nil
}

// GenerateReport generates a detailed report for a specific report module
func (m *Manager) GenerateReport(ctx context.Context, reportID string, params ReportParams) (ReportData, error) {
	report, err := m.GetReport(reportID)
	if err != nil {
		return ReportData{}, err
	}

	if !report.IsAvailable(ctx) {
		return ReportData{}, fmt.Errorf("report %s is not currently available", reportID)
	}

	// Validate parameters
	if err := report.Validate(params); err != nil {
		return ReportData{}, fmt.Errorf("invalid parameters: %w", err)
	}

	metadata := report.GetMetadata()

	// Check cache first
	if !params.ForceRefresh && params.UseCache {
		if cached := m.cache.GetReport(reportID, params); cached != nil {
			return *cached, nil
		}
	}

	// Generate fresh report
	m.logger.WithField("report_id", reportID).Info().Msg("Generating report")
	
	data, err := report.GenerateReport(ctx, params)
	if err != nil {
		m.logger.WithFields(map[string]interface{}{
			"report_id": reportID,
			"error":     err.Error(),
		}).Error().Msg("Failed to generate report")
		return ReportData{}, fmt.Errorf("failed to generate report: %w", err)
	}

	// Ensure metadata is set
	data.Metadata = metadata
	data.GeneratedAt = time.Now()

	// Cache the result
	if params.UseCache {
		m.cache.SetReport(reportID, params, &data, report.GetRefreshInterval())
	}

	m.logger.WithFields(map[string]interface{}{
		"report_id":    reportID,
		"data_points":  len(data.DataPoints),
		"charts":       len(data.Charts),
		"tables":       len(data.Tables),
	}).Info().Msg("Report generated successfully")

	return data, nil
}

// GetReportsByType returns all reports of a specific type
func (m *Manager) GetReportsByType(reportType ReportType) []ReportMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []ReportMetadata
	for _, report := range m.reports {
		metadata := report.GetMetadata()
		if metadata.Type == reportType {
			filtered = append(filtered, metadata)
		}
	}

	// Sort by priority (highest first), then by name
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Priority != filtered[j].Priority {
			return filtered[i].Priority > filtered[j].Priority
		}
		return filtered[i].Name < filtered[j].Name
	})

	return filtered
}

// RefreshCache invalidates all cached data
func (m *Manager) RefreshCache() {
	m.cache.Clear()
	m.logger.Info().Msg("Report cache cleared")
}

// GetCacheStats returns cache statistics
func (m *Manager) GetCacheStats() CacheStats {
	return m.cache.GetStats()
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info().Msg("Shutting down report manager")
	m.cache.Clear()
	return nil
}
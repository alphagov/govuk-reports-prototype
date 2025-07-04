package elasticache

import (
	"context"
	"time"

	"govuk-reports-dashboard/internal/reports"
	"govuk-reports-dashboard/pkg/logger"
)

type ElastiCacheReport struct {
	elastiCacheService *ElastiCacheService
	renderer           *reports.Renderer
	logger             *logger.Logger
}

func NewElastiCacheReport(elastiCacheService *ElastiCacheService, logger *logger.Logger) *ElastiCacheReport {
	return &ElastiCacheReport{
		elastiCacheService: elastiCacheService,
		renderer:           reports.NewRenderer(),
		logger:             logger,
	}
}

func (e *ElastiCacheReport) GetMetadata() reports.ReportMetadata {
	return reports.ReportMetadata{
		ID:          "elasticache",
		Name:        "ElastiCache patching report",
		Description: "ElastiCache discovery and patch compliance checking",
		Type:        reports.ReportTypeHealth,
		Version:     "1.0.0",
		Author:      "GOV.UK Platform Team",
		Tags:        []string{"elasticache", "redis", "valkey", "memcached", "versions", "patching"},
		Priority:    reports.PriorityMedium,
	}
}

func (e *ElastiCacheReport) GenerateSummary(ctx context.Context, params reports.ReportParams) ([]reports.Summary, error) {
	// TODO
	return []reports.Summary{}, nil
}

func (e *ElastiCacheReport) GenerateReport(ctx context.Context, params reports.ReportParams) (reports.ReportData, error) {
	// TODO
	return reports.ReportData{}, nil
}

func (e *ElastiCacheReport) IsAvailable(ctx context.Context) bool {
	_, err := e.elastiCacheService.GetServerlessCaches(ctx)
	return err == nil
}

// GetRefreshInterval returns how often this report should be refreshed
func (e *ElastiCacheReport) GetRefreshInterval() time.Duration {
	return 30 * time.Minute // Refresh every 30 minutes (ElastiCache data changes less frequently)
}

// Validate checks if the provided parameters are valid for this report
func (e *ElastiCacheReport) Validate(params reports.ReportParams) error {
	// ElastiCache reports don't have specific parameter requirements currently
	return nil
}

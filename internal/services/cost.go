package services

import (
	"time"

	"govuk-reports-dashboard/internal/models"
	"govuk-reports-dashboard/pkg/aws"
	"govuk-reports-dashboard/pkg/govuk"

	"govuk-reports-dashboard/pkg/logger"
)

type CostService struct {
	awsClient   *aws.Client
	govukClient *govuk.Client
	logger      *logger.Logger
}

func NewCostService(awsClient *aws.Client, govukClient *govuk.Client, log *logger.Logger) *CostService {
	return &CostService{
		awsClient:   awsClient,
		govukClient: govukClient,
		logger:      log,
	}
}

func (s *CostService) GetCostSummary() (*models.CostSummary, error) {
	s.logger.Info().Msg("Fetching AWS cost data")

	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Error().Msg("Failed to fetch AWS cost data")
		return nil, err
	}

	summary := &models.CostSummary{
		TotalCost:   calculateTotal(costData),
		Currency:    "GBP",
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		Services:    costData,
		LastUpdated: time.Now(),
	}

	return summary, nil
}

func calculateTotal(costs []models.CostData) float64 {
	total := 0.0
	for _, cost := range costs {
		total += cost.Amount
	}
	return total
}
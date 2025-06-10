package services

import (
	"time"

	"govuk-cost-dashboard/internal/models"
	"govuk-cost-dashboard/pkg/aws"
	"govuk-cost-dashboard/pkg/govuk"

	"github.com/sirupsen/logrus"
)

type CostService struct {
	awsClient   *aws.Client
	govukClient *govuk.Client
	logger      *logrus.Logger
}

func NewCostService(awsClient *aws.Client, govukClient *govuk.Client, logger *logrus.Logger) *CostService {
	return &CostService{
		awsClient:   awsClient,
		govukClient: govukClient,
		logger:      logger,
	}
}

func (s *CostService) GetCostSummary() (*models.CostSummary, error) {
	s.logger.Info("Fetching AWS cost data")

	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Error("Failed to fetch AWS cost data")
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
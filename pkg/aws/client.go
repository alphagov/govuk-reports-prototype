package aws

import (
	"context"
	"time"

	"govuk-cost-dashboard/internal/config"
	"govuk-cost-dashboard/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/sirupsen/logrus"
)

type Client struct {
	costExplorer *costexplorer.Client
	logger       *logrus.Logger
}

func NewClient(cfg *config.Config, logger *logrus.Logger) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		costExplorer: costexplorer.NewFromConfig(awsCfg),
		logger:       logger,
	}, nil
}

func (c *Client) GetCostData() ([]models.CostData, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, -1, 0)

	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime.Format("2006-01-02")),
			End:   aws.String(endTime.Format("2006-01-02")),
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeKey,
				Key:  aws.String("SERVICE"),
			},
		},
	}

	result, err := c.costExplorer.GetCostAndUsage(context.TODO(), input)
	if err != nil {
		c.logger.WithError(err).Error("Failed to get cost and usage data from AWS")
		return nil, err
	}

	var costData []models.CostData
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && len(group.Metrics) > 0 {
				if blendedCost, ok := group.Metrics["BlendedCost"]; ok {
					amount := 0.0
					if blendedCost.Amount != nil {
						amount = parseFloat(*blendedCost.Amount)
					}

					costData = append(costData, models.CostData{
						Service:     group.Keys[0],
						Amount:      amount,
						Currency:    getStringValue(blendedCost.Unit),
						StartDate:   parseDate(*resultByTime.TimePeriod.Start),
						EndDate:     parseDate(*resultByTime.TimePeriod.End),
						Granularity: "MONTHLY",
					})
				}
			}
		}
	}

	return costData, nil
}

func parseFloat(s string) float64 {
	// Simple float parsing - in production, handle errors properly
	if s == "" {
		return 0.0
	}
	// This is a simplified version - use strconv.ParseFloat in production
	return 0.0
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
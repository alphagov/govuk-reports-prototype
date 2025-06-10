package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"govuk-cost-dashboard/internal/config"
	"govuk-cost-dashboard/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/sirupsen/logrus"
)

type Client struct {
	costExplorer *costexplorer.Client
	logger       *logrus.Logger
}

// mfaTokenProvider prompts for MFA token input or reads from environment
func mfaTokenProvider() (string, error) {
	// First check if MFA token is provided via environment variable
	if token := os.Getenv("AWS_MFA_TOKEN"); token != "" {
		return token, nil
	}

	// If not in environment, prompt user for input
	fmt.Print("Enter MFA token: ")
	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token), nil
}

func NewClient(cfg *config.Config, logger *logrus.Logger) (*Client, error) {
	var configOptions []func(*awsconfig.LoadOptions) error
	
	// Set region
	configOptions = append(configOptions, awsconfig.WithRegion(cfg.AWS.Region))
	
	// Configure MFA token provider for assume role operations
	configOptions = append(configOptions, awsconfig.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
		options.TokenProvider = mfaTokenProvider
	}))
	
	// Use AWS profile if specified
	if cfg.AWS.Profile != "" {
		logger.WithField("profile", cfg.AWS.Profile).Info("Using AWS profile")
		configOptions = append(configOptions, awsconfig.WithSharedConfigProfile(cfg.AWS.Profile))
	}
	
	// If explicit credentials are provided, use them
	if cfg.AWS.AccessKeyID != "" && cfg.AWS.SecretAccessKey != "" {
		logger.Info("Using explicit AWS credentials")
		credentials := aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AWS.AccessKeyID,
				SecretAccessKey: cfg.AWS.SecretAccessKey,
				SessionToken:    cfg.AWS.SessionToken,
				Source:          "Environment",
			}, nil
		})
		configOptions = append(configOptions, awsconfig.WithCredentialsProvider(credentials))
	} else {
		logger.Info("Using AWS default credential chain (profile, environment, EC2 role)")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), configOptions...)
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
				Type: types.GroupDefinitionTypeDimension,
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
	if s == "" {
		return 0.0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return f
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
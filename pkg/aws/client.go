package aws

import (
	"bufio"
	"context"
	"fmt"
	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/pkg/logger"
	"govuk-reports-dashboard/pkg/common"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type Client struct {
	costExplorer *costexplorer.Client
	logger       *logger.Logger
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

func NewClient(cfg *config.Config, log *logger.Logger) (*Client, error) {
	var configOptions []func(*awsconfig.LoadOptions) error

	// Set region
	configOptions = append(configOptions, awsconfig.WithRegion(cfg.AWS.Region))

	// Configure MFA token provider for assume role operations
	configOptions = append(configOptions, awsconfig.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
		options.TokenProvider = mfaTokenProvider
	}))

	// Use AWS profile if specified
	if cfg.AWS.Profile != "" {
		log.WithField("profile", cfg.AWS.Profile).Info().Msgf("Using AWS profile: %s", cfg.AWS.Profile)
		configOptions = append(configOptions, awsconfig.WithSharedConfigProfile(cfg.AWS.Profile))
	}

	// If explicit credentials are provided, use them
	if cfg.AWS.AccessKeyID != "" && cfg.AWS.SecretAccessKey != "" {
		log.Info().Msg("Using explicit AWS credentials")
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
		log.Info().Msg("Using AWS default credential chain (profile, environment, EC2 role)")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), configOptions...)
	if err != nil {
		return nil, err
	}

	return &Client{
		costExplorer: costexplorer.NewFromConfig(awsCfg),
		logger:       log,
	}, nil
}

func (c *Client) GetCostData() ([]common.CostData, error) {
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
		c.logger.WithError(err).Error().Msg("Failed to get cost and usage data from AWS")
		return nil, err
	}

	var costData []common.CostData
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && len(group.Metrics) > 0 {
				if blendedCost, ok := group.Metrics["BlendedCost"]; ok {
					amount := 0.0
					if blendedCost.Amount != nil {
						amount = parseFloat(*blendedCost.Amount)
					}

					costData = append(costData, common.CostData{
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

func (c *Client) GetCostDataBySystemTag() ([]common.CostData, error) {
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
				Type: types.GroupDefinitionTypeTag,
				Key:  aws.String("system"),
			},
		},
	}

	result, err := c.costExplorer.GetCostAndUsage(context.TODO(), input)
	if err != nil {
		c.logger.WithError(err).Error().Msg("Failed to get cost and usage data by system tag from AWS")
		return nil, err
	}

	var costData []common.CostData
	tagPrefix := getTagPrefix()

	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && len(group.Metrics) > 0 {
				tagValue := group.Keys[0]
				
				// Filter to only include tags matching the govuk-* pattern
				if !strings.HasPrefix(tagValue, tagPrefix) {
					continue
				}

				if blendedCost, ok := group.Metrics["BlendedCost"]; ok {
					amount := 0.0
					if blendedCost.Amount != nil {
						amount = parseFloat(*blendedCost.Amount)
					}

					costData = append(costData, common.CostData{
						Service:     tagValue, // Using tag value as service for consistency
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

func (c *Client) GetCostDataForApplication(appName string) ([]common.CostData, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, -1, 0)
	tagPrefix := getTagPrefix()
	targetTag := tagPrefix + appName

	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime.Format("2006-01-02")),
			End:   aws.String(endTime.Format("2006-01-02")),
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeTag,
				Key:  aws.String("system"),
			},
		},
		Filter: &types.Expression{
			Tags: &types.TagValues{
				Key:    aws.String("system"),
				Values: []string{targetTag},
			},
		},
	}

	result, err := c.costExplorer.GetCostAndUsage(context.TODO(), input)
	if err != nil {
		c.logger.WithError(err).Error().Msgf("Failed to get cost data for application %s from AWS", appName)
		return nil, err
	}

	var costData []common.CostData
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && len(group.Metrics) > 0 {
				if blendedCost, ok := group.Metrics["BlendedCost"]; ok {
					amount := 0.0
					if blendedCost.Amount != nil {
						amount = parseFloat(*blendedCost.Amount)
					}

					costData = append(costData, common.CostData{
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

func getTagPrefix() string {
	prefix := os.Getenv("GOVUK_APP_TAG_PREFIX")
	if prefix == "" {
		prefix = "govuk-"
	}
	return prefix
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


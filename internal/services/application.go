package services

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"govuk-cost-dashboard/internal/models"
	"govuk-cost-dashboard/pkg/aws"
	"govuk-cost-dashboard/pkg/govuk"

	"github.com/sirupsen/logrus"
)

type ApplicationService struct {
	awsClient   *aws.Client
	govukClient *govuk.Client
	logger      *logrus.Logger
}

func NewApplicationService(awsClient *aws.Client, govukClient *govuk.Client, logger *logrus.Logger) *ApplicationService {
	return &ApplicationService{
		awsClient:   awsClient,
		govukClient: govukClient,
		logger:      logger,
	}
}

// GetAllApplications returns all applications with cost summaries
func (s *ApplicationService) GetAllApplications(ctx context.Context) (*models.ApplicationListResponse, error) {
	s.logger.Info("Fetching all applications with cost data")

	// Get applications from GOV.UK API
	apps, err := s.govukClient.GetAllApplications(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to fetch applications")
		return nil, err
	}

	// Get cost data from AWS (for demo, we'll simulate costs)
	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to fetch AWS cost data, using simulated data")
		costData = s.generateSimulatedCosts(apps)
	}

	var applicationSummaries []models.ApplicationSummary
	var totalCost float64

	for _, app := range apps {
		// Calculate cost for this application (simplified mapping)
		appCost := s.calculateApplicationCost(app, costData)
		totalCost += appCost

		summary := models.ApplicationSummary{
			Name:               app.AppName,
			Shortname:          app.Shortname,
			Team:               app.Team,
			ProductionHostedOn: app.ProductionHostedOn,
			TotalCost:          appCost,
			Currency:           "GBP",
			ServiceCount:       s.estimateServiceCount(app),
			LastUpdated:        time.Now(),
			Links: models.Links{
				Self:      app.Links.Self,
				HTMLURL:   app.Links.HTMLURL,
				RepoURL:   app.Links.RepoURL,
				SentryURL: s.getSentryURL(app.Links.SentryURL),
			},
		}

		applicationSummaries = append(applicationSummaries, summary)
	}

	response := &models.ApplicationListResponse{
		Applications: applicationSummaries,
		TotalCost:    totalCost,
		Currency:     "GBP",
		Count:        len(applicationSummaries),
		LastUpdated:  time.Now(),
	}

	s.logger.WithFields(logrus.Fields{
		"app_count":  len(applicationSummaries),
		"total_cost": totalCost,
	}).Info("Successfully processed applications with costs")

	return response, nil
}

// GetApplicationByName returns detailed application data with cost breakdown
func (s *ApplicationService) GetApplicationByName(ctx context.Context, name string) (*models.ApplicationDetail, error) {
	s.logger.WithField("app_name", name).Info("Fetching application details")

	// Get specific application
	app, err := s.govukClient.GetApplicationByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get cost data
	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to fetch AWS cost data, using simulated data")
		costData = s.generateSimulatedCosts([]govuk.Application{*app})
	}

	// Generate service breakdown
	services := s.generateServiceBreakdown(*app, costData)
	totalCost := 0.0
	for _, service := range services {
		totalCost += service.Cost
	}

	detail := &models.ApplicationDetail{
		ApplicationSummary: models.ApplicationSummary{
			Name:               app.AppName,
			Shortname:          app.Shortname,
			Team:               app.Team,
			ProductionHostedOn: app.ProductionHostedOn,
			TotalCost:          totalCost,
			Currency:           "GBP",
			ServiceCount:       len(services),
			LastUpdated:        time.Now(),
			Links: models.Links{
				Self:      app.Links.Self,
				HTMLURL:   app.Links.HTMLURL,
				RepoURL:   app.Links.RepoURL,
				SentryURL: s.getSentryURL(app.Links.SentryURL),
			},
		},
		Services: services,
	}

	return detail, nil
}

// GetApplicationServices returns service cost breakdown for an application
func (s *ApplicationService) GetApplicationServices(ctx context.Context, name string) ([]models.ServiceCost, error) {
	s.logger.WithField("app_name", name).Info("Fetching application service costs")

	// Get specific application
	app, err := s.govukClient.GetApplicationByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get cost data
	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to fetch AWS cost data, using simulated data")
		costData = s.generateSimulatedCosts([]govuk.Application{*app})
	}

	services := s.generateServiceBreakdown(*app, costData)
	return services, nil
}

// Helper functions

func (s *ApplicationService) calculateApplicationCost(app govuk.Application, costData []models.CostData) float64 {
	// Simplified cost calculation based on hosting platform and team
	baseCost := 100.0 // Base cost in GBP

	// Adjust based on hosting platform
	switch strings.ToLower(app.ProductionHostedOn) {
	case "eks":
		baseCost *= 1.5 // EKS costs more
	case "heroku":
		baseCost *= 0.8 // Heroku costs less
	case "gcp":
		baseCost *= 1.2
	}

	// Add some randomness for demo purposes
	rand.Seed(time.Now().UnixNano())
	multiplier := 0.5 + rand.Float64()*2 // Random multiplier between 0.5 and 2.5
	
	return baseCost * multiplier
}

func (s *ApplicationService) estimateServiceCount(app govuk.Application) int {
	// Estimate based on hosting platform
	switch strings.ToLower(app.ProductionHostedOn) {
	case "eks":
		return 4 + rand.Intn(6) // 4-9 services
	case "heroku":
		return 2 + rand.Intn(4) // 2-5 services
	default:
		return 3 + rand.Intn(5) // 3-7 services
	}
}

func (s *ApplicationService) generateServiceBreakdown(app govuk.Application, costData []models.CostData) []models.ServiceCost {
	// Common AWS services used by GOV.UK applications
	serviceNames := []string{
		"Amazon EC2",
		"Amazon RDS",
		"Amazon S3",
		"Amazon CloudFront",
		"AWS Lambda",
		"Amazon ElastiCache",
		"Amazon ELB",
		"Amazon CloudWatch",
	}

	var services []models.ServiceCost
	totalCost := s.calculateApplicationCost(app, costData)
	now := time.Now()

	// Generate realistic service distribution
	serviceCount := s.estimateServiceCount(app)
	usedServices := serviceNames[:serviceCount]

	for i, serviceName := range usedServices {
		// Generate realistic cost distribution
		var percentage float64
		switch i {
		case 0: // Primary service (usually EC2 or EKS)
			percentage = 0.4 + rand.Float64()*0.3 // 40-70%
		case 1: // Secondary service (usually RDS)
			percentage = 0.15 + rand.Float64()*0.2 // 15-35%
		default: // Other services
			percentage = 0.02 + rand.Float64()*0.1 // 2-12%
		}

		cost := totalCost * percentage

		service := models.ServiceCost{
			ServiceName: serviceName,
			Cost:        cost,
			Currency:    "GBP",
			Percentage:  percentage * 100,
			StartDate:   now.AddDate(0, -1, 0),
			EndDate:     now,
		}

		services = append(services, service)
	}

	// Normalize percentages to ensure they add up to 100%
	s.normalizeServiceCosts(services, totalCost)

	return services
}

func (s *ApplicationService) normalizeServiceCosts(services []models.ServiceCost, totalCost float64) {
	if len(services) == 0 || totalCost == 0 {
		return
	}

	currentTotal := 0.0
	for _, service := range services {
		currentTotal += service.Cost
	}

	// Adjust costs to match the expected total
	if currentTotal > 0 {
		ratio := totalCost / currentTotal
		for i := range services {
			services[i].Cost *= ratio
			services[i].Percentage = (services[i].Cost / totalCost) * 100
		}
	}
}

func (s *ApplicationService) generateSimulatedCosts(apps []govuk.Application) []models.CostData {
	var costData []models.CostData
	now := time.Now()

	for _, app := range apps {
		cost := models.CostData{
			Service:     app.AppName,
			Amount:      s.calculateApplicationCost(app, nil),
			Currency:    "GBP",
			StartDate:   now.AddDate(0, -1, 0),
			EndDate:     now,
			Granularity: "MONTHLY",
		}
		costData = append(costData, cost)
	}

	return costData
}

func (s *ApplicationService) getSentryURL(sentryURL *string) string {
	if sentryURL != nil {
		return *sentryURL
	}
	return ""
}
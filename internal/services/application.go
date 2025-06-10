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
	// Try to find exact cost match from AWS data first
	if exactCost := s.findExactCostMatch(app, costData); exactCost > 0 {
		return exactCost
	}

	// Fall back to intelligent estimation
	return s.estimateApplicationCost(app, costData)
}

// findExactCostMatch attempts to find direct cost attribution
func (s *ApplicationService) findExactCostMatch(app govuk.Application, costData []models.CostData) float64 {
	// Try different naming convention matches
	possibleMatches := []string{
		app.AppName,                                    // Direct name match
		app.Shortname,                                  // Short name match
		strings.ReplaceAll(app.AppName, "-", "_"),      // Underscore version
		strings.ReplaceAll(app.AppName, "_", "-"),      // Hyphen version
		"govuk-" + app.AppName,                         // Prefixed version
		app.AppName + "-production",                    // Environment suffix
		app.AppName + "-prod",                          // Short env suffix
		strings.ToLower(app.Team) + "-" + app.AppName,  // Team prefix
	}

	for _, costItem := range costData {
		serviceName := strings.ToLower(costItem.Service)
		
		for _, match := range possibleMatches {
			if strings.Contains(serviceName, strings.ToLower(match)) ||
			   strings.Contains(strings.ToLower(match), serviceName) {
				s.logger.WithFields(logrus.Fields{
					"app":     app.AppName,
					"service": costItem.Service,
					"match":   match,
					"cost":    costItem.Amount,
				}).Debug("Found exact cost match")
				return costItem.Amount
			}
		}
	}

	return 0 // No exact match found
}

// estimateApplicationCost provides intelligent cost estimation
func (s *ApplicationService) estimateApplicationCost(app govuk.Application, costData []models.CostData) float64 {
	// Base cost calculation using multiple factors
	baseCost := s.calculateBaseCost(app)
	
	// Apply team-based scaling
	teamMultiplier := s.getTeamCostMultiplier(app.Team)
	
	// Apply hosting platform multiplier
	platformMultiplier := s.getHostingPlatformMultiplier(app.ProductionHostedOn)
	
	// Apply application complexity multiplier
	complexityMultiplier := s.getComplexityMultiplier(app)
	
	// Calculate final cost
	finalCost := baseCost * teamMultiplier * platformMultiplier * complexityMultiplier
	
	// Add deterministic variation based on app name (for consistency)
	hashMultiplier := s.getConsistentHashMultiplier(app.AppName)
	finalCost *= hashMultiplier
	
	s.logger.WithFields(logrus.Fields{
		"app":                  app.AppName,
		"base_cost":           baseCost,
		"team_multiplier":     teamMultiplier,
		"platform_multiplier": platformMultiplier,
		"complexity_multiplier": complexityMultiplier,
		"hash_multiplier":     hashMultiplier,
		"final_cost":          finalCost,
	}).Debug("Calculated estimated cost")
	
	return finalCost
}

// calculateBaseCost determines base cost based on application characteristics
func (s *ApplicationService) calculateBaseCost(app govuk.Application) float64 {
	baseCost := 150.0 // Starting base cost in GBP
	
	// Adjust based on application type (inferred from name patterns)
	if strings.Contains(strings.ToLower(app.AppName), "api") {
		baseCost *= 1.3 // APIs typically consume more resources
	}
	if strings.Contains(strings.ToLower(app.AppName), "frontend") {
		baseCost *= 0.8 // Frontends typically consume less
	}
	if strings.Contains(strings.ToLower(app.AppName), "publisher") {
		baseCost *= 1.2 // Publishing apps have moderate load
	}
	if strings.Contains(strings.ToLower(app.AppName), "admin") {
		baseCost *= 0.7 // Admin tools typically have lower usage
	}
	if strings.Contains(strings.ToLower(app.AppName), "search") {
		baseCost *= 1.5 // Search systems are resource intensive
	}
	
	return baseCost
}

// getTeamCostMultiplier returns cost multiplier based on team size and activity
func (s *ApplicationService) getTeamCostMultiplier(team string) float64 {
	teamMultipliers := map[string]float64{
		"GOV.UK Platform":    1.4, // Platform team manages high-traffic infrastructure
		"Publishing Platform": 1.3, // Core publishing infrastructure
		"Data Products":      1.2, // Data processing workloads
		"Content":           1.0, // Standard content applications
		"Design System":     0.8, // Lower traffic design tools
		"Developer docs":    0.7, // Documentation sites
		"Performance":       1.1, // Monitoring and analytics
		"Cyber Security":    1.0, // Security tooling
		"Specialist Publisher": 0.9, // Specialized publishing tools
	}
	
	if multiplier, exists := teamMultipliers[team]; exists {
		return multiplier
	}
	
	// Default multiplier for unknown teams
	return 1.0
}

// getHostingPlatformMultiplier returns multiplier based on hosting platform costs
func (s *ApplicationService) getHostingPlatformMultiplier(platform string) float64 {
	switch strings.ToLower(platform) {
	case "eks", "kubernetes":
		return 1.6 // EKS with all the managed services
	case "ec2":
		return 1.2 // Traditional EC2 instances
	case "heroku":
		return 0.9 // Heroku's efficiency for smaller apps
	case "gcp", "google cloud":
		return 1.3 // GCP services
	case "aws fargate":
		return 1.4 // Serverless containers
	case "aws lambda":
		return 0.6 // Pay-per-execution model
	case "cloudflare":
		return 0.3 // CDN and edge compute
	default:
		return 1.0 // Unknown platforms
	}
}

// getComplexityMultiplier estimates complexity based on application characteristics
func (s *ApplicationService) getComplexityMultiplier(app govuk.Application) float64 {
	complexity := 1.0
	
	appNameLower := strings.ToLower(app.AppName)
	
	// Database-heavy applications
	if strings.Contains(appNameLower, "db") || 
	   strings.Contains(appNameLower, "database") ||
	   strings.Contains(appNameLower, "store") {
		complexity *= 1.3
	}
	
	// Workflow/orchestration applications
	if strings.Contains(appNameLower, "workflow") ||
	   strings.Contains(appNameLower, "router") ||
	   strings.Contains(appNameLower, "gateway") {
		complexity *= 1.4
	}
	
	// Simple static sites or documentation
	if strings.Contains(appNameLower, "static") ||
	   strings.Contains(appNameLower, "docs") ||
	   strings.Contains(appNameLower, "guide") {
		complexity *= 0.6
	}
	
	// High-traffic public-facing applications
	if strings.Contains(appNameLower, "www") ||
	   strings.Contains(appNameLower, "frontend") ||
	   strings.Contains(appNameLower, "gov.uk") {
		complexity *= 1.2
	}
	
	return complexity
}

// getConsistentHashMultiplier provides deterministic variation based on app name
func (s *ApplicationService) getConsistentHashMultiplier(appName string) float64 {
	// Simple hash function for consistent results
	hash := 0
	for _, char := range appName {
		hash = hash*31 + int(char)
	}
	
	// Convert to a multiplier between 0.7 and 1.3
	normalizedHash := float64(hash%100) / 100.0
	return 0.7 + normalizedHash*0.6
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
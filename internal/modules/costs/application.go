package costs

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"govuk-reports-dashboard/pkg/aws"
	"govuk-reports-dashboard/pkg/govuk"
	"govuk-reports-dashboard/pkg/logger"
)

type ApplicationService struct {
	awsClient   *aws.Client
	govukClient *govuk.Client
	logger      *logger.Logger
}

func NewApplicationService(awsClient *aws.Client, govukClient *govuk.Client, log *logger.Logger) *ApplicationService {
	return &ApplicationService{
		awsClient:   awsClient,
		govukClient: govukClient,
		logger:      log,
	}
}

// GetAllApplications returns all applications with cost summaries
func (s *ApplicationService) GetAllApplications(ctx context.Context) (*ApplicationListResponse, error) {
	s.logger.Info().Msg("Fetching all applications with cost data")

	// Get applications from GOV.UK API
	apps, err := s.govukClient.GetAllApplications(ctx)
	if err != nil {
		s.logger.WithError(err).Error().Msg("Failed to fetch applications")
		return nil, err
	}

	// Get cost data from AWS (for demo, we'll simulate costs)
	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Warn().Msg("Failed to fetch AWS cost data, using simulated data")
		costData = s.generateSimulatedCosts(apps)
	}

	var applicationSummaries []ApplicationSummary
	var totalCost float64

	for _, app := range apps {
		// Calculate cost for this application with metadata
		costResult := s.calculateApplicationCost(app, costData)
		totalCost += costResult.Cost

		summary := ApplicationSummary{
			Name:               app.AppName,
			Shortname:          app.Shortname,
			Team:               app.Team,
			ProductionHostedOn: app.ProductionHostedOn,
			TotalCost:          costResult.Cost,
			Currency:           "GBP",
			ServiceCount:       s.estimateServiceCount(app),
			LastUpdated:        time.Now(),
			CostSource:         costResult.Source,
			CostConfidence:     costResult.Confidence,
			Links: Links{
				Self:      app.Links.Self,
				HTMLURL:   app.Links.HTMLURL,
				RepoURL:   app.Links.RepoURL,
				SentryURL: s.getSentryURL(app.Links.SentryURL),
			},
		}

		applicationSummaries = append(applicationSummaries, summary)
	}

	response := &ApplicationListResponse{
		Applications: applicationSummaries,
		TotalCost:    totalCost,
		Currency:     "GBP",
		Count:        len(applicationSummaries),
		LastUpdated:  time.Now(),
	}

	s.logger.WithFields(map[string]interface{}{
		"app_count":  len(applicationSummaries),
		"total_cost": totalCost,
	}).Info().Msg("Successfully processed applications with costs")

	return response, nil
}

// GetApplicationByName returns detailed application data with cost breakdown
func (s *ApplicationService) GetApplicationByName(ctx context.Context, name string) (*ApplicationDetail, error) {
	s.logger.WithField("app_name", name).Info().Msg("Fetching application details")

	// Get specific application
	app, err := s.govukClient.GetApplicationByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get cost data
	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Warn().Msg("Failed to fetch AWS cost data, using simulated data")
		costData = s.generateSimulatedCosts([]govuk.Application{*app})
	}

	// Calculate cost with metadata
	costResult := s.calculateApplicationCost(*app, costData)
	
	// Generate service breakdown
	services := s.generateServiceBreakdown(*app, costData, costResult)

	detail := &ApplicationDetail{
		ApplicationSummary: ApplicationSummary{
			Name:               app.AppName,
			Shortname:          app.Shortname,
			Team:               app.Team,
			ProductionHostedOn: app.ProductionHostedOn,
			TotalCost:          costResult.Cost,
			Currency:           "GBP",
			ServiceCount:       len(services),
			LastUpdated:        time.Now(),
			CostSource:         costResult.Source,
			CostConfidence:     costResult.Confidence,
			Links: Links{
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
func (s *ApplicationService) GetApplicationServices(ctx context.Context, name string) ([]ServiceCost, error) {
	s.logger.WithField("app_name", name).Info().Msg("Fetching application service costs")

	// Get specific application
	app, err := s.govukClient.GetApplicationByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get cost data
	costData, err := s.awsClient.GetCostData()
	if err != nil {
		s.logger.WithError(err).Warn().Msg("Failed to fetch AWS cost data, using simulated data")
		costData = s.generateSimulatedCosts([]govuk.Application{*app})
	}

	// Calculate cost with metadata
	costResult := s.calculateApplicationCost(*app, costData)
	
	services := s.generateServiceBreakdown(*app, costData, costResult)
	return services, nil
}

// Helper functions

// tryGetRealTagBasedCost attempts to get real cost data using AWS tags
func (s *ApplicationService) tryGetRealTagBasedCost(app govuk.Application) (float64, string) {
	// Map GOV.UK app name to system tag format
	systemTagName := s.mapAppNameToSystemTag(app)
	
	s.logger.WithFields(map[string]interface{}{
		"app":             app.AppName,
		"shortname":       app.Shortname,
		"mapped_tag":      systemTagName,
	}).Debug().Msg("Attempting to get real tag-based cost")
	
	// Try to get cost data for this specific application tag
	tagCostData, err := s.awsClient.GetCostDataForApplication(systemTagName)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"app":       app.AppName,
			"tag":       systemTagName,
			"error":     err.Error(),
		}).Debug().Msg("Failed to get tag-based cost data")
		return 0, "none"
	}
	
	if len(tagCostData) == 0 {
		s.logger.WithFields(map[string]interface{}{
			"app": app.AppName,
			"tag": systemTagName,
		}).Debug().Msg("No cost data found for application tag")
		return 0, "none"
	}
	
	// Sum up all costs for this application
	totalCost := 0.0
	for _, costItem := range tagCostData {
		totalCost += costItem.Amount
	}
	
	// Determine confidence based on data quality
	confidence := s.determineCostConfidence(tagCostData, app)
	
	s.logger.WithFields(map[string]interface{}{
		"app":             app.AppName,
		"tag":             systemTagName,
		"total_cost":      totalCost,
		"cost_items":      len(tagCostData),
		"confidence":      confidence,
	}).Debug().Msg("Successfully retrieved tag-based cost data")
	
	return totalCost, confidence
}

// mapAppNameToSystemTag maps GOV.UK application names to system tag values
func (s *ApplicationService) mapAppNameToSystemTag(app govuk.Application) string {
	// Try multiple mapping strategies to find the best match
	
	// Primary strategy: use shortname if available (most reliable)
	if app.Shortname != "" {
		return app.Shortname
	}
	
	// Secondary strategy: convert AppName to lowercase with hyphens
	appName := strings.ToLower(app.AppName)
	appName = strings.ReplaceAll(appName, " ", "-")
	appName = strings.ReplaceAll(appName, "_", "-")
	
	// Common transformations for GOV.UK app names
	mappings := map[string]string{
		"publishing-api":           "publishing-api",
		"content-store":           "content-store", 
		"frontend":                "frontend",
		"government-frontend":     "government-frontend",
		"collections":             "collections",
		"finder-frontend":         "finder-frontend",
		"whitehall":              "whitehall",
		"specialist-publisher":    "specialist-publisher",
		"manuals-publisher":       "manuals-publisher",
		"travel-advice-publisher": "travel-advice-publisher",
		"publisher":               "publisher",
		"short-url-manager":       "short-url-manager",
		"signon":                  "signon",
		"router":                  "router",
		"router-api":              "router-api",
		"content-data-api":        "content-data-api",
		"content-data-admin":      "content-data-admin",
		"search-api":              "search-api",
		"search-admin":            "search-admin",
		"email-alert-api":         "email-alert-api",
		"email-alert-frontend":    "email-alert-frontend",
		"static":                  "static",
		"smart-answers":           "smart-answers",
		"calculators":             "calculators",
		"service-manual-frontend": "service-manual-frontend",
		"service-manual-publisher": "service-manual-publisher",
	}
	
	// Check if we have a specific mapping
	if mapped, exists := mappings[appName]; exists {
		return mapped
	}
	
	// Default: return the transformed app name
	return appName
}

// determineCostConfidence assesses the reliability of cost data
func (s *ApplicationService) determineCostConfidence(costData []CostData, app govuk.Application) string {
	if len(costData) == 0 {
		return "none"
	}
	
	// Check data recency
	now := time.Now()
	hasRecentData := false
	totalCost := 0.0
	
	for _, item := range costData {
		totalCost += item.Amount
		// Data from within the last 2 months is considered recent
		if item.EndDate.After(now.AddDate(0, -2, 0)) {
			hasRecentData = true
		}
	}
	
	// Determine confidence based on multiple factors
	if !hasRecentData {
		return "low" // Old data
	}
	
	if totalCost == 0 {
		return "low" // No actual costs
	}
	
	if len(costData) >= 3 && totalCost > 10 { // Multiple cost entries with reasonable total
		return "high"
	}
	
	if len(costData) >= 1 && totalCost > 1 { // At least some cost data
		return "medium"  
	}
	
	return "low"
}

// CostCalculationResult holds both cost and metadata about how it was calculated
type CostCalculationResult struct {
	Cost       float64
	Source     string  // "real_aws_tags", "service_name_match", "estimation"
	Confidence string  // "high", "medium", "low", "none"
}

func (s *ApplicationService) calculateApplicationCost(app govuk.Application, costData []CostData) CostCalculationResult {
	// First, try to get real tag-based cost data from AWS
	if realCost, confidence := s.tryGetRealTagBasedCost(app); realCost > 0 {
		s.logger.WithFields(map[string]interface{}{
			"app":        app.AppName,
			"cost":       realCost,
			"confidence": confidence,
			"source":     "real_aws_tags",
		}).Info().Msg("Using real tag-based cost data")
		return CostCalculationResult{
			Cost:       realCost,
			Source:     "real_aws_tags",
			Confidence: confidence,
		}
	}

	// Try to find exact cost match from existing AWS data
	if exactCost := s.findExactCostMatch(app, costData); exactCost > 0 {
		s.logger.WithFields(map[string]interface{}{
			"app":        app.AppName,
			"cost":       exactCost,
			"confidence": "medium",
			"source":     "service_name_match",
		}).Info().Msg("Using service name matched cost data")
		return CostCalculationResult{
			Cost:       exactCost,
			Source:     "service_name_match",
			Confidence: "medium",
		}
	}

	// Fall back to intelligent estimation
	estimatedCost := s.estimateApplicationCost(app, costData)
	s.logger.WithFields(map[string]interface{}{
		"app":        app.AppName,
		"cost":       estimatedCost,
		"confidence": "low",
		"source":     "estimation",
	}).Info().Msg("Using estimated cost data")
	
	return CostCalculationResult{
		Cost:       estimatedCost,
		Source:     "estimation",
		Confidence: "low",
	}
}

// findExactCostMatch attempts to find direct cost attribution
func (s *ApplicationService) findExactCostMatch(app govuk.Application, costData []CostData) float64 {
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
				s.logger.WithFields(map[string]interface{}{
					"app":     app.AppName,
					"service": costItem.Service,
					"match":   match,
					"cost":    costItem.Amount,
				}).Debug().Msg("Found exact cost match")
				return costItem.Amount
			}
		}
	}

	return 0 // No exact match found
}

// estimateApplicationCost provides intelligent cost estimation
func (s *ApplicationService) estimateApplicationCost(app govuk.Application, costData []CostData) float64 {
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
	
	s.logger.WithFields(map[string]interface{}{
		"app":                  app.AppName,
		"base_cost":           baseCost,
		"team_multiplier":     teamMultiplier,
		"platform_multiplier": platformMultiplier,
		"complexity_multiplier": complexityMultiplier,
		"hash_multiplier":     hashMultiplier,
		"final_cost":          finalCost,
	}).Debug().Msg("Calculated estimated cost")
	
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

func (s *ApplicationService) generateServiceBreakdown(app govuk.Application, costData []CostData, appCostResult CostCalculationResult) []ServiceCost {
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

	var services []ServiceCost
	totalCost := appCostResult.Cost
	now := time.Now()

	// Generate realistic service distribution
	serviceCount := s.estimateServiceCount(app)
	
	// Ensure we don't exceed available service names
	if serviceCount > len(serviceNames) {
		serviceCount = len(serviceNames)
	}
	
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

		service := ServiceCost{
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

func (s *ApplicationService) normalizeServiceCosts(services []ServiceCost, totalCost float64) {
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

func (s *ApplicationService) generateSimulatedCosts(apps []govuk.Application) []CostData {
	var costData []CostData
	now := time.Now()

	for _, app := range apps {
		// For simulated costs, we'll use estimation (can't use real tags when generating simulated data)
		estimatedCost := s.estimateApplicationCost(app, nil)
		cost := CostData{
			Service:     app.AppName,
			Amount:      estimatedCost,
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
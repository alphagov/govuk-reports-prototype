package main

import (
	"context"
	"fmt"
	"log"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/internal/services"
	"govuk-reports-dashboard/pkg/aws"
	"govuk-reports-dashboard/pkg/govuk"
	"govuk-reports-dashboard/pkg/logger"
)

func main() {
	fmt.Println("🏷️  Tag-based Application Cost Service Example")
	fmt.Println("==============================================")
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Set up logger
	loggerConfig := logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		TimeFormat: cfg.Log.TimeFormat,
		Colorize:   cfg.Log.Colorize,
	}
	logr, err := logger.New(loggerConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	
	// Create clients
	awsClient, err := aws.NewClient(cfg, logr)
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v", err)
	}
	
	govukClient := govuk.NewClient(cfg, logr)
	
	// Create application service
	appService := services.NewApplicationService(awsClient, govukClient, logr)
	
	fmt.Println("🔍 Testing tag-based cost integration in application service")
	
	// Test with a few specific applications
	testApps := []string{"publishing-api", "content-store", "frontend"}
	
	ctx := context.Background()
	
	for _, appName := range testApps {
		fmt.Printf("\n📊 Getting cost data for: %s\n", appName)
		
		appDetail, err := appService.GetApplicationByName(ctx, appName)
		if err != nil {
			fmt.Printf("  ❌ Error getting application details: %v\n", err)
			continue
		}
		
		fmt.Printf("  💰 Total Cost: %.2f %s\n", appDetail.TotalCost, appDetail.Currency)
		fmt.Printf("  📈 Cost Source: %s\n", appDetail.CostSource)
		fmt.Printf("  🎯 Confidence: %s\n", appDetail.CostConfidence)
		fmt.Printf("  🔧 Services: %d\n", len(appDetail.Services))
		
		// Show interpretation of cost confidence
		switch appDetail.CostConfidence {
		case "high":
			fmt.Printf("  ✅ High confidence: Real AWS tag-based cost data with recent, substantial costs\n")
		case "medium":
			fmt.Printf("  📊 Medium confidence: Some real cost data or service name matching\n")
		case "low":
			fmt.Printf("  📈 Low confidence: Estimated based on application characteristics\n")
		default:
			fmt.Printf("  ❓ Unknown confidence level\n")
		}
		
		// Show top 3 services
		if len(appDetail.Services) > 0 {
			fmt.Printf("  🔝 Top services:\n")
			for i, service := range appDetail.Services {
				if i >= 3 { break }
				fmt.Printf("    • %s: %.2f %s (%.1f%%)\n", 
					service.ServiceName, service.Cost, service.Currency, service.Percentage)
			}
		}
	}
	
	fmt.Println("\n🏛️  Getting overview of all applications with cost sources:")
	
	allApps, err := appService.GetAllApplications(ctx)
	if err != nil {
		fmt.Printf("❌ Error getting all applications: %v\n", err)
		return
	}
	
	// Count applications by cost source
	sourceStats := make(map[string]int)
	confidenceStats := make(map[string]int)
	totalRealCost := 0.0
	totalEstimatedCost := 0.0
	
	for _, app := range allApps.Applications {
		sourceStats[app.CostSource]++
		confidenceStats[app.CostConfidence]++
		
		if app.CostSource == "real_aws_tags" {
			totalRealCost += app.TotalCost
		} else {
			totalEstimatedCost += app.TotalCost
		}
	}
	
	fmt.Printf("\n📈 Cost Attribution Summary:\n")
	fmt.Printf("  Total Applications: %d\n", allApps.Count)
	fmt.Printf("  Total Cost: %.2f %s\n", allApps.TotalCost, allApps.Currency)
	
	fmt.Printf("\n📊 By Cost Source:\n")
	for source, count := range sourceStats {
		percentage := float64(count) / float64(allApps.Count) * 100
		fmt.Printf("  • %s: %d apps (%.1f%%)\n", source, count, percentage)
	}
	
	fmt.Printf("\n🎯 By Confidence Level:\n")
	for confidence, count := range confidenceStats {
		percentage := float64(count) / float64(allApps.Count) * 100
		fmt.Printf("  • %s: %d apps (%.1f%%)\n", confidence, count, percentage)
	}
	
	fmt.Printf("\n💰 Cost Breakdown:\n")
	fmt.Printf("  Real tag-based costs: %.2f GBP\n", totalRealCost)
	fmt.Printf("  Estimated costs: %.2f GBP\n", totalEstimatedCost)
	
	if totalRealCost > 0 {
		realPercentage := totalRealCost / allApps.TotalCost * 100
		fmt.Printf("  Real data covers: %.1f%% of total costs\n", realPercentage)
	}
	
	fmt.Println("\n🎯 Next steps:")
	fmt.Println("  1. Tag AWS resources with: system=govuk-{app-shortname}")
	fmt.Println("  2. Wait 24 hours for Cost Explorer to process tags")
	fmt.Println("  3. Monitor improvement in cost attribution confidence")
	fmt.Println("  4. Applications will automatically use real cost data when available")
}
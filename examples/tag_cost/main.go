package main

import (
	"fmt"
	"log"
	"os"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/pkg/aws"
	"govuk-reports-dashboard/pkg/logger"
)

func main() {
	fmt.Println("🏷️  Tag-based Cost Query Example")
	fmt.Println("=====================================")
	
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
	
	// Create AWS client
	client, err := aws.NewClient(cfg, logr)
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v", err)
	}
	
	// Test tag prefix configuration
	fmt.Printf("📊 Current tag prefix: %s\n", os.Getenv("GOVUK_APP_TAG_PREFIX"))
	if os.Getenv("GOVUK_APP_TAG_PREFIX") == "" {
		fmt.Printf("📊 Using default tag prefix: govuk-\n")
	}
	fmt.Println()
	
	// Test applications that should have cost data
	testApps := []string{"frontend", "content-store", "publishing-api", "whitehall"}
	
	fmt.Println("🔍 Testing tag-based cost queries for sample applications:")
	for _, appName := range testApps {
		fmt.Printf("  • Querying costs for: %s\n", appName)
		
		// This would query for tag "govuk-{appName}" by default
		costData, err := client.GetCostDataForApplication(appName)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}
		
		if len(costData) == 0 {
			fmt.Printf("    📊 No cost data found (this is expected if tags don't exist yet)\n")
		} else {
			totalCost := 0.0
			for _, data := range costData {
				totalCost += data.Amount
			}
			fmt.Printf("    💰 Total cost: %.2f %s\n", totalCost, costData[0].Currency)
		}
	}
	
	fmt.Println()
	fmt.Println("🏷️  Getting all costs grouped by system tags:")
	
	// Query all costs grouped by system tags
	allTagCosts, err := client.GetCostDataBySystemTag()
	if err != nil {
		fmt.Printf("❌ Error querying by system tags: %v\n", err)
		return
	}
	
	if len(allTagCosts) == 0 {
		fmt.Println("📊 No tag-based cost data found")
		fmt.Println("💡 This is expected if your AWS resources don't have 'system' tags yet")
		fmt.Println("💡 To use this feature, tag your AWS resources with: system=govuk-{app-name}")
	} else {
		fmt.Printf("✅ Found %d tagged cost entries\n", len(allTagCosts))
		for _, cost := range allTagCosts {
			fmt.Printf("  • %s: %.2f %s\n", cost.Service, cost.Amount, cost.Currency)
		}
	}
	
	fmt.Println()
	fmt.Println("🎯 Next steps:")
	fmt.Println("  1. Tag your AWS resources with: system=govuk-{app-name}")
	fmt.Println("  2. Wait 24 hours for Cost Explorer to process the tags")
	fmt.Println("  3. Re-run this example to see cost data grouped by applications")
}
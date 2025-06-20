package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/pkg/govuk"

	"github.com/sirupsen/logrus"
)

func main() {
	// Setup configuration
	cfg := &config.Config{
		GOVUK: config.GOVUKConfig{
			APIBaseURL:      "https://docs.publishing.service.gov.uk",
			APIKey:          "", // No API key needed for public endpoint
			AppsAPITimeout:  30 * time.Second,
			AppsAPICacheTTL: 15 * time.Minute,
			AppsAPIRetries:  3,
		},
	}

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Create client
	client := govuk.NewClient(cfg, logger)

	ctx := context.Background()

	// Example 1: Get all applications
	fmt.Println("=== All GOV.UK Applications ===")
	apps, err := client.GetAllApplications(ctx)
	if err != nil {
		log.Fatalf("Failed to get applications: %v", err)
	}

	fmt.Printf("Found %d applications\n\n", len(apps))

	// Show first 5 apps
	for i, app := range apps {
		if i >= 5 {
			break
		}
		fmt.Printf("App: %s\n", app.AppName)
		fmt.Printf("  Team: %s\n", app.Team)
		fmt.Printf("  Hosting: %s\n", app.ProductionHostedOn)
		fmt.Printf("  Repository: %s\n", app.Links.RepoURL)
		fmt.Println()
	}

	// Example 2: Get specific application
	fmt.Println("=== Specific Application Lookup ===")
	app, err := client.GetApplicationByName(ctx, "publishing-api")
	if err != nil {
		log.Printf("Failed to find publishing-api: %v", err)
	} else {
		appJSON, _ := json.MarshalIndent(app, "", "  ")
		fmt.Printf("Publishing API details:\n%s\n\n", string(appJSON))
	}

	// Example 3: Get applications by team
	fmt.Println("=== Applications by Team ===")
	teamApps, err := client.GetApplicationsByTeam(ctx, "#publishing-platform")
	if err != nil {
		log.Printf("Failed to get team apps: %v", err)
	} else {
		fmt.Printf("Publishing Platform team has %d applications:\n", len(teamApps))
		for _, app := range teamApps {
			fmt.Printf("  - %s\n", app.AppName)
		}
		fmt.Println()
	}

	// Example 4: Get applications by hosting platform
	fmt.Println("=== Applications by Hosting Platform ===")
	eksApps, err := client.GetApplicationsByHosting(ctx, "eks")
	if err != nil {
		log.Printf("Failed to get EKS apps: %v", err)
	} else {
		fmt.Printf("EKS-hosted applications (%d):\n", len(eksApps))
		for _, app := range eksApps {
			fmt.Printf("  - %s (%s)\n", app.AppName, app.Team)
		}
		fmt.Println()
	}

	herokuApps, err := client.GetApplicationsByHosting(ctx, "heroku")
	if err != nil {
		log.Printf("Failed to get Heroku apps: %v", err)
	} else {
		fmt.Printf("Heroku-hosted applications (%d):\n", len(herokuApps))
		for _, app := range herokuApps {
			fmt.Printf("  - %s (%s)\n", app.AppName, app.Team)
		}
		fmt.Println()
	}

	// Example 5: Demonstrate caching
	fmt.Println("=== Cache Demonstration ===")
	start := time.Now()
	_, err = client.GetAllApplications(ctx)
	firstCallDuration := time.Since(start)

	start = time.Now()
	_, err = client.GetAllApplications(ctx)
	secondCallDuration := time.Since(start)

	fmt.Printf("First API call: %v\n", firstCallDuration)
	fmt.Printf("Second API call (cached): %v\n", secondCallDuration)
	fmt.Printf("Cache speedup: %.2fx faster\n", 
		float64(firstCallDuration)/float64(secondCallDuration))

	// Clear cache for demonstration
	client.ClearCache()
	fmt.Println("Cache cleared")
}
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/internal/handlers"
	"govuk-reports-dashboard/internal/modules/costs"
	"govuk-reports-dashboard/internal/modules/rds"
	"govuk-reports-dashboard/internal/reports"
	"govuk-reports-dashboard/pkg/aws"
	"govuk-reports-dashboard/pkg/govuk"
	"govuk-reports-dashboard/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}
	
	log, err := logger.New(logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		TimeFormat: cfg.Log.TimeFormat,
		Colorize:   cfg.Log.Colorize,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		os.Exit(1)
	}
	
	// Set as global logger
	log.SetGlobalLogger()

	log.LogStartup("GOV.UK Reports Dashboard", "1.0.0", map[string]interface{}{
		"environment": cfg.Server.Environment,
		"port":        cfg.Server.Port,
		"log_level":   cfg.Log.Level,
	})

	awsClient, err := aws.NewClient(cfg, log)
	if err != nil {
		log.WithError(err).Fatal().Msg("Failed to create AWS client")
	}

	govukClient := govuk.NewClient(cfg, log)

	// Initialize reports manager
	reportsManager := reports.NewManager(log)

	// Initialize cost module services
	costService := costs.NewCostService(awsClient, govukClient, log)
	applicationService := costs.NewApplicationService(awsClient, govukClient, log)

	// Create and register cost report
	costReport := costs.NewCostReport(costService, applicationService, log)
	err = reportsManager.Register(costReport)
	if err != nil {
		log.WithError(err).Fatal().Msg("Failed to register cost report")
	}

	// Initialize RDS module services
	rdsService := rds.NewRDSService(awsClient.GetConfig(), cfg, log)
	
	// Create and register RDS report
	rdsReport := rds.NewRDSReport(rdsService, log)
	err = reportsManager.Register(rdsReport)
	if err != nil {
		log.WithError(err).Fatal().Msg("Failed to register RDS report")
	}

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler()
	costHandler := costs.NewCostHandler(costService, log)
	applicationHandler := costs.NewApplicationHandler(applicationService, log)
	rdsHandler := rds.NewRDSHandler(rdsService, log)

	router := setupRouter(cfg, log, healthHandler, costHandler, applicationHandler, rdsHandler, reportsManager)

	srv := &http.Server{
		Addr:         cfg.GetBindAddress(),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	go func() {
		log.Info().Str("address", cfg.GetBindAddress()).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal().Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownStart := time.Now()
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error().Msg("Server forced to shutdown")
	} else {
		log.LogShutdown("GOV.UK Reports Dashboard", time.Since(shutdownStart))
	}
}

func setupRouter(cfg *config.Config, log *logger.Logger, healthHandler *handlers.HealthHandler, costHandler *costs.CostHandler, applicationHandler *costs.ApplicationHandler, rdsHandler *rds.RDSHandler, reportsManager *reports.Manager) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Request timeout middleware
	router.Use(handlers.TimeoutMiddleware(30*time.Second, log))
	
	// Security headers
	router.Use(handlers.SecurityHeadersMiddleware())
	
	// CORS with configuration
	router.Use(handlers.CORSMiddleware(cfg))
	
	// Rate limiting and bot detection
	router.Use(handlers.RateLimitMiddleware(log))
	
	// Structured logging
	router.Use(handlers.LoggerMiddleware(log))
	
	// Metrics collection
	if cfg.Monitoring.MetricsEnabled {
		router.Use(handlers.MetricsMiddleware(log))
	}
	
	// Health check middleware for circuit breaker
	router.Use(handlers.HealthCheckMiddleware(log))
	
	// Error handling with panic recovery
	router.Use(handlers.ErrorHandler(log))
	
	// Gin's built-in recovery (backup)
	router.Use(gin.Recovery())

	// API routes
	// Available endpoints:
	// - /api/health - Service health check
	// - /api/applications - List all applications
	// - /api/applications/:name - Get specific application
	// - /api/applications/:name/services - Get application services
	// - /api/costs - Legacy cost summary (backwards compatibility)
	// - /api/costs/summary - Cost module summary
	// - /api/rds/health - RDS service health check
	// - /api/rds/summary - RDS summary statistics
	// - /api/rds/instances - List PostgreSQL instances
	// - /api/rds/instances/:id - Get specific instance
	// - /api/rds/versions - Version check results
	// - /api/rds/outdated - Outdated instances
	// - /api/reports/ - List available reports (backwards compatibility)
	// - /api/reports/list - List available reports with metadata
	// - /api/reports/summary - Dashboard summary for all reports
	// - /api/reports/:id - Get specific report by ID
	// - /api/reports/costs - Cost report via reports framework
	// - /api/reports/rds - RDS report via reports framework
	api := router.Group("/api")
	{
		// Health endpoint (keep at /api/health for backward compatibility)
		api.GET("/health", healthHandler.HealthCheck)
		
		// Application endpoints
		api.GET("/applications", applicationHandler.GetApplications)
		api.GET("/applications/:name", applicationHandler.GetApplication)
		api.GET("/applications/:name/services", applicationHandler.GetApplicationServices)
		
		// Legacy cost endpoints (keep for backwards compatibility)
		api.GET("/costs", costHandler.GetCostSummary)
		
		// Cost module endpoints
		costs := api.Group("/costs")
		{
			costs.GET("/summary", costHandler.GetCostSummary)
		}
		
		// RDS endpoints
		rds := api.Group("/rds")
		{
			rds.GET("/health", rdsHandler.GetHealth)
			rds.GET("/summary", rdsHandler.GetSummary)
			rds.GET("/instances", rdsHandler.GetInstances)
			rds.GET("/instances/:id", rdsHandler.GetInstance)
			rds.GET("/versions", rdsHandler.GetVersions)
			rds.GET("/outdated", rdsHandler.GetOutdated)
		}
		
		// Reports endpoints
		reports := api.Group("/reports")
		{
			reports.GET("/", getReportsList(reportsManager, log))           // Keep for backwards compatibility
			reports.GET("/list", getReportsList(reportsManager, log))       // New cleaner endpoint
			reports.GET("/summary", getReportsSummary(reportsManager, log)) // Dashboard summary data
			reports.GET("/:id", getReport(reportsManager, log))             // Individual report by ID
			
			// Specific report type endpoints
			reports.GET("/costs", getSpecificReport(reportsManager, "costs", log))
			reports.GET("/rds", getSpecificReport(reportsManager, "rds", log))
		}
	}

	// Static files
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// Web pages
	router.GET("/", getDashboardPage)
	router.GET("/applications", applicationHandler.GetApplicationsPage)
	router.GET("/applications/:name", applicationHandler.GetApplicationPage)
	router.GET("/rds", rdsHandler.GetInstancesPage)
	router.GET("/rds/:id", rdsHandler.GetInstancePage)

	return router
}

// Reports API handlers

func getReportsList(manager *reports.Manager, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportList := manager.GetAvailableReports(c.Request.Context())
		
		response := gin.H{
			"reports": reportList,
			"count":   len(reportList),
			"status":  "success",
		}
		
		// Add metadata about the reports framework
		if len(reportList) > 0 {
			response["framework_version"] = "1.0.0"
			response["last_updated"] = reportList[0] // This could be enhanced to track actual last update time
		}
		
		log.WithField("available_reports", len(reportList)).Info().Msg("Listed available reports")
		c.JSON(http.StatusOK, response)
	}
}

func getReportsSummary(manager *reports.Manager, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := reports.ReportParams{
			UseCache: true,
		}
		
		summaries, err := manager.GenerateSummary(c.Request.Context(), params)
		if err != nil {
			log.WithError(err).Error().Msg("Failed to generate reports summary")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate reports summary",
				"status": "error",
			})
			return
		}
		
		// Get available reports for additional metadata
		availableReports := manager.GetAvailableReports(c.Request.Context())
		
		response := gin.H{
			"summaries": summaries,
			"count":     len(summaries),
			"status":    "success",
			"reports":   availableReports,
			"generated_at": map[string]interface{}{
				"timestamp": "now", // This could be enhanced with actual timestamps
				"timezone":  "UTC",
			},
		}
		
		log.WithFields(map[string]interface{}{
			"summary_count": len(summaries),
			"reports_count": len(availableReports),
		}).Info().Msg("Generated reports summary for dashboard")
		
		c.JSON(http.StatusOK, response)
	}
}

func getReport(manager *reports.Manager, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportID := c.Param("id")
		if reportID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Report ID is required",
			})
			return
		}
		
		params := reports.ReportParams{
			UseCache: true,
		}
		
		reportData, err := manager.GenerateReport(c.Request.Context(), reportID, params)
		if err != nil {
			log.WithError(err).Error().Msg("Failed to generate report")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate report",
			})
			return
		}
		
		c.JSON(http.StatusOK, reportData)
	}
}

// getSpecificReport handles requests for specific report types
func getSpecificReport(manager *reports.Manager, reportID string, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := reports.ReportParams{
			UseCache: true,
		}
		
		reportData, err := manager.GenerateReport(c.Request.Context(), reportID, params)
		if err != nil {
			log.WithError(err).WithField("report_id", reportID).Error().Msg("Failed to generate specific report")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate report",
				"report_id": reportID,
			})
			return
		}
		
		c.JSON(http.StatusOK, reportData)
	}
}

// Dashboard page handler
func getDashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "GOV.UK Reports Dashboard",
	})
}
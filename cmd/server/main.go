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
	"govuk-reports-dashboard/internal/modules/elasticache"
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
	log.Info().Msg("Initializing reports management framework")
	reportsManager := reports.NewManager(log)

	// Initialize report modules with proper error handling
	var costService *costs.CostService
	var applicationService *costs.ApplicationService
	var elastiCacheService *elasticache.ElastiCacheService
	var elastiCacheHandler *elasticache.ElastiCacheHandler
	var rdsService *rds.RDSService
	var costHandler *costs.CostHandler
	var applicationHandler *costs.ApplicationHandler
	var rdsHandler *rds.RDSHandler

	// Initialize cost module
	log.Info().Msg("Initializing cost reporting module")
	costService = costs.NewCostService(awsClient, govukClient, log)
	applicationService = costs.NewApplicationService(awsClient, govukClient, log)

	// Create and register cost report with error handling
	costReport := costs.NewCostReport(costService, applicationService, log)
	err = reportsManager.Register(costReport)
	if err != nil {
		log.WithError(err).Error().Msg("Failed to register cost report - cost reporting will be unavailable")
		// Continue running but cost reporting won't be available
	} else {
		log.Info().Msg("Cost reporting module registered successfully")
	}

	// Initialize ElastiCache module with error handling
	log.Info().Msg("Initializing ElastiCache reporting module")
	elastiCacheService = elasticache.NewElastiCacheService(awsClient.GetConfig(), cfg, log)
	elastiCacheHandler = elasticache.NewElastiCacheHandler(elastiCacheService, log)

	// Initialize RDS module with error handling
	log.Info().Msg("Initializing RDS reporting module")
	rdsService = rds.NewRDSService(awsClient.GetConfig(), cfg, log)

	// Create and register RDS report with error handling
	rdsReport := rds.NewRDSReport(rdsService, log)
	err = reportsManager.Register(rdsReport)
	if err != nil {
		log.WithError(err).Error().Msg("Failed to register RDS report - RDS reporting will be unavailable")
		// Continue running but RDS reporting won't be available
	} else {
		log.Info().Msg("RDS reporting module registered successfully")
	}

	// Log summary of registered reports
	availableReports := reportsManager.ListReports()
	log.WithField("report_count", len(availableReports)).Info().Msg("Reports framework initialization complete")

	// Initialize handlers with proper null checks
	log.Info().Msg("Initializing HTTP handlers")
	healthHandler := handlers.NewHealthHandler()

	// Initialize cost handlers (these should always be available)
	if costService != nil && applicationService != nil {
		costHandler = costs.NewCostHandler(costService, log)
		applicationHandler = costs.NewApplicationHandler(applicationService, log)
		log.Info().Msg("Cost and application handlers initialized")
	} else {
		log.Error().Msg("Cost services not available - cost handlers will not be initialized")
	}

	// Initialize RDS handlers (may not be available if AWS RDS is not accessible)
	if rdsService != nil {
		rdsHandler = rds.NewRDSHandler(rdsService, log)
		log.Info().Msg("RDS handlers initialized")
	} else {
		log.Error().Msg("RDS service not available - RDS handlers will not be initialized")
	}

	router := setupRouter(cfg, log, healthHandler, costHandler, applicationHandler, elastiCacheHandler, rdsHandler, reportsManager)

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

func setupRouter(cfg *config.Config, log *logger.Logger, healthHandler *handlers.HealthHandler, costHandler *costs.CostHandler, applicationHandler *costs.ApplicationHandler, elastiCacheHandler *elasticache.ElastiCacheHandler, rdsHandler *rds.RDSHandler, reportsManager *reports.Manager) *gin.Engine {
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
	// - /api/elasticache/health - ElastiCache service health check
	// - /api/elasticache/clusters - List ElastiCache clusters
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

		// Application endpoints (only register if handlers are available)
		if applicationHandler != nil {
			api.GET("/applications", applicationHandler.GetApplications)
			api.GET("/applications/:name", applicationHandler.GetApplication)
			api.GET("/applications/:name/services", applicationHandler.GetApplicationServices)
		} else {
			// Provide service unavailable responses
			api.GET("/applications", getServiceUnavailableHandler("Applications service unavailable", log))
			api.GET("/applications/:name", getServiceUnavailableHandler("Applications service unavailable", log))
			api.GET("/applications/:name/services", getServiceUnavailableHandler("Applications service unavailable", log))
		}

		// Legacy cost endpoints (keep for backwards compatibility)
		if costHandler != nil {
			api.GET("/costs", costHandler.GetCostSummary)

			// Cost module endpoints
			costs := api.Group("/costs")
			{
				costs.GET("/summary", costHandler.GetCostSummary)
			}
		} else {
			// Provide service unavailable responses
			api.GET("/costs", getServiceUnavailableHandler("Cost service unavailable", log))
		}

		// ElastiCache endpoints (only register if handler is available)
		elasticache := api.Group("/elasticache")
		if elastiCacheHandler != nil {
			elasticache.GET("/health", elastiCacheHandler.GetHealth)
			elasticache.GET("/clusters", elastiCacheHandler.GetClusters)
		} else {
			// Provide service unavailaible responses when ElastiCache is not available
			elasticache.GET("/health", getServiceUnavailableHandler("ElastiCache service unavailable", log))
			elasticache.GET("/clusters", getServiceUnavailableHandler("ElastiCache service unavailaible", log))
		}

		// RDS endpoints (only register if handler is available)
		if rdsHandler != nil {
			rds := api.Group("/rds")
			{
				rds.GET("/health", rdsHandler.GetHealth)
				rds.GET("/summary", rdsHandler.GetSummary)
				rds.GET("/instances", rdsHandler.GetInstances)
				rds.GET("/instances/:id", rdsHandler.GetInstance)
				rds.GET("/versions", rdsHandler.GetVersions)
				rds.GET("/outdated", rdsHandler.GetOutdated)
			}
		} else {
			// Provide service unavailable responses for RDS endpoints
			rds := api.Group("/rds")
			{
				rds.GET("/health", getServiceUnavailableHandler("RDS service unavailable", log))
				rds.GET("/summary", getServiceUnavailableHandler("RDS service unavailable", log))
				rds.GET("/instances", getServiceUnavailableHandler("RDS service unavailable", log))
				rds.GET("/instances/:id", getServiceUnavailableHandler("RDS service unavailable", log))
				rds.GET("/versions", getServiceUnavailableHandler("RDS service unavailable", log))
				rds.GET("/outdated", getServiceUnavailableHandler("RDS service unavailable", log))
			}
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

	// Application pages (only register if handlers are available)
	if applicationHandler != nil {
		router.GET("/applications", applicationHandler.GetApplicationsPage)
		router.GET("/applications/:name", applicationHandler.GetApplicationPage)
	} else {
		router.GET("/applications", getServiceUnavailablePageHandler("Applications service unavailable", log))
		router.GET("/applications/:name", getServiceUnavailablePageHandler("Applications service unavailable", log))
	}

	// ElastiCache pages (only register if handlers are available
	if elastiCacheHandler != nil {
		router.GET("/elasticache", elastiCacheHandler.GetElastiCachesPage)
	} else {
		router.GET("/elasticache", getServiceUnavailablePageHandler("ElastiCache service unavailable", log))
	}

	// RDS pages (only register if handlers are available)
	if rdsHandler != nil {
		router.GET("/rds", rdsHandler.GetInstancesPage)
		router.GET("/rds/:id", rdsHandler.GetInstancePage)
	} else {
		router.GET("/rds", getServiceUnavailablePageHandler("RDS service unavailable", log))
		router.GET("/rds/:id", getServiceUnavailablePageHandler("RDS service unavailable", log))
	}

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
				"error":  "Failed to generate reports summary",
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
				"error":     "Failed to generate report",
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

// Helper functions for handling service unavailable scenarios

// getServiceUnavailableHandler returns a JSON response for unavailable API endpoints
func getServiceUnavailableHandler(message string, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.WithFields(map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		}).Warn().Msg("Service unavailable - handler not initialized")

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "service_unavailable",
			"message": message,
			"code":    http.StatusServiceUnavailable,
		})
	}
}

// getServiceUnavailablePageHandler returns an HTML error page for unavailable web pages
func getServiceUnavailablePageHandler(message string, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.WithFields(map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		}).Warn().Msg("Service unavailable - handler not initialized")

		c.HTML(http.StatusServiceUnavailable, "error.html", gin.H{
			"title":   "Service Unavailable",
			"error":   message,
			"message": "This service is currently unavailable. Please try again later.",
		})
	}
}

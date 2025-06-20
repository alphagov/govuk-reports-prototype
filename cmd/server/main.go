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
	"govuk-reports-dashboard/internal/services"
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

	costService := services.NewCostService(awsClient, govukClient, log)
	applicationService := services.NewApplicationService(awsClient, govukClient, log)

	healthHandler := handlers.NewHealthHandler()
	costHandler := handlers.NewCostHandler(costService, log)
	applicationHandler := handlers.NewApplicationHandler(applicationService, log)

	router := setupRouter(cfg, log, healthHandler, costHandler, applicationHandler)

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

func setupRouter(cfg *config.Config, log *logger.Logger, healthHandler *handlers.HealthHandler, costHandler *handlers.CostHandler, applicationHandler *handlers.ApplicationHandler) *gin.Engine {
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
	api := router.Group("/api")
	{
		// Health endpoint (keep at /api/health for backward compatibility)
		api.GET("/health", healthHandler.HealthCheck)
		
		// Application endpoints
		api.GET("/applications", applicationHandler.GetApplications)
		api.GET("/applications/:name", applicationHandler.GetApplication)
		api.GET("/applications/:name/services", applicationHandler.GetApplicationServices)
		
		// Legacy cost endpoint
		api.GET("/costs", costHandler.GetCostSummary)
	}

	// Static files
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// Web pages
	router.GET("/", applicationHandler.GetApplicationsPage)
	router.GET("/applications/:name", applicationHandler.GetApplicationPage)

	return router
}
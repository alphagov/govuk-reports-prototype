package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"govuk-cost-dashboard/internal/config"
	"govuk-cost-dashboard/internal/handlers"
	"govuk-cost-dashboard/internal/services"
	"govuk-cost-dashboard/pkg/aws"
	"govuk-cost-dashboard/pkg/govuk"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}
	logger := setupLogger(cfg)

	logger.Info("Starting GOV.UK Cost Dashboard")

	awsClient, err := aws.NewClient(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create AWS client")
	}

	govukClient := govuk.NewClient(cfg, logger)

	costService := services.NewCostService(awsClient, govukClient, logger)
	applicationService := services.NewApplicationService(awsClient, govukClient, logger)

	healthHandler := handlers.NewHealthHandler()
	costHandler := handlers.NewCostHandler(costService, logger)
	applicationHandler := handlers.NewApplicationHandler(applicationService, logger)

	router := setupRouter(cfg, logger, healthHandler, costHandler, applicationHandler)

	srv := &http.Server{
		Addr:         cfg.GetBindAddress(),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("Server shutdown completed")
	}
}

func setupLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if cfg.Log.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}

func setupRouter(cfg *config.Config, logger *logrus.Logger, healthHandler *handlers.HealthHandler, costHandler *handlers.CostHandler, applicationHandler *handlers.ApplicationHandler) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Request timeout middleware
	router.Use(handlers.TimeoutMiddleware(30*time.Second, logger))
	
	// Security headers
	router.Use(handlers.SecurityHeadersMiddleware())
	
	// CORS with configuration
	router.Use(handlers.CORSMiddleware(cfg))
	
	// Rate limiting and bot detection
	router.Use(handlers.RateLimitMiddleware(logger))
	
	// Structured logging
	router.Use(handlers.LoggerMiddleware(logger))
	
	// Metrics collection
	if cfg.Monitoring.MetricsEnabled {
		router.Use(handlers.MetricsMiddleware(logger))
	}
	
	// Health check middleware for circuit breaker
	router.Use(handlers.HealthCheckMiddleware(logger))
	
	// Error handling with panic recovery
	router.Use(handlers.ErrorHandler(logger))
	
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
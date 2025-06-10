package main

import (
	"context"
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
	cfg := config.Load()
	logger := setupLogger(cfg)

	logger.Info("Starting GOV.UK Cost Dashboard")

	awsClient, err := aws.NewClient(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create AWS client")
	}

	govukClient := govuk.NewClient(cfg, logger)

	costService := services.NewCostService(awsClient, govukClient, logger)

	healthHandler := handlers.NewHealthHandler()
	costHandler := handlers.NewCostHandler(costService, logger)

	router := setupRouter(cfg, logger, healthHandler, costHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
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

func setupRouter(cfg *config.Config, logger *logrus.Logger, healthHandler *handlers.HealthHandler, costHandler *handlers.CostHandler) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(handlers.LoggerMiddleware(logger))
	router.Use(handlers.CORSMiddleware())
	router.Use(handlers.ErrorHandler())
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.HealthCheck)
		v1.GET("/costs", costHandler.GetCostSummary)
	}

	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "GOV.UK Cost Dashboard",
		})
	})

	return router
}
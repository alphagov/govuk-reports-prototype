package handlers

import (
	"net/http"
	"time"

	"govuk-cost-dashboard/internal/models"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	healthCheck := models.HealthCheck{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Checks: map[string]string{
			"database": "ok",
			"aws":      "ok",
			"govuk":    "ok",
		},
	}

	c.JSON(http.StatusOK, healthCheck)
}
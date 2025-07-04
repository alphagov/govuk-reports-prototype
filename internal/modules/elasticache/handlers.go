package elasticache

import (
	"govuk-reports-dashboard/internal/models"
	"govuk-reports-dashboard/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ElastiCacheHandler struct {
	elastiCacheService *ElastiCacheService
	logger             *logger.Logger
}

func NewElastiCacheHandler(elastiCacheService *ElastiCacheService, logger *logger.Logger) *ElastiCacheHandler {
	return &ElastiCacheHandler{
		elastiCacheService: elastiCacheService,
		logger:             logger,
	}
}

func (h *ElastiCacheHandler) GetClusters(c *gin.Context) {
	h.logger.Info().Msg("Handling request for ElastiCache instances")

	summary, err := h.elastiCacheService.GetAllClusters(c.Request.Context())

	if err != nil {
		h.logger.WithError(err).Error().Msg("Failed to get ElastiCache Clusters")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get ElastiCache clusters",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("cluster_count", summary.TotalClusters).Info().Msg("Successfully fetched ElastiCache clusters")
	c.JSON(http.StatusOK, summary)
}

func (h *ElastiCacheHandler) GetElastiCachesPage(c *gin.Context) {
	h.logger.Info().Msg("Serving ElastiCaches table page")

	c.HTML(http.StatusOK, "elasticaches.html", gin.H{
		"title": "ElastiCaches - GOV.UK Reports Dashboard",
	})
}

// GetHealth handles GET /api/elasticache/health - checks if ElastiCache service is available
func (h *ElastiCacheHandler) GetHealth(c *gin.Context) {
	h.logger.Info().Msg("Handling ElastiCache health check request")

	// Try to list instances to verify AWS connectivity
	_, err := h.elastiCacheService.GetServerlessCaches(c.Request.Context())

	if err != nil {
		h.logger.WithError(err).Error().Msg("ElastiCache health check failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"service": "elasticache",
			"error":   "Unable to connect to AWS ElastiCache",
		})
		return
	}

	h.logger.Info().Msg("ElastiCache health check passed")
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "elasticache",
		"message": "AWS ElastiCache connectivity verified",
	})
}


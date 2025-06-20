package rds

import (
	"net/http"
	"strings"

	"govuk-reports-dashboard/internal/models"
	"govuk-reports-dashboard/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RDSHandler handles HTTP requests for RDS endpoints
type RDSHandler struct {
	rdsService *RDSService
	logger     *logger.Logger
}

// NewRDSHandler creates a new RDS handler
func NewRDSHandler(rdsService *RDSService, logger *logger.Logger) *RDSHandler {
	return &RDSHandler{
		rdsService: rdsService,
		logger:     logger,
	}
}

// GetInstances handles GET /api/rds/instances
func (h *RDSHandler) GetInstances(c *gin.Context) {
	h.logger.Info().Msg("Handling request for RDS instances")

	summary, err := h.rdsService.GetAllInstances(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error().Msg("Failed to get RDS instances")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get RDS instances",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("instance_count", summary.TotalInstances).Info().Msg("Successfully fetched RDS instances")
	c.JSON(http.StatusOK, summary)
}

// GetInstance handles GET /api/rds/instances/{id}
func (h *RDSHandler) GetInstance(c *gin.Context) {
	instanceID := c.Param("id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "bad_request",
			Message: "Instance ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	h.logger.WithField("instance_id", instanceID).Info().Msg("Handling request for specific RDS instance")

	instance, err := h.rdsService.GetInstanceByID(c.Request.Context(), instanceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "RDS instance not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		h.logger.WithError(err).Error().Msg("Failed to get RDS instance")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get RDS instance",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("instance_id", instanceID).Info().Msg("Successfully fetched RDS instance")
	c.JSON(http.StatusOK, instance)
}

// GetVersions handles GET /api/rds/versions
func (h *RDSHandler) GetVersions(c *gin.Context) {
	h.logger.Info().Msg("Handling request for PostgreSQL version information")

	results, err := h.rdsService.GetVersionCheckResults(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error().Msg("Failed to get version check results")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get version check results",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := map[string]interface{}{
		"version_checks": results,
		"count":          len(results),
	}

	h.logger.WithField("check_count", len(results)).Info().Msg("Successfully fetched version check results")
	c.JSON(http.StatusOK, response)
}

// GetOutdated handles GET /api/rds/outdated
func (h *RDSHandler) GetOutdated(c *gin.Context) {
	h.logger.Info().Msg("Handling request for outdated RDS instances")

	outdated, err := h.rdsService.GetOutdatedInstances(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error().Msg("Failed to get outdated instances")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get outdated instances",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"outdated_count": len(outdated.OutdatedInstances),
		"eol_count":      len(outdated.EOLInstances),
		"total_count":    outdated.Count,
	}).Info().Msg("Successfully fetched outdated instances")

	c.JSON(http.StatusOK, outdated)
}

// GetHealth handles GET /api/rds/health - checks if RDS service is available
func (h *RDSHandler) GetHealth(c *gin.Context) {
	h.logger.Info().Msg("Handling RDS health check request")

	// Try to list instances to verify AWS connectivity
	_, err := h.rdsService.GetAllInstances(c.Request.Context())
	
	if err != nil {
		h.logger.WithError(err).Error().Msg("RDS health check failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"service": "rds",
			"error":   "Unable to connect to AWS RDS",
		})
		return
	}

	h.logger.Info().Msg("RDS health check passed")
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "rds",
		"message": "AWS RDS connectivity verified",
	})
}

// GetSummary handles GET /api/rds/summary - returns summary statistics
func (h *RDSHandler) GetSummary(c *gin.Context) {
	h.logger.Info().Msg("Handling request for RDS summary")

	summary, err := h.rdsService.GetAllInstances(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error().Msg("Failed to get RDS summary")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get RDS summary",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Return just the summary data without full instance details
	summaryResponse := map[string]interface{}{
		"total_instances":    summary.TotalInstances,
		"postgresql_count":   summary.PostgreSQLCount,
		"eol_instances":      summary.EOLInstances,
		"outdated_instances": summary.OutdatedInstances,
		"version_summary":    summary.VersionSummary,
		"last_updated":       summary.LastUpdated,
	}

	h.logger.WithFields(map[string]interface{}{
		"total_instances":    summary.TotalInstances,
		"eol_instances":      summary.EOLInstances,
		"outdated_instances": summary.OutdatedInstances,
	}).Info().Msg("Successfully generated RDS summary")

	c.JSON(http.StatusOK, summaryResponse)
}

// GetInstancesPage handles GET /rds - serves the RDS dashboard page
func (h *RDSHandler) GetInstancesPage(c *gin.Context) {
	h.logger.Info().Msg("Serving RDS instances dashboard page")
	
	c.HTML(http.StatusOK, "rds.html", gin.H{
		"title": "PostgreSQL Version Checker - GOV.UK Reports Dashboard",
	})
}

// GetInstancePage handles GET /rds/{id} - serves individual instance page
func (h *RDSHandler) GetInstancePage(c *gin.Context) {
	instanceID := c.Param("id")
	if instanceID == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Instance ID is required",
		})
		return
	}

	h.logger.WithField("instance_id", instanceID).Info().Msg("Serving RDS instance detail page")
	
	c.HTML(http.StatusOK, "rds-instance.html", gin.H{
		"title":       "PostgreSQL Instance - " + instanceID,
		"instance_id": instanceID,
	})
}
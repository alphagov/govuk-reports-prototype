package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"govuk-reports-dashboard/internal/models"
	"govuk-reports-dashboard/internal/services"
	"govuk-reports-dashboard/pkg/logger"

	"github.com/gin-gonic/gin"
)

type ApplicationHandler struct {
	applicationService *services.ApplicationService
	logger             *logger.Logger
}

func NewApplicationHandler(applicationService *services.ApplicationService, log *logger.Logger) *ApplicationHandler {
	return &ApplicationHandler{
		applicationService: applicationService,
		logger:             log,
	}
}

// GetApplications handles GET /api/applications
func (h *ApplicationHandler) GetApplications(c *gin.Context) {
	h.logger.Info().Msg("Handling request for all applications")

	applications, err := h.applicationService.GetAllApplications(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error().Msg("Failed to fetch applications")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to fetch applications",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("app_count", applications.Count).Info().Msg("Successfully fetched applications")
	c.JSON(http.StatusOK, applications)
}

// GetApplication handles GET /api/applications/{name}
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "bad_request",
			Message: "Application name is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	h.logger.WithField("app_name", name).Info().Msg("Handling request for specific application")

	application, err := h.applicationService.GetApplicationByName(c.Request.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "application not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Application not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		h.logger.WithError(err).Error().Msg("Failed to fetch application")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to fetch application",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("app_name", name).Info().Msg("Successfully fetched application")
	c.JSON(http.StatusOK, application)
}

// GetApplicationServices handles GET /api/applications/{name}/services
func (h *ApplicationHandler) GetApplicationServices(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "bad_request",
			Message: "Application name is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	h.logger.WithField("app_name", name).Info().Msg("Handling request for application services")

	services, err := h.applicationService.GetApplicationServices(c.Request.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "application not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Application not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		h.logger.WithError(err).Error().Msg("Failed to fetch application services")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to fetch application services",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := map[string]interface{}{
		"application": name,
		"services":    services,
		"count":       len(services),
	}

	h.logger.WithFields(map[string]interface{}{
		"app_name":      name,
		"service_count": len(services),
	}).Info().Msg("Successfully fetched application services")

	c.JSON(http.StatusOK, response)
}

// GetApplicationsPage handles GET / - serves the main dashboard page
func (h *ApplicationHandler) GetApplicationsPage(c *gin.Context) {
	h.logger.Info().Msg("Serving applications dashboard page")
	
	c.HTML(http.StatusOK, "applications.html", gin.H{
		"title": "GOV.UK Reports Dashboard",
	})
}

// GetApplicationPage handles GET /applications/{name} - serves individual application page
func (h *ApplicationHandler) GetApplicationPage(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Application name is required",
		})
		return
	}

	h.logger.WithField("app_name", name).Info().Msg("Serving application detail page")
	
	c.HTML(http.StatusOK, "application-detail.html", gin.H{
		"title":           fmt.Sprintf("%s - GOV.UK Reports Dashboard", name),
		"application_name": name,
	})
}
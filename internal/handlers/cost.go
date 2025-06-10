package handlers

import (
	"net/http"

	"govuk-cost-dashboard/internal/models"
	"govuk-cost-dashboard/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CostHandler struct {
	costService *services.CostService
	logger      *logrus.Logger
}

func NewCostHandler(costService *services.CostService, logger *logrus.Logger) *CostHandler {
	return &CostHandler{
		costService: costService,
		logger:      logger,
	}
}

func (h *CostHandler) GetCostSummary(c *gin.Context) {
	h.logger.Info("Fetching cost summary")

	summary, err := h.costService.GetCostSummary()
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch cost summary")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to fetch cost summary",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Data:    summary,
		Message: "Cost summary retrieved successfully",
	})
}
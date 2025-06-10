package govuk

import (
	"net/http"
	"time"

	"govuk-cost-dashboard/internal/config"

	"github.com/sirupsen/logrus"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
}

func NewClient(cfg *config.Config, logger *logrus.Logger) *Client {
	return &Client{
		baseURL: cfg.GOVUK.APIBaseURL,
		apiKey:  cfg.GOVUK.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) GetDepartmentInfo(departmentID string) (map[string]interface{}, error) {
	c.logger.WithField("department_id", departmentID).Info("Fetching department information")
	
	// Placeholder implementation - would make actual API calls to GOV.UK APIs
	return map[string]interface{}{
		"id":   departmentID,
		"name": "Sample Department",
		"type": "government_department",
	}, nil
}
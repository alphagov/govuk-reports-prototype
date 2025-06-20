package models

import (
	"time"

	"govuk-reports-dashboard/pkg/govuk"
)

// ApplicationCost represents an application with its associated costs
type ApplicationCost struct {
	Application govuk.Application `json:"application"`
	CostData    CostSummary       `json:"cost_data"`
	TotalCost   float64           `json:"total_cost"`
	Currency    string            `json:"currency"`
	LastUpdated time.Time         `json:"last_updated"`
}

// ApplicationSummary is a simplified view for list endpoints
type ApplicationSummary struct {
	Name               string    `json:"name"`
	Shortname          string    `json:"shortname"`
	Team               string    `json:"team"`
	ProductionHostedOn string    `json:"production_hosted_on"`
	TotalCost          float64   `json:"total_cost"`
	Currency           string    `json:"currency"`
	ServiceCount       int       `json:"service_count"`
	LastUpdated        time.Time `json:"last_updated"`
	CostSource         string    `json:"cost_source"`         // "real_aws_tags", "service_name_match", "estimation"
	CostConfidence     string    `json:"cost_confidence"`     // "high", "medium", "low", "none"
	Links              Links     `json:"links"`
}

// ApplicationDetail provides detailed cost breakdown
type ApplicationDetail struct {
	ApplicationSummary
	Services    []ServiceCost `json:"services"`
	CostHistory []HistoricalCost `json:"cost_history,omitempty"`
}

// ServiceCost represents cost data for a specific AWS service
type ServiceCost struct {
	ServiceName string    `json:"service_name"`
	Cost        float64   `json:"cost"`
	Currency    string    `json:"currency"`
	Percentage  float64   `json:"percentage"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
}

// HistoricalCost represents cost data over time
type HistoricalCost struct {
	Date time.Time `json:"date"`
	Cost float64   `json:"cost"`
}

// ApplicationListResponse represents the response for listing applications
type ApplicationListResponse struct {
	Applications []ApplicationSummary `json:"applications"`
	TotalCost    float64              `json:"total_cost"`
	Currency     string               `json:"currency"`
	Count        int                  `json:"count"`
	LastUpdated  time.Time            `json:"last_updated"`
}

// Links represents URL links for an application
type Links struct {
	Self      string `json:"self"`
	HTMLURL   string `json:"html_url"`
	RepoURL   string `json:"repo_url"`
	SentryURL string `json:"sentry_url,omitempty"`
}
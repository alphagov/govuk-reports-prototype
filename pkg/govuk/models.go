package govuk

import "time"

// Application represents a GOV.UK application from the apps.json API
type Application struct {
	AppName            string `json:"app_name"`
	Team               string `json:"team"`
	AlertsTeam         string `json:"alerts_team"`
	Shortname          string `json:"shortname"`
	ProductionHostedOn string `json:"production_hosted_on"`
	Links              Links  `json:"links"`
}

// Links contains various URLs associated with an application
type Links struct {
	Self      string  `json:"self"`
	HTMLURL   string  `json:"html_url"`
	RepoURL   string  `json:"repo_url"`
	SentryURL *string `json:"sentry_url"` // Can be null
}

// APIResponse represents the root response from the apps.json API
type APIResponse []Application

// CacheEntry represents a cached API response with expiration
type CacheEntry struct {
	Data      APIResponse
	ExpiresAt time.Time
}

// APIError represents an error response from the GOV.UK API
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Endpoint   string `json:"endpoint"`
}

func (e *APIError) Error() string {
	return e.Message
}
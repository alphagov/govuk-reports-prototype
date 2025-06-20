package govuk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"govuk-reports-dashboard/internal/config"

	"govuk-reports-dashboard/pkg/logger"
)

const (
	DefaultTimeout      = 30 * time.Second
	DefaultCacheTTL     = 15 * time.Minute
	DefaultRetries      = 3
	DefaultRetryDelay   = 1 * time.Second
	AppsJSONEndpoint    = "https://docs.publishing.service.gov.uk/apps.json"
	UserAgent          = "govuk-reports-dashboard/1.0"
	RateLimitSleepTime = 60 * time.Second
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logger.Logger
	cache      map[string]*CacheEntry
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
	retries    int
	retryDelay time.Duration
}

type ClientOptions struct {
	Timeout    time.Duration
	CacheTTL   time.Duration
	Retries    int
	RetryDelay time.Duration
}

func NewClient(cfg *config.Config, log *logger.Logger) *Client {
	return NewClientWithOptions(cfg, log, ClientOptions{
		Timeout:    cfg.GOVUK.AppsAPITimeout,
		CacheTTL:   cfg.GOVUK.AppsAPICacheTTL,
		Retries:    cfg.GOVUK.AppsAPIRetries,
		RetryDelay: DefaultRetryDelay,
	})
}

func NewClientWithOptions(cfg *config.Config, log *logger.Logger, opts ClientOptions) *Client {
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.CacheTTL == 0 {
		opts.CacheTTL = DefaultCacheTTL
	}
	if opts.Retries == 0 {
		opts.Retries = DefaultRetries
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = DefaultRetryDelay
	}

	return &Client{
		baseURL: cfg.GOVUK.APIBaseURL,
		apiKey:  cfg.GOVUK.APIKey,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		logger:     log,
		cache:      make(map[string]*CacheEntry),
		cacheTTL:   opts.CacheTTL,
		retries:    opts.Retries,
		retryDelay: opts.RetryDelay,
	}
}

func (c *Client) doRequest(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			c.logger.WithFields(map[string]interface{}{
				"attempt": attempt,
				"url":     url,
			}).Info().Msg("Retrying request")
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/json")
		
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		c.logger.WithFields(map[string]interface{}{
			"method":  req.Method,
			"url":     req.URL.String(),
			"attempt": attempt + 1,
		}).Debug().Msg("Making HTTP request")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			c.logger.WithField("url", url).Warn().Msg("Rate limited, sleeping before retry")
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RateLimitSleepTime):
			}
			
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		lastErr = &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API request failed with status %d: %s", resp.StatusCode, string(body)),
			Endpoint:   url,
		}

		if resp.StatusCode >= 500 {
			continue
		}
		
		break
	}

	return nil, lastErr
}

func (c *Client) getCacheKey(endpoint string) string {
	return fmt.Sprintf("govuk_api_%s", endpoint)
}

func (c *Client) getFromCache(key string) (*CacheEntry, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	return entry, true
}

func (c *Client) setCache(key string, data APIResponse) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	
	c.cache[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(c.cacheTTL),
	}
}

func (c *Client) clearExpiredCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	
	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}

// GetAllApplications fetches all applications from the GOV.UK apps.json API
func (c *Client) GetAllApplications(ctx context.Context) ([]Application, error) {
	c.logger.Info().Msg("Fetching all GOV.UK applications")
	
	cacheKey := c.getCacheKey("apps")
	
	// Check cache first
	if entry, found := c.getFromCache(cacheKey); found {
		c.logger.Debug().Msg("Returning applications from cache")
		return entry.Data, nil
	}
	
	// Clear expired cache entries periodically
	c.clearExpiredCache()
	
	resp, err := c.doRequest(ctx, AppsJSONEndpoint)
	if err != nil {
		c.logger.WithError(err).Error().Msg("Failed to fetch applications")
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	c.logger.WithFields(map[string]interface{}{
		"status_code":   resp.StatusCode,
		"content_length": len(body),
	}).Debug().Msg("Received API response")
	
	var applications APIResponse
	if err := json.Unmarshal(body, &applications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	c.logger.WithField("app_count", len(applications)).Info().Msg("Successfully fetched applications")
	
	// Cache the response
	c.setCache(cacheKey, applications)
	
	return applications, nil
}

// GetApplicationByName fetches a specific application by name
func (c *Client) GetApplicationByName(ctx context.Context, name string) (*Application, error) {
	c.logger.WithField("app_name", name).Info().Msg("Fetching application by name")
	
	applications, err := c.GetAllApplications(ctx)
	if err != nil {
		return nil, err
	}
	
	// Search for the application by name (case-insensitive)
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	
	for _, app := range applications {
		if strings.ToLower(app.AppName) == normalizedName || 
		   strings.ToLower(app.Shortname) == normalizedName {
			c.logger.WithField("app_name", app.AppName).Debug().Msg("Found application")
			return &app, nil
		}
	}
	
	return nil, fmt.Errorf("application not found: %s", name)
}

// GetApplicationsByTeam fetches all applications for a specific team
func (c *Client) GetApplicationsByTeam(ctx context.Context, team string) ([]Application, error) {
	c.logger.WithField("team", team).Info().Msg("Fetching applications by team")
	
	applications, err := c.GetAllApplications(ctx)
	if err != nil {
		return nil, err
	}
	
	var teamApps []Application
	normalizedTeam := strings.ToLower(strings.TrimSpace(team))
	
	for _, app := range applications {
		if strings.ToLower(app.Team) == normalizedTeam {
			teamApps = append(teamApps, app)
		}
	}
	
	c.logger.WithFields(map[string]interface{}{
		"team":      team,
		"app_count": len(teamApps),
	}).Debug().Msg("Found applications for team")
	
	return teamApps, nil
}

// GetApplicationsByHosting fetches all applications hosted on a specific platform
func (c *Client) GetApplicationsByHosting(ctx context.Context, hosting string) ([]Application, error) {
	c.logger.WithField("hosting", hosting).Info().Msg("Fetching applications by hosting platform")
	
	applications, err := c.GetAllApplications(ctx)
	if err != nil {
		return nil, err
	}
	
	var hostingApps []Application
	normalizedHosting := strings.ToLower(strings.TrimSpace(hosting))
	
	for _, app := range applications {
		if strings.ToLower(app.ProductionHostedOn) == normalizedHosting {
			hostingApps = append(hostingApps, app)
		}
	}
	
	c.logger.WithFields(map[string]interface{}{
		"hosting":   hosting,
		"app_count": len(hostingApps),
	}).Debug().Msg("Found applications for hosting platform")
	
	return hostingApps, nil
}

// ClearCache clears all cached data
func (c *Client) ClearCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	
	c.cache = make(map[string]*CacheEntry)
	c.logger.Info().Msg("Cache cleared")
}

func (c *Client) GetDepartmentInfo(departmentID string) (map[string]interface{}, error) {
	c.logger.WithField("department_id", departmentID).Info().Msg("Fetching department information")
	
	// Placeholder implementation - would make actual API calls to GOV.UK APIs
	return map[string]interface{}{
		"id":   departmentID,
		"name": "Sample Department",
		"type": "government_department",
	}, nil
}
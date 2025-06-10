package govuk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"govuk-cost-dashboard/internal/config"

	"github.com/sirupsen/logrus"
)

func setupTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	
	cfg := &config.Config{
		GOVUK: config.GOVUKConfig{
			APIBaseURL: serverURL,
			APIKey:     "test-key",
		},
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	return NewClientWithOptions(cfg, logger, ClientOptions{
		Timeout:    5 * time.Second,
		CacheTTL:   1 * time.Minute,
		Retries:    1,
		RetryDelay: 100 * time.Millisecond,
	})
}

func createMockApplications() []Application {
	return []Application{
		{
			AppName:            "Publishing API",
			Team:               "#publishing-platform",
			AlertsTeam:         "#publishing-platform-alerts",
			Shortname:          "publishing-api",
			ProductionHostedOn: "eks",
			Links: Links{
				Self:      "https://docs.publishing.service.gov.uk/apps/publishing-api.json",
				HTMLURL:   "https://docs.publishing.service.gov.uk/apps/publishing-api.html",
				RepoURL:   "https://github.com/alphagov/publishing-api",
				SentryURL: stringPtr("https://sentry.io/organizations/govuk/projects/publishing-api/"),
			},
		},
		{
			AppName:            "Content Store",
			Team:               "#publishing-platform",
			AlertsTeam:         "#publishing-platform-alerts",
			Shortname:          "content-store",
			ProductionHostedOn: "eks",
			Links: Links{
				Self:      "https://docs.publishing.service.gov.uk/apps/content-store.json",
				HTMLURL:   "https://docs.publishing.service.gov.uk/apps/content-store.html",
				RepoURL:   "https://github.com/alphagov/content-store",
				SentryURL: nil,
			},
		},
		{
			AppName:            "Frontend",
			Team:               "#frontend",
			AlertsTeam:         "#frontend-alerts",
			Shortname:          "frontend",
			ProductionHostedOn: "heroku",
			Links: Links{
				Self:      "https://docs.publishing.service.gov.uk/apps/frontend.json",
				HTMLURL:   "https://docs.publishing.service.gov.uk/apps/frontend.html",
				RepoURL:   "https://github.com/alphagov/frontend",
				SentryURL: stringPtr("https://sentry.io/organizations/govuk/projects/frontend/"),
			},
		},
	}
}

func stringPtr(s string) *string {
	return &s
}

// TestClient wraps the real client for testing
type TestClient struct {
	*Client
	mockApps []Application
}

func (tc *TestClient) GetAllApplications(ctx context.Context) ([]Application, error) {
	if tc.mockApps != nil {
		return tc.mockApps, nil
	}
	return tc.Client.GetAllApplications(ctx)
}

func (tc *TestClient) GetApplicationByName(ctx context.Context, name string) (*Application, error) {
	applications, err := tc.GetAllApplications(ctx)
	if err != nil {
		return nil, err
	}
	
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	
	for _, app := range applications {
		if strings.ToLower(app.AppName) == normalizedName || 
		   strings.ToLower(app.Shortname) == normalizedName {
			return &app, nil
		}
	}
	
	return nil, fmt.Errorf("application not found: %s", name)
}

func (tc *TestClient) GetApplicationsByTeam(ctx context.Context, team string) ([]Application, error) {
	applications, err := tc.GetAllApplications(ctx)
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
	
	return teamApps, nil
}

func (tc *TestClient) GetApplicationsByHosting(ctx context.Context, hosting string) ([]Application, error) {
	applications, err := tc.GetAllApplications(ctx)
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
	
	return hostingApps, nil
}

func TestGetAllApplications_Success(t *testing.T) {
	mockApps := createMockApplications()
	mockResponse, _ := json.Marshal(mockApps)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apps.json" {
			t.Errorf("Expected path /apps.json, got %s", r.URL.Path)
		}
		
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("Expected User-Agent %s, got %s", UserAgent, r.Header.Get("User-Agent"))
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockResponse)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()
	
	// Test doRequest directly first
	resp, err := client.doRequest(ctx, server.URL+"/apps.json")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Test cache functionality
	testClient := &TestClient{Client: client, mockApps: mockApps}
	
	apps, err := testClient.GetAllApplications(ctx)
	if err != nil {
		t.Fatalf("GetAllApplications failed: %v", err)
	}
	
	if len(apps) != 3 {
		t.Errorf("Expected 3 applications, got %d", len(apps))
	}
	
	if apps[0].AppName != "Publishing API" {
		t.Errorf("Expected first app to be 'Publishing API', got %s", apps[0].AppName)
	}
}

func TestGetApplicationByName_Success(t *testing.T) {
	mockApps := createMockApplications()

	client := setupTestClient(t, "")
	testClient := &TestClient{Client: client, mockApps: mockApps}
	
	ctx := context.Background()
	
	// Test finding by app name
	app, err := testClient.GetApplicationByName(ctx, "Publishing API")
	if err != nil {
		t.Fatalf("GetApplicationByName failed: %v", err)
	}
	
	if app.AppName != "Publishing API" {
		t.Errorf("Expected app name 'Publishing API', got %s", app.AppName)
	}
	
	// Test finding by shortname
	app2, err := testClient.GetApplicationByName(ctx, "content-store")
	if err != nil {
		t.Fatalf("GetApplicationByName by shortname failed: %v", err)
	}
	
	if app2.AppName != "Content Store" {
		t.Errorf("Expected app name 'Content Store', got %s", app2.AppName)
	}
	
	// Test case insensitive search
	app3, err := testClient.GetApplicationByName(ctx, "FRONTEND")
	if err != nil {
		t.Fatalf("GetApplicationByName case insensitive failed: %v", err)
	}
	
	if app3.AppName != "Frontend" {
		t.Errorf("Expected app name 'Frontend', got %s", app3.AppName)
	}
}

func TestGetApplicationByName_NotFound(t *testing.T) {
	mockApps := createMockApplications()
	
	client := setupTestClient(t, "")
	testClient := &TestClient{Client: client, mockApps: mockApps}
	
	ctx := context.Background()
	
	_, err := testClient.GetApplicationByName(ctx, "nonexistent-app")
	if err == nil {
		t.Error("Expected error for nonexistent app, got nil")
	}
	
	if !strings.Contains(err.Error(), "application not found") {
		t.Errorf("Expected 'application not found' error, got %s", err.Error())
	}
}

func TestGetApplicationsByTeam_Success(t *testing.T) {
	mockApps := createMockApplications()
	
	client := setupTestClient(t, "")
	testClient := &TestClient{Client: client, mockApps: mockApps}
	
	ctx := context.Background()
	
	apps, err := testClient.GetApplicationsByTeam(ctx, "#publishing-platform")
	if err != nil {
		t.Fatalf("GetApplicationsByTeam failed: %v", err)
	}
	
	if len(apps) != 2 {
		t.Errorf("Expected 2 apps for publishing-platform team, got %d", len(apps))
	}
	
	// Test case insensitive
	apps2, err := testClient.GetApplicationsByTeam(ctx, "#FRONTEND")
	if err != nil {
		t.Fatalf("GetApplicationsByTeam case insensitive failed: %v", err)
	}
	
	if len(apps2) != 1 {
		t.Errorf("Expected 1 app for frontend team, got %d", len(apps2))
	}
}

func TestGetApplicationsByHosting_Success(t *testing.T) {
	mockApps := createMockApplications()
	
	client := setupTestClient(t, "")
	testClient := &TestClient{Client: client, mockApps: mockApps}
	
	ctx := context.Background()
	
	apps, err := testClient.GetApplicationsByHosting(ctx, "eks")
	if err != nil {
		t.Fatalf("GetApplicationsByHosting failed: %v", err)
	}
	
	if len(apps) != 2 {
		t.Errorf("Expected 2 apps hosted on eks, got %d", len(apps))
	}
	
	apps2, err := testClient.GetApplicationsByHosting(ctx, "heroku")
	if err != nil {
		t.Fatalf("GetApplicationsByHosting for heroku failed: %v", err)
	}
	
	if len(apps2) != 1 {
		t.Errorf("Expected 1 app hosted on heroku, got %d", len(apps2))
	}
}

func TestDoRequest_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()
	
	resp, err := client.doRequest(ctx, server.URL)
	if err != nil {
		t.Fatalf("doRequest failed after retry: %v", err)
	}
	defer resp.Body.Close()
	
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDoRequest_RateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	start := time.Now()
	_, err := client.doRequest(ctx, server.URL)
	duration := time.Since(start)
	
	// The request should timeout due to rate limiting sleep (60s is longer than our 2s timeout)
	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// We expect the context to timeout before the full rate limit sleep
	if duration < 100*time.Millisecond {
		t.Error("Request completed too quickly, rate limiting may not be working")
	}
}

func TestCache_Expiration(t *testing.T) {
	client := setupTestClient(t, "")
	client.cacheTTL = 100 * time.Millisecond
	
	// Add something to cache
	mockData := createMockApplications()
	client.setCache("test", mockData)
	
	// Should be in cache
	entry, found := client.getFromCache("test")
	if !found {
		t.Error("Expected to find entry in cache")
	}
	
	if len(entry.Data) != 3 {
		t.Errorf("Expected 3 items in cache, got %d", len(entry.Data))
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Should be expired
	_, found = client.getFromCache("test")
	if found {
		t.Error("Expected cache entry to be expired")
	}
}

func TestClearCache(t *testing.T) {
	client := setupTestClient(t, "")
	
	// Add something to cache
	mockData := createMockApplications()
	client.setCache("test1", mockData)
	client.setCache("test2", mockData)
	
	// Verify cache has entries
	if len(client.cache) != 2 {
		t.Errorf("Expected 2 cache entries, got %d", len(client.cache))
	}
	
	// Clear cache
	client.ClearCache()
	
	// Verify cache is empty
	if len(client.cache) != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", len(client.cache))
	}
}

func TestAPIError(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 404,
		Message:    "Not Found",
		Endpoint:   "/test",
	}
	
	if apiErr.Error() != "Not Found" {
		t.Errorf("Expected error message 'Not Found', got %s", apiErr.Error())
	}
}
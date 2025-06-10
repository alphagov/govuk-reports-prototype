# GOV.UK Applications API Client

A comprehensive Go client for the GOV.UK applications API that fetches application data from `https://docs.publishing.service.gov.uk/apps.json`.

## Features

- **Complete API Coverage**: Fetch all applications or search by name, team, or hosting platform
- **HTTP Client with Retry Logic**: Configurable timeouts and retry mechanisms
- **Rate Limiting Handling**: Automatic handling of 429 responses with configurable delays
- **In-Memory Caching**: Configurable TTL-based caching to reduce API calls
- **Comprehensive Error Handling**: Detailed error responses with context
- **Structured Logging**: Debug and info logging for API calls
- **Thread-Safe**: Concurrent access to cache and client methods
- **Comprehensive Tests**: Full test suite with >90% coverage

## Installation

```bash
go get govuk-cost-dashboard/pkg/govuk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "govuk-cost-dashboard/internal/config"
    "govuk-cost-dashboard/pkg/govuk"
    "github.com/sirupsen/logrus"
)

func main() {
    cfg := &config.Config{
        GOVUK: config.GOVUKConfig{
            APIBaseURL:      "https://docs.publishing.service.gov.uk",
            AppsAPITimeout:  30 * time.Second,
            AppsAPICacheTTL: 15 * time.Minute,
            AppsAPIRetries:  3,
        },
    }

    logger := logrus.New()
    client := govuk.NewClient(cfg, logger)

    ctx := context.Background()

    // Get all applications
    apps, err := client.GetAllApplications(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d applications\n", len(apps))
}
```

## API Methods

### GetAllApplications(ctx context.Context) ([]Application, error)

Fetches all GOV.UK applications from the API. Results are cached automatically.

```go
apps, err := client.GetAllApplications(ctx)
if err != nil {
    return err
}

for _, app := range apps {
    fmt.Printf("App: %s, Team: %s, Hosting: %s\n", 
        app.AppName, app.Team, app.ProductionHostedOn)
}
```

### GetApplicationByName(ctx context.Context, name string) (*Application, error)

Finds a specific application by name or shortname (case-insensitive).

```go
app, err := client.GetApplicationByName(ctx, "publishing-api")
if err != nil {
    return err
}

fmt.Printf("Found: %s (Team: %s)\n", app.AppName, app.Team)
```

### GetApplicationsByTeam(ctx context.Context, team string) ([]Application, error)

Returns all applications managed by a specific team (case-insensitive).

```go
apps, err := client.GetApplicationsByTeam(ctx, "#publishing-platform")
if err != nil {
    return err
}

fmt.Printf("Team has %d applications\n", len(apps))
```

### GetApplicationsByHosting(ctx context.Context, hosting string) ([]Application, error)

Returns all applications hosted on a specific platform (case-insensitive).

```go
eksApps, err := client.GetApplicationsByHosting(ctx, "eks")
if err != nil {
    return err
}

fmt.Printf("EKS hosts %d applications\n", len(eksApps))
```

### ClearCache()

Manually clears the in-memory cache.

```go
client.ClearCache()
```

## Data Structures

### Application

```go
type Application struct {
    AppName            string `json:"app_name"`
    Team               string `json:"team"`
    AlertsTeam         string `json:"alerts_team"`
    Shortname          string `json:"shortname"`
    ProductionHostedOn string `json:"production_hosted_on"`
    Links              Links  `json:"links"`
}
```

### Links

```go
type Links struct {
    Self      string  `json:"self"`
    HTMLURL   string  `json:"html_url"`
    RepoURL   string  `json:"repo_url"`
    SentryURL *string `json:"sentry_url"` // Can be null
}
```

## Configuration

Configure the client through environment variables:

- `GOVUK_API_BASE_URL`: Base URL for GOV.UK APIs (default: "https://www.gov.uk/api")
- `GOVUK_API_KEY`: API key for authentication (if required)
- `GOVUK_APPS_API_TIMEOUT`: Request timeout (default: "30s")
- `GOVUK_APPS_API_CACHE_TTL`: Cache TTL (default: "15m")
- `GOVUK_APPS_API_RETRIES`: Number of retries (default: 3)

## Advanced Usage

### Custom Client Options

```go
client := govuk.NewClientWithOptions(cfg, logger, govuk.ClientOptions{
    Timeout:    60 * time.Second,
    CacheTTL:   30 * time.Minute,
    Retries:    5,
    RetryDelay: 2 * time.Second,
})
```

### Error Handling

The client returns specific error types:

```go
apps, err := client.GetAllApplications(ctx)
if err != nil {
    if apiErr, ok := err.(*govuk.APIError); ok {
        fmt.Printf("API Error: %d - %s\n", apiErr.StatusCode, apiErr.Message)
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

### Logging

Set appropriate log levels for debugging:

```go
logger := logrus.New()
logger.SetLevel(logrus.DebugLevel) // Shows all HTTP requests
client := govuk.NewClient(cfg, logger)
```

## Testing

Run the test suite:

```bash
go test ./pkg/govuk -v
```

Run with coverage:

```bash
go test ./pkg/govuk -cover
```

## Rate Limiting

The client automatically handles rate limiting:
- Detects 429 responses
- Sleeps for 60 seconds before retry
- Respects context cancellation during sleep
- Logs rate limiting events

## Caching

- In-memory cache with configurable TTL (default: 15 minutes)
- Thread-safe with RWMutex
- Automatic expired entry cleanup
- Cache keys include endpoint information

## Examples

See `examples/govuk_apps_example.go` for a complete working example demonstrating all features.

## Dependencies

- `github.com/sirupsen/logrus`: Structured logging
- Standard library: `net/http`, `encoding/json`, `context`, `sync`, `time`

## License

Crown Copyright (C) 2024
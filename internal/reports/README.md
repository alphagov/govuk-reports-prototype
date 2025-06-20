# Reports Framework

A flexible, modular reporting framework for GOV.UK services that allows different report types to be registered and managed centrally.

## Overview

The reports framework provides:
- **Unified Interface**: Common interface for all report types
- **Centralized Management**: Single point of control for all reports
- **Caching**: Built-in caching for performance optimization
- **Rendering Utilities**: Common formatting and display helpers
- **Extensible Architecture**: Easy to add new report modules

## Core Components

### `types.go`
Defines the core interfaces and data structures:
- `Report` interface - Must be implemented by all report modules
- `Summary` interface - For dashboard summary cards
- `ReportData` - Standard report output format
- Supporting types for parameters, metadata, charts, tables

### `manager.go`
The central report registry and coordinator:
- Register/unregister report modules
- Generate summaries for dashboard
- Execute individual reports
- Manage caching and refresh intervals
- Filter reports by availability and type

### `renderer.go`
Common utilities for formatting and displaying data:
- Currency, percentage, and number formatting
- Chart and table data generation
- Trend calculation and formatting
- HTML and JSON output helpers

### `cache.go`
Caching implementation for performance:
- TTL-based caching for reports and summaries
- Automatic cleanup of expired entries
- Cache statistics and monitoring
- Per-report invalidation

## Usage

### Basic Setup

```go
// Create manager
manager := reports.NewManager(logger)

// Register report modules
costReport := costs.NewCostReport(costService)
manager.Register(costReport)

rdsReport := rds.NewRDSReport(dbService)
manager.Register(rdsReport)
```

### Generate Dashboard Summary

```go
params := reports.ReportParams{
    StartTime: &startTime,
    EndTime:   &endTime,
    UseCache:  true,
}

summaries, err := manager.GenerateSummary(ctx, params)
if err != nil {
    return err
}

// Use summaries in dashboard template
```

### Generate Detailed Report

```go
params := reports.ReportParams{
    StartTime:    &startTime,
    EndTime:      &endTime,
    Applications: []string{"publishing-api"},
    UseCache:     true,
}

data, err := manager.GenerateReport(ctx, "costs", params)
if err != nil {
    return err
}

// Use data for detailed report view
```

## Report Module Interface

All report modules must implement the `Report` interface:

```go
type Report interface {
    GetMetadata() ReportMetadata
    GenerateSummary(ctx context.Context, params ReportParams) ([]Summary, error)
    GenerateReport(ctx context.Context, params ReportParams) (ReportData, error)
    IsAvailable(ctx context.Context) bool
    GetRefreshInterval() time.Duration
    Validate(params ReportParams) error
}
```

### Implementation Example

```go
type CostReport struct {
    service *CostService
    logger  *logger.Logger
}

func (r *CostReport) GetMetadata() reports.ReportMetadata {
    return reports.ReportMetadata{
        ID:          "costs",
        Name:        "Cost Analysis",
        Description: "AWS cost tracking and analysis",
        Type:        reports.ReportTypeCost,
        Version:     "1.0.0",
        Priority:    reports.PriorityHigh,
    }
}

func (r *CostReport) GenerateSummary(ctx context.Context, params reports.ReportParams) ([]reports.Summary, error) {
    // Implementation here
}
```

## Data Flow

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   HTTP Request  │───▶│   Manager    │───▶│  Report Module  │
└─────────────────┘    └──────────────┘    └─────────────────┘
                              │                      │
                              ▼                      ▼
                       ┌──────────────┐    ┌─────────────────┐
                       │    Cache     │    │ Data Services   │
                       └──────────────┘    └─────────────────┘
                              │                      │
                              ▼                      ▼
                       ┌──────────────┐    ┌─────────────────┐
                       │   Renderer   │    │  Raw Data       │
                       └──────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────┐
                       │   Response   │
                       └──────────────┘
```

## Configuration

Reports can be configured through environment variables or configuration files:

```go
type Config struct {
    Reports struct {
        CacheEnabled   bool          `json:"cache_enabled"`
        DefaultTTL     time.Duration `json:"default_ttl"`
        CleanupPeriod  time.Duration `json:"cleanup_period"`
        MaxCacheSize   int           `json:"max_cache_size"`
    } `json:"reports"`
}
```

## Testing

The framework provides utilities for testing report modules:

```go
func TestCostReportSummary(t *testing.T) {
    report := costs.NewCostReport(mockService)
    
    params := reports.ReportParams{
        StartTime: &startTime,
        EndTime:   &endTime,
    }
    
    summaries, err := report.GenerateSummary(context.Background(), params)
    assert.NoError(t, err)
    assert.NotEmpty(t, summaries)
}
```

## Performance Considerations

- **Caching**: Use caching for expensive operations
- **Streaming**: Consider streaming for large datasets
- **Pagination**: Implement pagination for large result sets
- **Async**: Use background processing for long-running reports
- **Monitoring**: Track performance metrics and cache hit rates
# Report Modules

This directory contains individual report modules that implement the reports framework.

## Architecture

Each module is a self-contained package that implements the `reports.Report` interface defined in `internal/reports/types.go`. This allows for:

- **Modularity**: Each report type is isolated and can be developed independently
- **Extensibility**: New report types can be added without modifying existing code
- **Flexibility**: Modules can be enabled/disabled based on configuration
- **Testing**: Each module can be unit tested in isolation

## Module Structure

Each module should follow this standard structure:

```
module_name/
├── README.md          # Module documentation
├── report.go          # Main report implementation
├── service.go         # Business logic and data fetching
├── models.go          # Module-specific data structures
├── handlers.go        # HTTP endpoints (if needed)
└── *_test.go          # Unit tests
```

## Available Modules

### 📊 Cost Reports (`costs/`)
- AWS cost analysis and trending
- Service cost breakdowns
- Budget tracking and alerts
- **Status**: 🚧 Placeholder

### 🗄️ RDS Reports (`rds/`)
- Database performance metrics
- Connection pool statistics
- Query performance analysis
- **Status**: 🚧 Placeholder

## Creating a New Module

1. Create a new directory under `internal/modules/`
2. Implement the `reports.Report` interface in `report.go`
3. Register the module in the main application setup
4. Add tests and documentation

## Interface Requirements

All modules must implement:

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

## Registration

Modules are registered with the reports manager during application startup:

```go
manager := reports.NewManager(logger)
manager.Register(costs.NewCostReport(costService))
manager.Register(rds.NewRDSReport(dbService))
```
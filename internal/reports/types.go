package reports

import (
	"context"
	"time"
)

// ReportType defines the category of report
type ReportType string

const (
	ReportTypeCost        ReportType = "cost"
	ReportTypePerformance ReportType = "performance"
	ReportTypeHealth      ReportType = "health"
	ReportTypeUsage       ReportType = "usage"
	ReportTypeCustom      ReportType = "custom"
)

// ReportStatus represents the current state of a report
type ReportStatus string

const (
	StatusPending    ReportStatus = "pending"
	StatusRunning    ReportStatus = "running" 
	StatusCompleted  ReportStatus = "completed"
	StatusFailed     ReportStatus = "failed"
	StatusCached     ReportStatus = "cached"
)

// Priority defines the importance level of a report
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Summary represents high-level summary data for dashboard display
type Summary interface {
	// GetTitle returns the display title for this summary
	GetTitle() string
	
	// GetValue returns the primary value to display
	GetValue() string
	
	// GetSubtitle returns additional context information
	GetSubtitle() string
	
	// GetTrend returns trend information (e.g., "+5.2%", "-12.1%")
	GetTrend() *TrendData
	
	// GetType returns the type of summary for styling/icons
	GetType() SummaryType
	
	// IsHealthy returns whether this summary indicates a healthy state
	IsHealthy() bool
}

// TrendData represents trending information
type TrendData struct {
	Direction TrendDirection `json:"direction"`
	Value     string         `json:"value"`
	Period    string         `json:"period"`
}

// TrendDirection indicates the direction of a trend
type TrendDirection string

const (
	TrendUp    TrendDirection = "up"
	TrendDown  TrendDirection = "down"
	TrendFlat  TrendDirection = "flat"
)

// SummaryType categorizes summary cards for styling
type SummaryType string

const (
	SummaryTypeMetric    SummaryType = "metric"
	SummaryTypeCount     SummaryType = "count"
	SummaryTypeCurrency  SummaryType = "currency"
	SummaryTypeHealth    SummaryType = "health"
	SummaryTypeAlert     SummaryType = "alert"
)

// Report defines the interface that all report modules must implement
type Report interface {
	// GetMetadata returns basic information about this report
	GetMetadata() ReportMetadata
	
	// GenerateSummary creates summary data for dashboard display
	GenerateSummary(ctx context.Context, params ReportParams) ([]Summary, error)
	
	// GenerateReport creates detailed report data
	GenerateReport(ctx context.Context, params ReportParams) (ReportData, error)
	
	// IsAvailable checks if this report can run with current configuration
	IsAvailable(ctx context.Context) bool
	
	// GetRefreshInterval returns how often this report should be refreshed
	GetRefreshInterval() time.Duration
	
	// Validate checks if the provided parameters are valid for this report
	Validate(params ReportParams) error
}

// ReportMetadata contains information about a report module
type ReportMetadata struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        ReportType `json:"type"`
	Version     string     `json:"version"`
	Author      string     `json:"author"`
	Tags        []string   `json:"tags"`
	Priority    Priority   `json:"priority"`
}

// ReportParams contains parameters for report generation
type ReportParams struct {
	// Time range
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	
	// Filtering
	Applications []string               `json:"applications,omitempty"`
	Teams        []string               `json:"teams,omitempty"`
	Environments []string               `json:"environments,omitempty"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	
	// Grouping and aggregation
	GroupBy   []string `json:"group_by,omitempty"`
	SortBy    string   `json:"sort_by,omitempty"`
	SortOrder string   `json:"sort_order,omitempty"`
	
	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
	
	// Output format
	Format string `json:"format,omitempty"`
	
	// Caching
	UseCache    bool          `json:"use_cache,omitempty"`
	CacheTTL    time.Duration `json:"cache_ttl,omitempty"`
	ForceRefresh bool         `json:"force_refresh,omitempty"`
}

// ReportData represents the output of a report generation
type ReportData struct {
	Metadata    ReportMetadata  `json:"metadata"`
	Status      ReportStatus    `json:"status"`
	GeneratedAt time.Time       `json:"generated_at"`
	DataPoints  []DataPoint     `json:"data_points"`
	Summary     []Summary       `json:"summary"`
	Charts      []ChartData     `json:"charts,omitempty"`
	Tables      []TableData     `json:"tables,omitempty"`
	Errors      []ReportError   `json:"errors,omitempty"`
	Warnings    []ReportWarning `json:"warnings,omitempty"`
}

// DataPoint represents a single data measurement
type DataPoint struct {
	Timestamp  time.Time              `json:"timestamp"`
	Labels     map[string]string      `json:"labels"`
	Values     map[string]interface{} `json:"values"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ChartData represents data formatted for chart visualization
type ChartData struct {
	Title       string                   `json:"title"`
	Type        string                   `json:"type"` // bar, line, pie, etc.
	XAxis       string                   `json:"x_axis"`
	YAxis       string                   `json:"y_axis"`
	Series      []ChartSeries            `json:"series"`
	Options     map[string]interface{}   `json:"options,omitempty"`
}

// ChartSeries represents a data series in a chart
type ChartSeries struct {
	Name   string                 `json:"name"`
	Data   []ChartPoint           `json:"data"`
	Style  map[string]interface{} `json:"style,omitempty"`
}

// ChartPoint represents a single point in a chart series
type ChartPoint struct {
	X interface{} `json:"x"`
	Y interface{} `json:"y"`
}

// TableData represents tabular data
type TableData struct {
	Title   string                `json:"title"`
	Headers []TableHeader         `json:"headers"`
	Rows    []map[string]interface{} `json:"rows"`
	Footer  map[string]interface{} `json:"footer,omitempty"`
}

// TableHeader defines a table column
type TableHeader struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"` // string, number, currency, date, etc.
	Sortable    bool   `json:"sortable"`
	Filterable  bool   `json:"filterable"`
}

// ReportError represents an error that occurred during report generation
type ReportError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ReportWarning represents a warning during report generation
type ReportWarning struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
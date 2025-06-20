package rds

import (
	"time"
)

// PostgreSQLInstance represents a PostgreSQL RDS instance
type PostgreSQLInstance struct {
	InstanceID         string    `json:"instance_id"`
	Name               string    `json:"name"`
	Version            string    `json:"version"`
	MajorVersion       string    `json:"major_version"`
	Status             string    `json:"status"`
	IsEOL              bool      `json:"is_eol"`
	EOLDate            *time.Time `json:"eol_date,omitempty"`
	Application        string    `json:"application,omitempty"`
	Environment        string    `json:"environment,omitempty"`
	Engine             string    `json:"engine"`
	InstanceClass      string    `json:"instance_class"`
	AllocatedStorage   int32     `json:"allocated_storage"`
	StorageType        string    `json:"storage_type"`
	MultiAZ            bool      `json:"multi_az"`
	PubliclyAccessible bool      `json:"publicly_accessible"`
	Region             string    `json:"region"`
	AvailabilityZone   string    `json:"availability_zone"`
	CreatedAt          time.Time `json:"created_at"`
	LastModified       time.Time `json:"last_modified"`
}

// VersionInfo represents PostgreSQL version information
type VersionInfo struct {
	MajorVersion string     `json:"major_version"`
	FullVersion  string     `json:"full_version"`
	IsSupported  bool       `json:"is_supported"`
	IsEOL        bool       `json:"is_eol"`
	EOLDate      *time.Time `json:"eol_date,omitempty"`
	ReleaseDate  time.Time  `json:"release_date"`
	SupportEnds  *time.Time `json:"support_ends,omitempty"`
}

// InstancesSummary represents a summary of RDS instances
type InstancesSummary struct {
	TotalInstances    int                   `json:"total_instances"`
	PostgreSQLCount   int                   `json:"postgresql_count"`
	EOLInstances      int                   `json:"eol_instances"`
	OutdatedInstances int                   `json:"outdated_instances"`
	Instances         []PostgreSQLInstance  `json:"instances"`
	VersionSummary    []VersionSummaryItem  `json:"version_summary"`
	LastUpdated       time.Time             `json:"last_updated"`
}

// VersionSummaryItem represents a summary for a specific version
type VersionSummaryItem struct {
	MajorVersion string `json:"major_version"`
	Count        int    `json:"count"`
	IsEOL        bool   `json:"is_eol"`
	IsOutdated   bool   `json:"is_outdated"`
}

// OutdatedInstancesResponse represents instances that need version updates
type OutdatedInstancesResponse struct {
	OutdatedInstances []PostgreSQLInstance `json:"outdated_instances"`
	EOLInstances      []PostgreSQLInstance `json:"eol_instances"`
	Count             int                  `json:"count"`
	LastChecked       time.Time            `json:"last_checked"`
}

// VersionCheckResult represents the result of checking instance versions
type VersionCheckResult struct {
	InstanceID       string     `json:"instance_id"`
	CurrentVersion   string     `json:"current_version"`
	MajorVersion     string     `json:"major_version"`
	IsEOL            bool       `json:"is_eol"`
	IsOutdated       bool       `json:"is_outdated"`
	RecommendedAction string    `json:"recommended_action"`
	EOLDate          *time.Time `json:"eol_date,omitempty"`
	LatestInMajor    string     `json:"latest_in_major,omitempty"`
}

// PostgreSQLVersions contains EOL and support information for PostgreSQL versions
type PostgreSQLVersions struct {
	Versions map[string]VersionInfo `json:"versions"`
	Current  string                 `json:"current_stable"`
	EOL      []string               `json:"eol_versions"`
}

// Alert represents a version-related alert
type Alert struct {
	Type        AlertType `json:"type"`
	Severity    Severity  `json:"severity"`
	InstanceID  string    `json:"instance_id"`
	Message     string    `json:"message"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeEOL         AlertType = "eol"
	AlertTypeOutdated    AlertType = "outdated"
	AlertTypeDeprecated  AlertType = "deprecated"
	AlertTypeUpcoming    AlertType = "upcoming_eol"
)

// Severity represents the severity level of an alert
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// RDSMetrics represents performance and operational metrics
type RDSMetrics struct {
	InstanceID           string    `json:"instance_id"`
	CPUUtilization       float64   `json:"cpu_utilization"`
	DatabaseConnections  int32     `json:"database_connections"`
	FreeableMemory       int64     `json:"freeable_memory"`
	FreeStorageSpace     int64     `json:"free_storage_space"`
	ReadIOPS             float64   `json:"read_iops"`
	WriteIOPS            float64   `json:"write_iops"`
	ReadLatency          float64   `json:"read_latency"`
	WriteLatency         float64   `json:"write_latency"`
	Timestamp            time.Time `json:"timestamp"`
}
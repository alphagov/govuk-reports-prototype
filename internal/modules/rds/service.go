package rds

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/pkg/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// RDSService handles PostgreSQL instance discovery and version checking
type RDSService struct {
	client  *rds.Client
	config  *config.Config
	logger  *logger.Logger
	eolData PostgreSQLVersions
}

// NewRDSService creates a new RDS service instance
func NewRDSService(awsConfig aws.Config, cfg *config.Config, log *logger.Logger) *RDSService {
	client := rds.NewFromConfig(awsConfig)
	
	service := &RDSService{
		client: client,
		config: cfg,
		logger: log,
		eolData: getPostgreSQLVersionData(),
	}
	
	return service
}

// GetAllInstances discovers all PostgreSQL RDS instances
func (s *RDSService) GetAllInstances(ctx context.Context) (*InstancesSummary, error) {
	s.logger.Info().Msg("Discovering PostgreSQL RDS instances")

	// Get all DB instances
	input := &rds.DescribeDBInstancesInput{}
	
	var allInstances []PostgreSQLInstance
	paginator := rds.NewDescribeDBInstancesPaginator(s.client, input)
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.WithError(err).Error().Msg("Failed to describe RDS instances")
			return nil, fmt.Errorf("failed to describe RDS instances: %w", err)
		}

		// Filter and process PostgreSQL instances
		for _, dbInstance := range page.DBInstances {
			if s.isPostgreSQL(dbInstance) {
				instance := s.convertToPostgreSQLInstance(dbInstance)
				instance = s.enrichWithVersionInfo(instance)
				allInstances = append(allInstances, instance)
			}
		}
	}

	// Generate summary
	summary := s.generateInstancesSummary(allInstances)
	
	s.logger.WithFields(map[string]interface{}{
		"total_instances":    summary.TotalInstances,
		"postgresql_count":   summary.PostgreSQLCount,
		"eol_instances":      summary.EOLInstances,
		"outdated_instances": summary.OutdatedInstances,
	}).Info().Msg("PostgreSQL instances discovered")

	return summary, nil
}

// GetOutdatedInstances returns instances that need version updates
func (s *RDSService) GetOutdatedInstances(ctx context.Context) (*OutdatedInstancesResponse, error) {
	s.logger.Info().Msg("Checking for outdated PostgreSQL instances")

	summary, err := s.GetAllInstances(ctx)
	if err != nil {
		return nil, err
	}

	var outdatedInstances []PostgreSQLInstance
	var eolInstances []PostgreSQLInstance

	for _, instance := range summary.Instances {
		if instance.IsEOL {
			eolInstances = append(eolInstances, instance)
		} else if s.isOutdated(instance) {
			outdatedInstances = append(outdatedInstances, instance)
		}
	}

	response := &OutdatedInstancesResponse{
		OutdatedInstances: outdatedInstances,
		EOLInstances:      eolInstances,
		Count:             len(outdatedInstances) + len(eolInstances),
		LastChecked:       time.Now(),
	}

	return response, nil
}

// GetVersionCheckResults performs version checking for all instances
func (s *RDSService) GetVersionCheckResults(ctx context.Context) ([]VersionCheckResult, error) {
	summary, err := s.GetAllInstances(ctx)
	if err != nil {
		return nil, err
	}

	var results []VersionCheckResult
	for _, instance := range summary.Instances {
		result := s.checkInstanceVersion(instance)
		results = append(results, result)
	}

	return results, nil
}

// GetInstanceByID retrieves a specific PostgreSQL instance
func (s *RDSService) GetInstanceByID(ctx context.Context, instanceID string) (*PostgreSQLInstance, error) {
	s.logger.WithField("instance_id", instanceID).Info().Msg("Getting PostgreSQL instance details")

	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	result, err := s.client.DescribeDBInstances(ctx, input)
	if err != nil {
		s.logger.WithError(err).Error().Msg("Failed to describe RDS instance")
		return nil, fmt.Errorf("failed to describe RDS instance: %w", err)
	}

	if len(result.DBInstances) == 0 {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	dbInstance := result.DBInstances[0]
	if !s.isPostgreSQL(dbInstance) {
		return nil, fmt.Errorf("instance is not PostgreSQL: %s", instanceID)
	}

	instance := s.convertToPostgreSQLInstance(dbInstance)
	instance = s.enrichWithVersionInfo(instance)

	return &instance, nil
}

// Helper methods

// isPostgreSQL checks if the DB instance is PostgreSQL
func (s *RDSService) isPostgreSQL(dbInstance types.DBInstance) bool {
	if dbInstance.Engine == nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(*dbInstance.Engine), "postgres")
}

// convertToPostgreSQLInstance converts AWS RDS instance to our model
func (s *RDSService) convertToPostgreSQLInstance(dbInstance types.DBInstance) PostgreSQLInstance {
	instance := PostgreSQLInstance{
		InstanceID:       aws.ToString(dbInstance.DBInstanceIdentifier),
		Name:             aws.ToString(dbInstance.DBName),
		Version:          aws.ToString(dbInstance.EngineVersion),
		Engine:           aws.ToString(dbInstance.Engine),
		InstanceClass:    aws.ToString(dbInstance.DBInstanceClass),
		Status:           aws.ToString(dbInstance.DBInstanceStatus),
		Region:           aws.ToString(dbInstance.AvailabilityZone), // Will extract region from AZ
		AvailabilityZone: aws.ToString(dbInstance.AvailabilityZone),
		MultiAZ:          aws.ToBool(dbInstance.MultiAZ),
	}

	// Extract region from availability zone (e.g., "us-east-1a" -> "us-east-1")
	if instance.AvailabilityZone != "" {
		re := regexp.MustCompile(`^([a-z0-9-]+)-[a-z]$`)
		if matches := re.FindStringSubmatch(instance.AvailabilityZone); len(matches) > 1 {
			instance.Region = matches[1]
		}
	}

	// Set default name if empty
	if instance.Name == "" {
		instance.Name = instance.InstanceID
	}

	// Extract major version
	instance.MajorVersion = s.extractMajorVersion(instance.Version)

	// Set timestamps
	if dbInstance.InstanceCreateTime != nil {
		instance.CreatedAt = *dbInstance.InstanceCreateTime
	}

	// Try to extract application and environment from tags or instance name
	instance.Application, instance.Environment = s.extractApplicationInfo(instance.InstanceID)

	// Set other fields
	if dbInstance.AllocatedStorage != nil {
		instance.AllocatedStorage = *dbInstance.AllocatedStorage
	}
	if dbInstance.StorageType != nil {
		instance.StorageType = *dbInstance.StorageType
	}
	if dbInstance.PubliclyAccessible != nil {
		instance.PubliclyAccessible = *dbInstance.PubliclyAccessible
	}

	instance.LastModified = time.Now()

	return instance
}

// enrichWithVersionInfo adds EOL and version information
func (s *RDSService) enrichWithVersionInfo(instance PostgreSQLInstance) PostgreSQLInstance {
	versionInfo, exists := s.eolData.Versions[instance.MajorVersion]
	if exists {
		instance.IsEOL = versionInfo.IsEOL
		instance.EOLDate = versionInfo.EOLDate
	} else {
		// If version not in our data, consider it potentially EOL if very old
		majorVersionNum, err := strconv.Atoi(instance.MajorVersion)
		if err == nil && majorVersionNum < 12 {
			instance.IsEOL = true
		}
	}

	return instance
}

// extractMajorVersion extracts major version from full version string
func (s *RDSService) extractMajorVersion(version string) string {
	// PostgreSQL versions like "14.9", "13.13", "12.17"
	re := regexp.MustCompile(`^(\d+)\.`)
	if matches := re.FindStringSubmatch(version); len(matches) > 1 {
		return matches[1]
	}
	
	// Fallback: try to extract just the number
	re = regexp.MustCompile(`^(\d+)`)
	if matches := re.FindStringSubmatch(version); len(matches) > 1 {
		return matches[1]
	}
	
	return version
}

// extractApplicationInfo tries to extract application and environment from instance identifier
func (s *RDSService) extractApplicationInfo(instanceID string) (string, string) {
	// Common patterns: app-env-db, app-db-env, govuk-app-env
	parts := strings.Split(strings.ToLower(instanceID), "-")
	
	var application, environment string
	
	// Look for common environment indicators
	envKeywords := map[string]string{
		"prod":        "production",
		"production":  "production",
		"staging":     "staging",
		"stage":       "staging",
		"test":        "test",
		"testing":     "test",
		"dev":         "development",
		"development": "development",
		"demo":        "demo",
	}
	
	for _, part := range parts {
		if env, isEnv := envKeywords[part]; isEnv {
			environment = env
		} else if part != "db" && part != "database" && part != "postgres" && part != "postgresql" && part != "govuk" {
			if application == "" {
				application = part
			}
		}
	}
	
	// If no environment found, default to production
	if environment == "" {
		environment = "production"
	}
	
	return application, environment
}

// generateInstancesSummary creates a summary of all instances
func (s *RDSService) generateInstancesSummary(instances []PostgreSQLInstance) *InstancesSummary {
	summary := &InstancesSummary{
		TotalInstances:  len(instances),
		PostgreSQLCount: len(instances),
		Instances:       instances,
		LastUpdated:     time.Now(),
	}

	// Count EOL and outdated instances
	versionCounts := make(map[string]int)
	
	for _, instance := range instances {
		if instance.IsEOL {
			summary.EOLInstances++
		}
		if s.isOutdated(instance) {
			summary.OutdatedInstances++
		}
		versionCounts[instance.MajorVersion]++
	}

	// Generate version summary
	for version, count := range versionCounts {
		versionInfo, exists := s.eolData.Versions[version]
		isEOL := exists && versionInfo.IsEOL
		isOutdated := exists && !versionInfo.IsSupported
		
		summary.VersionSummary = append(summary.VersionSummary, VersionSummaryItem{
			MajorVersion: version,
			Count:        count,
			IsEOL:        isEOL,
			IsOutdated:   isOutdated,
		})
	}

	return summary
}

// isOutdated checks if an instance version is outdated but not EOL
func (s *RDSService) isOutdated(instance PostgreSQLInstance) bool {
	if instance.IsEOL {
		return false // EOL is handled separately
	}
	
	versionInfo, exists := s.eolData.Versions[instance.MajorVersion]
	if !exists {
		return true // Unknown version, consider outdated
	}
	
	return !versionInfo.IsSupported
}

// checkInstanceVersion performs version checking for a single instance
func (s *RDSService) checkInstanceVersion(instance PostgreSQLInstance) VersionCheckResult {
	result := VersionCheckResult{
		InstanceID:     instance.InstanceID,
		CurrentVersion: instance.Version,
		MajorVersion:   instance.MajorVersion,
		IsEOL:          instance.IsEOL,
		IsOutdated:     s.isOutdated(instance),
		EOLDate:        instance.EOLDate,
	}

	// Determine recommended action
	if result.IsEOL {
		result.RecommendedAction = "Critical: Upgrade immediately - version is end-of-life"
	} else if result.IsOutdated {
		result.RecommendedAction = "Upgrade recommended - newer stable version available"
	} else {
		result.RecommendedAction = "No action needed - version is current"
	}

	// Get latest version in major release
	if versionInfo, exists := s.eolData.Versions[instance.MajorVersion]; exists {
		result.LatestInMajor = versionInfo.FullVersion
	}

	return result
}

// getPostgreSQLVersionData returns PostgreSQL version EOL data
func getPostgreSQLVersionData() PostgreSQLVersions {
	now := time.Now()
	
	// PostgreSQL version data based on official EOL schedule
	// Reference: https://www.postgresql.org/support/versioning/
	versions := map[string]VersionInfo{
		"16": {
			MajorVersion: "16",
			FullVersion:  "16.1",
			IsSupported:  true,
			IsEOL:        false,
			ReleaseDate:  time.Date(2023, 9, 14, 0, 0, 0, 0, time.UTC),
			SupportEnds:  timePtr(time.Date(2028, 11, 9, 0, 0, 0, 0, time.UTC)),
		},
		"15": {
			MajorVersion: "15",
			FullVersion:  "15.5",
			IsSupported:  true,
			IsEOL:        false,
			ReleaseDate:  time.Date(2022, 10, 13, 0, 0, 0, 0, time.UTC),
			SupportEnds:  timePtr(time.Date(2027, 11, 11, 0, 0, 0, 0, time.UTC)),
		},
		"14": {
			MajorVersion: "14",
			FullVersion:  "14.10",
			IsSupported:  true,
			IsEOL:        false,
			ReleaseDate:  time.Date(2021, 9, 30, 0, 0, 0, 0, time.UTC),
			SupportEnds:  timePtr(time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)),
		},
		"13": {
			MajorVersion: "13",
			FullVersion:  "13.13",
			IsSupported:  true,
			IsEOL:        false,
			ReleaseDate:  time.Date(2020, 9, 24, 0, 0, 0, 0, time.UTC),
			SupportEnds:  timePtr(time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)),
		},
		"12": {
			MajorVersion: "12",
			FullVersion:  "12.17",
			IsSupported:  true,
			IsEOL:        false,
			ReleaseDate:  time.Date(2019, 10, 3, 0, 0, 0, 0, time.UTC),
			SupportEnds:  timePtr(time.Date(2024, 11, 14, 0, 0, 0, 0, time.UTC)),
		},
		"11": {
			MajorVersion: "11",
			FullVersion:  "11.22",
			IsSupported:  false,
			IsEOL:        true,
			ReleaseDate:  time.Date(2018, 10, 18, 0, 0, 0, 0, time.UTC),
			EOLDate:      timePtr(time.Date(2023, 11, 9, 0, 0, 0, 0, time.UTC)),
		},
		"10": {
			MajorVersion: "10",
			FullVersion:  "10.23",
			IsSupported:  false,
			IsEOL:        true,
			ReleaseDate:  time.Date(2017, 10, 5, 0, 0, 0, 0, time.UTC),
			EOLDate:      timePtr(time.Date(2022, 11, 10, 0, 0, 0, 0, time.UTC)),
		},
		"9.6": {
			MajorVersion: "9.6",
			FullVersion:  "9.6.24",
			IsSupported:  false,
			IsEOL:        true,
			ReleaseDate:  time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC),
			EOLDate:      timePtr(time.Date(2021, 11, 11, 0, 0, 0, 0, time.UTC)),
		},
	}

	// Update IsEOL based on current date
	for version, info := range versions {
		if info.EOLDate != nil && now.After(*info.EOLDate) {
			info.IsEOL = true
			info.IsSupported = false
			versions[version] = info
		}
	}

	return PostgreSQLVersions{
		Versions: versions,
		Current:  "16",
		EOL:      []string{"9.6", "10", "11"},
	}
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
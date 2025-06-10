package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server     ServerConfig
	AWS        AWSConfig
	GOVUK      GOVUKConfig
	Log        LogConfig
	Cache      CacheConfig
	Monitoring MonitoringConfig
}

type ServerConfig struct {
	Port         string
	Host         string
	Environment  string
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
	TLSEnabled   bool
	CertFile     string
	KeyFile      string
}

type AWSConfig struct {
	Region             string
	AccessKeyID        string
	SecretAccessKey    string
	SessionToken       string
	Profile            string
	MFAToken           string
	CostExplorerRegion string
	MaxRetries         int
	RetryDelay         time.Duration
}

type GOVUKConfig struct {
	APIBaseURL      string
	APIKey          string
	AppsAPITimeout  time.Duration
	AppsAPICacheTTL time.Duration
	AppsAPIRetries  int
	RateLimit       int
	UserAgent       string
}

type LogConfig struct {
	Level  string
	Format string
	Output string
}

type CacheConfig struct {
	DefaultTTL     time.Duration
	CleanupPeriod  time.Duration
	MaxSize        int
	EvictionPolicy string
}

type MonitoringConfig struct {
	MetricsEnabled bool
	MetricsPort    string
	HealthPath     string
	ReadyzPath     string
	LivezPath      string
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error for %s: %s", e.Field, e.Message)
}

// Load loads and validates configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", ""),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  getEnvAsInt("READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("WRITE_TIMEOUT", 30),
			IdleTimeout:  getEnvAsInt("IDLE_TIMEOUT", 120),
			TLSEnabled:   getEnvAsBool("TLS_ENABLED", false),
			CertFile:     getEnv("TLS_CERT_FILE", ""),
			KeyFile:      getEnv("TLS_KEY_FILE", ""),
		},
		AWS: AWSConfig{
			Region:             getEnv("AWS_REGION", "eu-west-2"),
			AccessKeyID:        getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:    getEnv("AWS_SECRET_ACCESS_KEY", ""),
			SessionToken:       getEnv("AWS_SESSION_TOKEN", ""),
			Profile:            getEnv("AWS_PROFILE", ""),
			MFAToken:           getEnv("AWS_MFA_TOKEN", ""),
			CostExplorerRegion: getEnv("AWS_COST_EXPLORER_REGION", "us-east-1"),
			MaxRetries:         getEnvAsInt("AWS_MAX_RETRIES", 3),
			RetryDelay:         getEnvAsDuration("AWS_RETRY_DELAY", 1*time.Second),
		},
		GOVUK: GOVUKConfig{
			APIBaseURL:      getEnv("GOVUK_API_BASE_URL", "https://www.gov.uk/api"),
			APIKey:          getEnv("GOVUK_API_KEY", ""),
			AppsAPITimeout:  getEnvAsDuration("GOVUK_APPS_API_TIMEOUT", 30*time.Second),
			AppsAPICacheTTL: getEnvAsDuration("GOVUK_APPS_API_CACHE_TTL", 15*time.Minute),
			AppsAPIRetries:  getEnvAsInt("GOVUK_APPS_API_RETRIES", 3),
			RateLimit:       getEnvAsInt("GOVUK_RATE_LIMIT", 100),
			UserAgent:       getEnv("GOVUK_USER_AGENT", "GOV.UK-Cost-Dashboard/1.0"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
		Cache: CacheConfig{
			DefaultTTL:     getEnvAsDuration("CACHE_DEFAULT_TTL", 10*time.Minute),
			CleanupPeriod:  getEnvAsDuration("CACHE_CLEANUP_PERIOD", 5*time.Minute),
			MaxSize:        getEnvAsInt("CACHE_MAX_SIZE", 1000),
			EvictionPolicy: getEnv("CACHE_EVICTION_POLICY", "LRU"),
		},
		Monitoring: MonitoringConfig{
			MetricsEnabled: getEnvAsBool("METRICS_ENABLED", true),
			MetricsPort:    getEnv("METRICS_PORT", "9090"),
			HealthPath:     getEnv("HEALTH_PATH", "/api/health"),
			ReadyzPath:     getEnv("READYZ_PATH", "/api/readyz"),
			LivezPath:      getEnv("LIVEZ_PATH", "/api/livez"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	var errors []ValidationError

	// Server validation
	if c.Server.Port == "" {
		errors = append(errors, ValidationError{"server.port", "port cannot be empty"})
	}
	
	if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
		errors = append(errors, ValidationError{"server.port", "port must be a valid number between 1 and 65535"})
	}

	validEnvs := []string{"development", "staging", "production"}
	if !contains(validEnvs, c.Server.Environment) {
		errors = append(errors, ValidationError{"server.environment", "environment must be one of: development, staging, production"})
	}

	if c.Server.ReadTimeout < 1 || c.Server.ReadTimeout > 300 {
		errors = append(errors, ValidationError{"server.read_timeout", "read timeout must be between 1 and 300 seconds"})
	}

	if c.Server.WriteTimeout < 1 || c.Server.WriteTimeout > 300 {
		errors = append(errors, ValidationError{"server.write_timeout", "write timeout must be between 1 and 300 seconds"})
	}

	if c.Server.TLSEnabled {
		if c.Server.CertFile == "" {
			errors = append(errors, ValidationError{"server.cert_file", "TLS cert file path required when TLS is enabled"})
		}
		if c.Server.KeyFile == "" {
			errors = append(errors, ValidationError{"server.key_file", "TLS key file path required when TLS is enabled"})
		}
	}

	// AWS validation
	if c.AWS.Region == "" {
		errors = append(errors, ValidationError{"aws.region", "AWS region cannot be empty"})
	}

	// Check if AWS credentials are provided in some form (allow for demo mode)
	hasProfile := c.AWS.Profile != ""
	hasAccessKeys := c.AWS.AccessKeyID != "" && c.AWS.SecretAccessKey != ""
	isDevelopment := c.Server.Environment == "development"
	
	if !hasProfile && !hasAccessKeys && !isDevelopment {
		errors = append(errors, ValidationError{"aws.credentials", "either AWS profile or access keys must be provided in production/staging environments"})
	}

	if c.AWS.MaxRetries < 0 || c.AWS.MaxRetries > 10 {
		errors = append(errors, ValidationError{"aws.max_retries", "max retries must be between 0 and 10"})
	}

	// GOVUK validation
	if c.GOVUK.APIBaseURL == "" {
		errors = append(errors, ValidationError{"govuk.api_base_url", "GOVUK API base URL cannot be empty"})
	}

	if c.GOVUK.AppsAPITimeout < 1*time.Second || c.GOVUK.AppsAPITimeout > 5*time.Minute {
		errors = append(errors, ValidationError{"govuk.apps_api_timeout", "API timeout must be between 1 second and 5 minutes"})
	}

	if c.GOVUK.AppsAPIRetries < 0 || c.GOVUK.AppsAPIRetries > 10 {
		errors = append(errors, ValidationError{"govuk.apps_api_retries", "API retries must be between 0 and 10"})
	}

	if c.GOVUK.RateLimit < 1 || c.GOVUK.RateLimit > 10000 {
		errors = append(errors, ValidationError{"govuk.rate_limit", "rate limit must be between 1 and 10000 requests per minute"})
	}

	// Log validation
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	if !contains(validLogLevels, strings.ToLower(c.Log.Level)) {
		errors = append(errors, ValidationError{"log.level", "log level must be one of: trace, debug, info, warn, error, fatal, panic"})
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, c.Log.Format) {
		errors = append(errors, ValidationError{"log.format", "log format must be 'json' or 'text'"})
	}

	// Cache validation
	if c.Cache.MaxSize < 1 || c.Cache.MaxSize > 100000 {
		errors = append(errors, ValidationError{"cache.max_size", "cache max size must be between 1 and 100000"})
	}

	validEvictionPolicies := []string{"LRU", "LFU", "FIFO"}
	if !contains(validEvictionPolicies, c.Cache.EvictionPolicy) {
		errors = append(errors, ValidationError{"cache.eviction_policy", "eviction policy must be one of: LRU, LFU, FIFO"})
	}

	// Monitoring validation
	if c.Monitoring.MetricsEnabled {
		if port, err := strconv.Atoi(c.Monitoring.MetricsPort); err != nil || port < 1 || port > 65535 {
			errors = append(errors, ValidationError{"monitoring.metrics_port", "metrics port must be a valid number between 1 and 65535"})
		}
	}

	if len(errors) > 0 {
		return &ConfigValidationError{Errors: errors}
	}

	return nil
}

// ConfigValidationError wraps multiple validation errors
type ConfigValidationError struct {
	Errors []ValidationError
}

func (e *ConfigValidationError) Error() string {
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("configuration validation failed:\n%s", strings.Join(messages, "\n"))
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// GetBindAddress returns the full bind address for the server
func (c *Config) GetBindAddress() string {
	if c.Server.Host != "" {
		return c.Server.Host + ":" + c.Server.Port
	}
	return ":" + c.Server.Port
}

// Helper functions

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valueStr := strings.ToLower(getEnv(key, ""))
	if valueStr == "true" || valueStr == "1" || valueStr == "yes" || valueStr == "on" {
		return true
	} else if valueStr == "false" || valueStr == "0" || valueStr == "no" || valueStr == "off" {
		return false
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
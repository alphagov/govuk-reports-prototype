package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server ServerConfig
	AWS    AWSConfig
	GOVUK  GOVUKConfig
	Log    LogConfig
}

type ServerConfig struct {
	Port         string
	Environment  string
	ReadTimeout  int
	WriteTimeout int
}

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Profile         string
	MFAToken        string
}

type GOVUKConfig struct {
	APIBaseURL      string
	APIKey          string
	AppsAPITimeout  time.Duration
	AppsAPICacheTTL time.Duration
	AppsAPIRetries  int
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  getEnvAsInt("READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("WRITE_TIMEOUT", 30),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "eu-west-2"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			SessionToken:    getEnv("AWS_SESSION_TOKEN", ""),
			Profile:         getEnv("AWS_PROFILE", ""),
			MFAToken:        getEnv("AWS_MFA_TOKEN", ""),
		},
		GOVUK: GOVUKConfig{
			APIBaseURL:      getEnv("GOVUK_API_BASE_URL", "https://www.gov.uk/api"),
			APIKey:          getEnv("GOVUK_API_KEY", ""),
			AppsAPITimeout:  getEnvAsDuration("GOVUK_APPS_API_TIMEOUT", 30*time.Second),
			AppsAPICacheTTL: getEnvAsDuration("GOVUK_APPS_API_CACHE_TTL", 15*time.Minute),
			AppsAPIRetries:  getEnvAsInt("GOVUK_APPS_API_RETRIES", 3),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}
}

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

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultVal
}
package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Clear environment variables
	clearEnvVars()

	// Test with default values
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error with default config, got: %v", err)
	}

	// Check default values
	if cfg.Server.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Server.Port)
	}

	if cfg.Server.Environment != "development" {
		t.Errorf("Expected default environment development, got %s", cfg.Server.Environment)
	}

	if cfg.AWS.Region != "eu-west-2" {
		t.Errorf("Expected default AWS region eu-west-2, got %s", cfg.AWS.Region)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Expected default log level info, got %s", cfg.Log.Level)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorField  string
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"PORT":                    "8080",
				"ENVIRONMENT":             "production",
				"AWS_REGION":             "eu-west-1",
				"AWS_PROFILE":            "test-profile",
				"GOVUK_API_BASE_URL":     "https://api.test.gov.uk",
				"LOG_LEVEL":              "info",
				"LOG_FORMAT":             "json",
			},
			expectError: false,
		},
		{
			name: "invalid port",
			envVars: map[string]string{
				"PORT":                    "invalid",
				"AWS_PROFILE":            "test-profile",
				"GOVUK_API_BASE_URL":     "https://api.test.gov.uk",
			},
			expectError: true,
			errorField:  "server.port",
		},
		{
			name: "invalid environment",
			envVars: map[string]string{
				"PORT":                    "8080",
				"ENVIRONMENT":             "invalid",
				"AWS_PROFILE":            "test-profile",
				"GOVUK_API_BASE_URL":     "https://api.test.gov.uk",
			},
			expectError: true,
			errorField:  "server.environment",
		},
		{
			name: "missing AWS credentials",
			envVars: map[string]string{
				"PORT":                    "8080",
				"ENVIRONMENT":             "production",
				"GOVUK_API_BASE_URL":     "https://api.test.gov.uk",
			},
			expectError: true,
			errorField:  "aws.credentials",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"PORT":                    "8080",
				"AWS_PROFILE":            "test-profile",
				"GOVUK_API_BASE_URL":     "https://api.test.gov.uk",
				"LOG_LEVEL":              "invalid",
			},
			expectError: true,
			errorField:  "log.level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			clearEnvVars()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				if configErr, ok := err.(*ConfigValidationError); ok {
					found := false
					for _, validationErr := range configErr.Errors {
						if validationErr.Field == tt.errorField {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error for field %s, but didn't find it in errors: %v", tt.errorField, configErr.Errors)
					}
				} else {
					t.Errorf("Expected ConfigValidationError but got: %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				if cfg == nil {
					t.Errorf("Expected config but got nil")
				}
			}

			// Clean up
			clearEnvVars()
		})
	}
}

func TestConfigMethods(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Environment: "development",
			Host:        "localhost",
			Port:        "8080",
		},
	}

	if !cfg.IsDevelopment() {
		t.Errorf("Expected IsDevelopment() to return true")
	}

	if cfg.IsProduction() {
		t.Errorf("Expected IsProduction() to return false")
	}

	expectedAddr := "localhost:8080"
	if addr := cfg.GetBindAddress(); addr != expectedAddr {
		t.Errorf("Expected bind address %s, got %s", expectedAddr, addr)
	}

	// Test without host
	cfg.Server.Host = ""
	expectedAddr = ":8080"
	if addr := cfg.GetBindAddress(); addr != expectedAddr {
		t.Errorf("Expected bind address %s, got %s", expectedAddr, addr)
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Test getEnv
	os.Setenv("TEST_STRING", "test_value")
	if value := getEnv("TEST_STRING", "default"); value != "test_value" {
		t.Errorf("Expected test_value, got %s", value)
	}

	if value := getEnv("NON_EXISTENT", "default"); value != "default" {
		t.Errorf("Expected default, got %s", value)
	}

	// Test getEnvAsInt
	os.Setenv("TEST_INT", "42")
	if value := getEnvAsInt("TEST_INT", 0); value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}

	os.Setenv("TEST_INT_INVALID", "invalid")
	if value := getEnvAsInt("TEST_INT_INVALID", 10); value != 10 {
		t.Errorf("Expected 10, got %d", value)
	}

	// Test getEnvAsBool
	os.Setenv("TEST_BOOL_TRUE", "true")
	if value := getEnvAsBool("TEST_BOOL_TRUE", false); !value {
		t.Errorf("Expected true, got %v", value)
	}

	os.Setenv("TEST_BOOL_FALSE", "false")
	if value := getEnvAsBool("TEST_BOOL_FALSE", true); value {
		t.Errorf("Expected false, got %v", value)
	}

	os.Setenv("TEST_BOOL_ONE", "1")
	if value := getEnvAsBool("TEST_BOOL_ONE", false); !value {
		t.Errorf("Expected true, got %v", value)
	}

	// Test getEnvAsDuration
	os.Setenv("TEST_DURATION", "5m")
	expected := 5 * time.Minute
	if value := getEnvAsDuration("TEST_DURATION", time.Second); value != expected {
		t.Errorf("Expected %v, got %v", expected, value)
	}

	os.Setenv("TEST_DURATION_INVALID", "invalid")
	if value := getEnvAsDuration("TEST_DURATION_INVALID", time.Second); value != time.Second {
		t.Errorf("Expected %v, got %v", time.Second, value)
	}

	// Clean up
	clearEnvVars()
}

func clearEnvVars() {
	envVars := []string{
		"PORT", "HOST", "ENVIRONMENT", "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
		"TLS_ENABLED", "TLS_CERT_FILE", "TLS_KEY_FILE",
		"AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
		"AWS_PROFILE", "AWS_MFA_TOKEN", "AWS_COST_EXPLORER_REGION", "AWS_MAX_RETRIES", "AWS_RETRY_DELAY",
		"GOVUK_API_BASE_URL", "GOVUK_API_KEY", "GOVUK_APPS_API_TIMEOUT", "GOVUK_APPS_API_CACHE_TTL",
		"GOVUK_APPS_API_RETRIES", "GOVUK_RATE_LIMIT", "GOVUK_USER_AGENT",
		"LOG_LEVEL", "LOG_FORMAT", "LOG_OUTPUT",
		"CACHE_DEFAULT_TTL", "CACHE_CLEANUP_PERIOD", "CACHE_MAX_SIZE", "CACHE_EVICTION_POLICY",
		"METRICS_ENABLED", "METRICS_PORT", "HEALTH_PATH", "READYZ_PATH", "LIVEZ_PATH",
		"TEST_STRING", "TEST_INT", "TEST_INT_INVALID", "TEST_BOOL_TRUE", "TEST_BOOL_FALSE",
		"TEST_BOOL_ONE", "TEST_DURATION", "TEST_DURATION_INVALID",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
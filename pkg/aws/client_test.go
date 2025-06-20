package aws

import (
	"os"
	"testing"
)

func TestGetTagPrefix_Default(t *testing.T) {
	// Ensure no environment variable is set
	os.Unsetenv("GOVUK_APP_TAG_PREFIX")
	
	expected := "govuk-"
	actual := getTagPrefix()
	
	if actual != expected {
		t.Errorf("Expected default tag prefix '%s', got '%s'", expected, actual)
	}
}

func TestGetTagPrefix_CustomEnvironment(t *testing.T) {
	customPrefix := "test-prefix-"
	os.Setenv("GOVUK_APP_TAG_PREFIX", customPrefix)
	defer os.Unsetenv("GOVUK_APP_TAG_PREFIX")
	
	actual := getTagPrefix()
	
	if actual != customPrefix {
		t.Errorf("Expected custom tag prefix '%s', got '%s'", customPrefix, actual)
	}
}

func TestTagMappingPatterns(t *testing.T) {
	testCases := []struct {
		name       string
		appName    string
		prefix     string
		expected   string
	}{
		{
			name:     "Default prefix with simple app name",
			appName:  "frontend",
			prefix:   "govuk-",
			expected: "govuk-frontend",
		},
		{
			name:     "Default prefix with hyphenated app name",
			appName:  "content-store",
			prefix:   "govuk-",
			expected: "govuk-content-store",
		},
		{
			name:     "Custom prefix",
			appName:  "publishing-api",
			prefix:   "custom-",
			expected: "custom-publishing-api",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("GOVUK_APP_TAG_PREFIX", tc.prefix)
			defer os.Unsetenv("GOVUK_APP_TAG_PREFIX")
			
			tagPrefix := getTagPrefix()
			actual := tagPrefix + tc.appName
			
			if actual != tc.expected {
				t.Errorf("Expected tag '%s', got '%s'", tc.expected, actual)
			}
		})
	}
}
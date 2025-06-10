package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog with additional functionality
type Logger struct {
	zerolog.Logger
}

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	Output     string // stdout, stderr, file path
	TimeFormat string // RFC3339, Unix, etc.
	Colorize   bool   // Enable colors for console output
}

// New creates a new logger with the given configuration
func New(config Config) (*Logger, error) {
	// Set up the output writer
	var output io.Writer = os.Stdout
	if config.Output == "stderr" {
		output = os.Stderr
	} else if config.Output != "stdout" && config.Output != "" {
		// TODO: Support file output if needed
		output = os.Stdout
	}

	// Configure zerolog
	if config.Format == "console" || config.Colorize {
		// Use pretty console output with colors
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: "15:04:05",
			FormatLevel: func(i interface{}) string {
				return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprintf("%-50s", i)
			},
			FormatFieldName: func(i interface{}) string {
				return fmt.Sprintf("%s:", i)
			},
			FormatFieldValue: func(i interface{}) string {
				return fmt.Sprintf("%s", i)
			},
		}
	}

	// Set global log level
	level := parseLogLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	// Create logger
	logger := zerolog.New(output).With().Timestamp().Logger()

	// Configure time format
	if config.TimeFormat != "" {
		switch config.TimeFormat {
		case "unix":
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		case "unixms":
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
		case "unixmicro":
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
		case "rfc3339":
			zerolog.TimeFieldFormat = time.RFC3339
		case "rfc3339nano":
			zerolog.TimeFieldFormat = time.RFC3339Nano
		}
	}

	return &Logger{Logger: logger}, nil
}

// parseLogLevel converts string level to zerolog level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	event := l.Logger.With()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	return &Logger{Logger: event.Logger()}
}

// WithField adds a single field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{Logger: l.Logger.With().Interface(key, value).Logger()}
}

// WithError adds an error field to the logger context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.Logger.With().Err(err).Logger()}
}

// HTTP request logging helpers
func (l *Logger) LogHTTPRequest(method, path string, statusCode int, latency time.Duration, clientIP string, bodySize int) {
	var level zerolog.Level
	if statusCode >= 500 {
		level = zerolog.ErrorLevel
	} else if statusCode >= 400 {
		level = zerolog.WarnLevel
	} else {
		level = zerolog.InfoLevel
	}

	l.WithLevel(level).
		Str("method", method).
		Str("path", path).
		Int("status_code", statusCode).
		Dur("latency", latency).
		Str("client_ip", clientIP).
		Int("body_size", bodySize).
		Bool("slow_request", latency > 5*time.Second).
		Msg("HTTP Request")
}

// Application-specific logging helpers
func (l *Logger) LogApplicationCost(appName string, cost float64, team string, platform string) {
	l.Info().
		Str("app_name", appName).
		Float64("cost", cost).
		Str("team", team).
		Str("platform", platform).
		Msg("Application cost calculated")
}

func (l *Logger) LogAPICall(service string, endpoint string, duration time.Duration, success bool) {
	event := l.Info()
	if !success {
		event = l.Error()
	}
	
	event.
		Str("service", service).
		Str("endpoint", endpoint).
		Dur("duration", duration).
		Bool("success", success).
		Msg("External API call")
}

func (l *Logger) LogCacheOperation(operation string, key string, hit bool, ttl time.Duration) {
	l.Debug().
		Str("operation", operation).
		Str("cache_key", key).
		Bool("cache_hit", hit).
		Dur("ttl", ttl).
		Msg("Cache operation")
}

// Security logging
func (l *Logger) LogSecurityEvent(event string, clientIP string, userAgent string, details map[string]interface{}) {
	logEvent := l.Warn().
		Str("security_event", event).
		Str("client_ip", clientIP).
		Str("user_agent", userAgent)
	
	for k, v := range details {
		logEvent = logEvent.Interface(k, v)
	}
	
	logEvent.Msg("Security event detected")
}

// Performance logging
func (l *Logger) LogPerformance(operation string, duration time.Duration, metadata map[string]interface{}) {
	event := l.Info()
	if duration > 1*time.Second {
		event = l.Warn()
	}
	
	logEvent := event.
		Str("operation", operation).
		Dur("duration", duration).
		Bool("slow_operation", duration > 1*time.Second)
	
	for k, v := range metadata {
		logEvent = logEvent.Interface(k, v)
	}
	
	logEvent.Msg("Performance metric")
}

// Startup and shutdown logging
func (l *Logger) LogStartup(component string, version string, config map[string]interface{}) {
	logEvent := l.Info().
		Str("component", component).
		Str("version", version)
	
	for k, v := range config {
		logEvent = logEvent.Interface(k, v)
	}
	
	logEvent.Msg("Component started")
}

func (l *Logger) LogShutdown(component string, duration time.Duration) {
	l.Info().
		Str("component", component).
		Dur("shutdown_duration", duration).
		Msg("Component shutdown completed")
}

// GetZerologLogger returns the underlying zerolog logger for direct use
func (l *Logger) GetZerologLogger() zerolog.Logger {
	return l.Logger
}

// SetGlobalLogger sets this logger as the global zerolog logger
func (l *Logger) SetGlobalLogger() {
	log.Logger = l.Logger
}
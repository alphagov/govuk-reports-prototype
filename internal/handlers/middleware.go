package handlers

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"govuk-cost-dashboard/internal/config"
	"govuk-cost-dashboard/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler provides comprehensive error handling with proper logging
func ErrorHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovery := recover(); recovery != nil {
				// Log panic with stack trace
				logger.WithFields(logrus.Fields{
					"panic":      recovery,
					"stack":      string(debug.Stack()),
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"client_ip":  c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
				}).Error("Panic recovered")

				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "internal_server_error",
					Message: "An unexpected error occurred",
					Code:    http.StatusInternalServerError,
				})
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// Log the error
			logger.WithFields(logrus.Fields{
				"error":      err.Error(),
				"type":       err.Type,
				"path":       c.Request.URL.Path,
				"method":     c.Request.Method,
				"client_ip":  c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			}).Error("Request error")

			switch err.Type {
			case gin.ErrorTypeBind:
				c.JSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "bad_request",
					Message: sanitizeErrorMessage(err.Error()),
					Code:    http.StatusBadRequest,
				})
			case gin.ErrorTypePublic:
				c.JSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "validation_error",
					Message: sanitizeErrorMessage(err.Error()),
					Code:    http.StatusBadRequest,
				})
			default:
				// Check if it's a context timeout
				if errors.Is(err.Err, context.DeadlineExceeded) {
					c.JSON(http.StatusRequestTimeout, models.ErrorResponse{
						Error:   "request_timeout",
						Message: "Request timed out. Please try again later.",
						Code:    http.StatusRequestTimeout,
					})
				} else {
					c.JSON(http.StatusInternalServerError, models.ErrorResponse{
						Error:   "internal_server_error",
						Message: "An unexpected error occurred",
						Code:    http.StatusInternalServerError,
					})
				}
			}
		}
	}
}

// TimeoutMiddleware adds request timeout handling
func TimeoutMiddleware(timeout time.Duration, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			defer close(finished)
			c.Next()
		}()

		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				logger.WithFields(logrus.Fields{
					"path":      c.Request.URL.Path,
					"method":    c.Request.Method,
					"client_ip": c.ClientIP(),
					"timeout":   timeout,
				}).Warn("Request timeout")

				c.JSON(http.StatusRequestTimeout, models.ErrorResponse{
					Error:   "request_timeout",
					Message: "Request timed out. Please try again later.",
					Code:    http.StatusRequestTimeout,
				})
				c.Abort()
			}
		case <-finished:
			// Request completed normally
		}
	}
}

// RateLimitMiddleware provides basic rate limiting (simplified implementation)
func RateLimitMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health checks
		if strings.HasPrefix(c.Request.URL.Path, "/api/health") ||
		   strings.HasPrefix(c.Request.URL.Path, "/api/readyz") ||
		   strings.HasPrefix(c.Request.URL.Path, "/api/livez") {
			c.Next()
			return
		}

		// For demo purposes, we'll use a simple client IP based approach
		clientIP := c.ClientIP()
		
		// In production, you'd integrate with Redis or similar
		// For now, we'll just log potential abuse
		userAgent := c.Request.UserAgent()
		if userAgent == "" || strings.Contains(strings.ToLower(userAgent), "bot") {
			logger.WithFields(logrus.Fields{
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"path":       c.Request.URL.Path,
			}).Info("Potential bot traffic detected")
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'")
		
		c.Next()
	}
}

// LoggerMiddleware provides structured request logging
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}

		// Determine log level based on status code
		logLevel := logrus.InfoLevel
		if statusCode >= 400 && statusCode < 500 {
			logLevel = logrus.WarnLevel
		} else if statusCode >= 500 {
			logLevel = logrus.ErrorLevel
		}

		fields := logrus.Fields{
			"status_code": statusCode,
			"latency_ms":  latency.Milliseconds(),
			"client_ip":   clientIP,
			"method":      method,
			"path":        path,
			"body_size":   bodySize,
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Log slow requests
		if latency > 5*time.Second {
			fields["slow_request"] = true
		}

		logger.WithFields(fields).Log(logLevel, "HTTP Request")
	}
}

// CORSMiddleware provides configurable CORS handling
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// In production, restrict origins
		if cfg.IsProduction() {
			allowedOrigins := []string{
				"https://gov.uk",
				"https://*.gov.uk",
				"https://publishing.service.gov.uk",
			}
			
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin || 
				   (strings.Contains(allowedOrigin, "*") && 
				    strings.HasSuffix(origin, strings.TrimPrefix(allowedOrigin, "*"))) {
					allowed = true
					break
				}
			}
			
			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		} else {
			// Development mode - allow all origins
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// HealthCheckMiddleware provides circuit breaker functionality for health checks
func HealthCheckMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for actual health check endpoints
		if strings.HasPrefix(c.Request.URL.Path, "/api/health") ||
		   strings.HasPrefix(c.Request.URL.Path, "/api/readyz") ||
		   strings.HasPrefix(c.Request.URL.Path, "/api/livez") {
			c.Next()
			return
		}

		// For other endpoints, we could implement circuit breaker logic here
		// For now, just continue
		c.Next()
	}
}

// MetricsMiddleware collects basic metrics
func MetricsMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		
		// Log metrics for monitoring systems to pick up
		logger.WithFields(logrus.Fields{
			"metric_type":    "http_request",
			"method":         c.Request.Method,
			"path":          c.Request.URL.Path,
			"status_code":   c.Writer.Status(),
			"duration_ms":   duration.Milliseconds(),
			"response_size": c.Writer.Size(),
		}).Debug("HTTP metrics")
	}
}

// Helper functions

// sanitizeErrorMessage removes sensitive information from error messages
func sanitizeErrorMessage(message string) string {
	// Remove potential sensitive data patterns
	sensitivePatterns := []string{
		"password",
		"token",
		"key",
		"secret",
		"credential",
	}
	
	lowerMessage := strings.ToLower(message)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerMessage, pattern) {
			return "Invalid request parameters"
		}
	}
	
	return message
}
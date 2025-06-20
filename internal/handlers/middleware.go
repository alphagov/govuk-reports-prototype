package handlers

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/internal/models"
	"govuk-reports-dashboard/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ErrorHandler provides comprehensive error handling with proper logging
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovery := recover(); recovery != nil {
				// Log panic with stack trace
				log.Error().
					Interface("panic", recovery).
					Str("stack", string(debug.Stack())).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Str("client_ip", c.ClientIP()).
					Str("user_agent", c.Request.UserAgent()).
					Msg("Panic recovered")

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
			log.Error().
				Str("error", err.Error()).
				Interface("type", err.Type).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("client_ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Request error")

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
func TimeoutMiddleware(timeout time.Duration, log *logger.Logger) gin.HandlerFunc {
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
				log.Warn().
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Str("client_ip", c.ClientIP()).
					Dur("timeout", timeout).
					Msg("Request timeout")

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
func RateLimitMiddleware(log *logger.Logger) gin.HandlerFunc {
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
			log.LogSecurityEvent("potential_bot_traffic", clientIP, userAgent, map[string]interface{}{
				"path": c.Request.URL.Path,
			})
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
func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
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

		// Use the optimized HTTP request logging helper
		log.LogHTTPRequest(method, path, statusCode, latency, clientIP, bodySize)
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
func HealthCheckMiddleware(log *logger.Logger) gin.HandlerFunc {
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
func MetricsMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		
		// Log metrics for monitoring systems to pick up
		log.LogPerformance("http_request", duration, map[string]interface{}{
			"method":        c.Request.Method,
			"path":         c.Request.URL.Path,
			"status_code":  c.Writer.Status(),
			"response_size": c.Writer.Size(),
		})
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
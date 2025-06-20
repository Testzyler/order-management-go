package middleware

import (
	"time"

	"github.com/Testzyler/order-management-go/infrastructure/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware adds a unique request ID to each request for Fiber
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set response header
		c.Set(RequestIDHeader, requestID)

		// Store request ID in locals for easy access
		c.Locals("request_id", requestID)

		return c.Next()
	}
}

// LoggingMiddleware logs HTTP requests with structured logging for Fiber
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get request ID from locals
		requestID, _ := c.Locals("request_id").(string)
		if requestID == "" {
			requestID = "unknown"
		}

		// Add request ID to the Fiber context so it can be passed to handlers
		ctx := logger.WithRequestIDToContext(c.Context(), requestID)
		c.SetUserContext(ctx)

		// Create logger with request context and additional metadata
		requestLogger := logger.GetDefault().WithFields(map[string]interface{}{
			"request_id":     requestID,
			"method":         c.Method(),
			"path":           c.Path(),
			"original_url":   c.OriginalURL(),
			"remote_ip":      c.IP(),
			"user_agent":     c.Get("User-Agent"),
			"content_type":   c.Get("Content-Type"),
			"content_length": c.Get("Content-Length"),
		})

		// Store logger in locals for handlers to use
		c.Locals("logger", requestLogger)

		// Log request start with detailed information
		requestLogger.Info("HTTP request started",
			"query_params", string(c.Request().URI().QueryString()),
			"headers", getSelectedHeaders(c),
		)

		// Process request
		err := c.Next()

		// Log request completion with comprehensive details
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()
		responseSize := len(c.Response().Body())

		// Enhanced request completion logging
		logger.LogHTTPRequest(requestLogger, c.Method(), c.Path(), statusCode, duration, requestID)

		requestLogger.WithFields(map[string]interface{}{
			"status_code":   statusCode,
			"duration_ms":   duration.Milliseconds(),
			"duration_ns":   duration.Nanoseconds(),
			"response_size": responseSize,
			"error":         err,
		}).Info("HTTP request completed")

		// Log error details if present
		if err != nil {
			requestLogger.WithError(err).Error("HTTP request failed")
		}

		return err
	}
}

// getSelectedHeaders extracts important headers for logging (excluding sensitive ones)
func getSelectedHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)

	// Safe headers to log
	safeHeaders := []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Cache-Control",
		"Connection",
		"Content-Type",
		"Origin",
		"Referer",
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Request-ID",
	}

	for _, header := range safeHeaders {
		if value := c.Get(header); value != "" {
			headers[header] = value
		}
	}

	return headers
}

// RecoveryMiddleware recovers from panics and logs them for Fiber
func RecoveryMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if err := recover(); err != nil {
				requestLogger := GetLoggerFromFiberContext(c)

				// Enhanced panic logging with stack trace information
				requestLogger.WithFields(map[string]interface{}{
					"panic":        err,
					"method":       c.Method(),
					"path":         c.Path(),
					"original_url": c.OriginalURL(),
					"remote_ip":    c.IP(),
					"user_agent":   c.Get("User-Agent"),
					"query_params": string(c.Request().URI().QueryString()),
				}).Error("Panic recovered in HTTP handler")

				// Log the panic details
				logger.Error("Application panic recovered",
					"error", err,
					"request_id", c.Locals("request_id"),
					"path", c.Path(),
					"method", c.Method(),
				)

				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":      "Internal Server Error",
					"request_id": c.Locals("request_id"),
				})
			}
		}()
		return c.Next()
	}
}

// GetLoggerFromFiberContext retrieves logger from Fiber context
func GetLoggerFromFiberContext(c *fiber.Ctx) *logger.Logger {
	if logger, ok := c.Locals("logger").(*logger.Logger); ok {
		return logger
	}
	// Fallback: create logger with request ID if available
	if requestID, ok := c.Locals("request_id").(string); ok {
		return logger.WithRequestID(requestID)
	}
	return logger.GetDefault()
}

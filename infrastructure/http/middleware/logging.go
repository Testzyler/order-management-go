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
		
		// Create logger with request context
		requestLogger := logger.GetDefault().WithFields(map[string]interface{}{
			"request_id": requestID,
			"method":     c.Method(),
			"path":       c.Path(),
			"remote_ip":  c.IP(),
			"user_agent": c.Get("User-Agent"),
		})

		// Store logger in locals for handlers to use
		c.Locals("logger", requestLogger)

		// Log request start
		requestLogger.Info("HTTP request started")

		// Process request
		err := c.Next()

		// Log request completion
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()
		
		logger.LogHTTPRequest(requestLogger, c.Method(), c.Path(), statusCode, duration, requestID)

		return err
	}
}

// RecoveryMiddleware recovers from panics and logs them for Fiber
func RecoveryMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if err := recover(); err != nil {
				requestLogger := GetLoggerFromFiberContext(c)
				requestLogger.WithField("panic", err).Error("Panic recovered")
				
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal Server Error",
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

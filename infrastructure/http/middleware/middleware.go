package middleware

import (
	"context"
	"time"

	"github.com/Testzyler/order-management-go/infrastructure/utils/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// ContextMiddleware adds context with timeout and cancellation support to each request
func ContextMiddleware(parentCtx context.Context) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create a new context with timeout for each request
		timeoutDuration := 30 * time.Second // Default request timeout
		if c.Get("X-Timeout-Duration") != "" {
			if duration, err := time.ParseDuration(c.Get("X-Timeout-Duration")); err == nil {
				timeoutDuration = duration
			}
		}

		ctx, cancel := context.WithTimeout(parentCtx, timeoutDuration)
		defer cancel()

		c.SetUserContext(ctx)

		c.Locals("context_cancel", cancel)

		return c.Next()
	}
}

// TimeoutMiddleware creates a middleware that enforces request timeout
func TimeoutMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
		defer cancel()

		c.SetUserContext(ctx)

		return c.Next()
	}
}

// CancellationMiddleware checks for context cancellation
func CancellationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.UserContext().Err(); err != nil {
			if err == context.Canceled {
				return fiber.NewError(499, "Request was cancelled")
			} else if err == context.DeadlineExceeded {
				return fiber.NewError(fiber.StatusRequestTimeout, "Request timeout exceeded")
			}
		}

		return c.Next()
	}
}

// RecoveryMiddleware handles panics and returns a 500 error
func RecoveryMiddleware() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
	})
}

// RequestIDMiddleware adds a unique request ID to each request for Fiber
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDHeader, requestID)

		c.Locals("request_id", requestID)

		ctx := logger.WithRequestIDToContext(c.UserContext(), requestID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// LoggingMiddleware logs HTTP requests with structured logging for Fiber
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		requestID, _ := c.Locals("request_id").(string)
		if requestID == "" {
			requestID = "unknown"
		}

		requestLogger := logger.GetDefault().WithFields(map[string]interface{}{
			"request_id": requestID,
			"method":     c.Method(),
			"path":       c.Path(),
			"user_agent": c.Get("User-Agent"),
			"remote_ip":  c.IP(),
			"referer":    c.Get("Referer"),
		})

		err := c.Next()

		duration := time.Since(start)

		status := c.Response().StatusCode()

		logFields := map[string]interface{}{
			"status":      status,
			"duration_ms": duration.Milliseconds(),
			"size":        len(c.Response().Body()),
		}

		if err != nil {
			logFields["error"] = err.Error()
			requestLogger.WithFields(logFields).Error("Request completed with error")
		} else if status >= 500 {
			requestLogger.WithFields(logFields).Error("Request completed with server error")
		} else if status >= 400 {
			requestLogger.WithFields(logFields).Warn("Request completed with client error")
		} else {
			requestLogger.WithFields(logFields).Info("Request completed successfully")
		}

		return err
	}
}

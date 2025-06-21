package api

import (
	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/Testzyler/order-management-go/infrastructure/utils/logger"
	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Initialize implements HandlerInitializer interface
func (h *HealthHandler) Initialize() {
	// No initialization needed for health check
}

// GetRouteDefinition implements HandlerInitializer interface
func (h *HealthHandler) GetRouteDefinition() route.RouteDefinition {
	return route.RouteDefinition{
		Routes: route.Routes{
			route.Route{
				Name:        "HealthCheck",
				Path:        "/healthz",
				Method:      constants.METHOD_GET,
				HandlerFunc: h.HealthCheck,
			},
		},
		Prefix: "",
	}
}

func init() {
	route.RegisterHandler(NewHealthHandler())
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context())

	requestLogger.Debug("Health check requested")

	response := fiber.Map{
		"status":  "OK",
		"message": "Service is healthy",
	}

	requestLogger.Info("Health check completed successfully")
	return c.JSON(response)
}

package route

import (
	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
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
func (h *HealthHandler) GetRouteDefinition() RouteDefinition {
	return RouteDefinition{
		Routes: Routes{
			Route{
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
	RegisterHandler(NewHealthHandler())
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("health-handler")

	requestLogger.Debug("Health check requested")

	response := fiber.Map{
		"status":  "OK",
		"message": "Service is healthy",
	}

	requestLogger.Info("Health check completed successfully")
	return c.JSON(response)
}

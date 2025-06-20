package route

import (
	"github.com/Testzyler/order-management-go/application/constants"
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
		Prefix: "", // No prefix - this will be at root level
	}
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "OK",
		"message": "Service is healthy",
	})
}

// Auto-register the health handler
func init() {
	RegisterHandler(NewHealthHandler())
}

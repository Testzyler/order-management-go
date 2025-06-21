package v1

import (
	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	// No service needed for health check
}

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
		Prefix: "", // No prefix - this will be at root level
	}
}

// Auto-register the handler
func init() {
	route.RegisterHandler(NewHealthHandler())
}

// HealthCheck handler
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "OK",
		"message": "Service is healthy",
	})
}

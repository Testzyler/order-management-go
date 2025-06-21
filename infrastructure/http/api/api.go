package api

import (
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	v1 "github.com/Testzyler/order-management-go/infrastructure/http/api/v1"
	"github.com/gofiber/fiber/v2"
)

// Add v1 prefixed routes
func AddRoute(router *fiber.Router) {
	v1Route := (*router).Group("/v1")
	v1.AddRoute(&v1Route)
}

// Add root level routes (no prefix)
func AddRootRoutes(router *fiber.Router) {
	route.AddRoutesPrefix(router)
}

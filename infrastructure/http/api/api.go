package api

import (
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	v1 "github.com/Testzyler/order-management-go/infrastructure/http/api/v1"
	"github.com/gofiber/fiber/v2"
)

func AddRoute(router *fiber.Router) {
	// Add v1 prefixed routes
	v1Route := (*router).Group("/v1")
	v1.AddRoute(&v1Route)
}

func AddRootRoutes(router *fiber.Router) {
	// Add root level routes (no prefix)
	route.AddRoutesPrefix(router)
}

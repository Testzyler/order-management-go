package api

import (
	v1 "github.com/Testzyler/order-management-go/infrastructure/http/api/v1"
	"github.com/gofiber/fiber/v2"
)

func AddRoute(router *fiber.Router) {
	v1Route := (*router).Group("/v1")
	v1.AddRoute(&v1Route)
}

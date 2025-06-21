package v1

import (
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/gofiber/fiber/v2"
)

func AddRoute(router *fiber.Router) {
	route.AddRoutesPrefix(router)
}

package http

import (
	"github.com/gofiber/fiber/v2"
)

func Healthz(c *fiber.Ctx) error {
	return c.SendString("OK")
}

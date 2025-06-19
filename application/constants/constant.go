package constants

import "github.com/gofiber/fiber/v2"

type HandlerFunc func(*fiber.Ctx) error

const (
	METHOD_GET    = "GET"
	METHOD_POST   = "POST"
	METHOD_PUT    = "PUT"
	METHOD_DELETE = "DELETE"
	METHOD_PATCH  = "PATCH"
	METHOD_ALL    = "ALL"
)

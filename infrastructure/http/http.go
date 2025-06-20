package http

import (
	"log"
	"time"

	"github.com/Testzyler/order-management-go/infrastructure/http/api"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

var AppServer *fiber.App

func InitHttpServer() {

	// Initialize all handlers first (after database is ready)
	route.InitializeAllHandlers()

	// Config Port and Address
	httpPort := viper.GetString("HttpServer.Port")
	AppServer = fiber.New(fiber.Config{
		Prefork: true,
		ReadBufferSize:  1024 * 1024, // 1MB
		WriteBufferSize: 1024 * 1024, // 1MB
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		
	})

	// Add Api Path (includes health check now)
	apiGroup := AppServer.Group("/api")
	api.AddRoute(&apiGroup)

	// Add health check at root level
	AppServer.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "OK",
			"message": "Service is healthy",
		})
	})

	// Start Server
	log.Printf("serving http at http://127.0.0.1:%s", httpPort)
	err := AppServer.Listen(":" + httpPort)
	if err != nil {
		log.Fatal(err)
	}
}

func ShutdownHttpServer() {
	log.Println("http server is shutting down")
	if err := AppServer.Shutdown(); err != nil {
		log.Printf("http server shut down failed: %s", err)
		return
	}
	log.Println("http server shut down completed")
}

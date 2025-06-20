package http

import (
	"time"

	"github.com/Testzyler/order-management-go/infrastructure/http/api"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/Testzyler/order-management-go/infrastructure/http/middleware"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

var AppServer *fiber.App

func InitHttpServer() {
	httpLogger := logger.WithComponent("http-server")
	httpLogger.Info("Initializing HTTP server")

	// Initialize all handlers first (after database is ready)
	route.InitializeAllHandlers()

	// Config Port and Address
	httpPort := viper.GetString("HttpServer.Port")
	AppServer = fiber.New(fiber.Config{
		DisableStartupMessage: true,        // Disable the startup banner
		ReadBufferSize:        1024 * 1024, // 1MB
		WriteBufferSize:       1024 * 1024, // 1MB
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
	})

	// Add middleware
	AppServer.Use(middleware.RecoveryMiddleware())
	AppServer.Use(middleware.RequestIDMiddleware())
	AppServer.Use(middleware.LoggingMiddleware())

	// Add Api Path (includes health check now)
	apiGroup := AppServer.Group("/api")
	api.AddRoute(&apiGroup)

	// Add health check at root level
	// AppServer.Get("/healthz", func(c *fiber.Ctx) error {
	// 	return c.JSON(fiber.Map{
	// 		"status":  "OK",
	// 		"message": "Service is healthy",
	// 	})
	// })

	// Start Server
	httpLogger.Info("Starting HTTP server", "port", httpPort, "address", "127.0.0.1")
	err := AppServer.Listen(":" + httpPort)
	if err != nil {
		httpLogger.Error("Failed to start HTTP server", "error", err)
		logger.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func ShutdownHttpServer() {
	logger.Info("http server is shutting down")
	if err := AppServer.Shutdown(); err != nil {
		logger.Errorf("http server shut down failed: %s", err)
		return
	}
	logger.Info("http server shut down completed")
}

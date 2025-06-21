package http

import (
	"context"
	"time"

	"github.com/Testzyler/order-management-go/infrastructure/http/api"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/Testzyler/order-management-go/infrastructure/http/middleware"
	"github.com/Testzyler/order-management-go/infrastructure/utils/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

var AppServer *fiber.App

func InitHttpServer(ctx context.Context) {
	httpLogger := logger.GetDefault()
	httpLogger.Info("Initializing HTTP server")

	// Initialize all handlers first (after database is ready)
	route.InitializeAllHandlers()

	// Config Port and Address
	httpPort := viper.GetString("HttpServer.Port")
	readTimeout := viper.GetDuration("HttpServer.ServerTimeout")
	writeTimeout := viper.GetDuration("HttpServer.ServerTimeout")
	idleTimeout := viper.GetDuration("HttpServer.IdleTimeout")
	requestTimeout := viper.GetDuration("HttpServer.RequestTimeout")

	httpLogger.Info("HTTP server configuration",
		"port", httpPort,
		"read_timeout", readTimeout,
		"write_timeout", writeTimeout,
		"idle_timeout", idleTimeout,
		"request_timeout", requestTimeout,
	)

	// Set defaults if not configured
	if readTimeout == 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}
	if requestTimeout == 0 {
		requestTimeout = 30 * time.Second
	}

	AppServer = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ReadTimeout:           readTimeout,
		WriteTimeout:          writeTimeout,
		IdleTimeout:           idleTimeout,
	})

	AppServer.Use(middleware.ContextMiddleware(ctx))
	AppServer.Use(middleware.CancellationMiddleware())
	AppServer.Use(middleware.TimeoutMiddleware(requestTimeout))
	AppServer.Use(middleware.RequestIDMiddleware())
	AppServer.Use(middleware.RecoveryMiddleware())

	// Add root level routes (like /healthz) directly to AppServer
	baseRouter := AppServer.Group("")
	api.AddRootRoutes(&baseRouter)

	// Add API routes under /api prefix
	apiGroup := AppServer.Group("/api")
	api.AddRoute(&apiGroup)

	// Start Server in goroutine
	go func() {
		httpLogger.Info("Started HTTP server", "port", httpPort, "address", "127.0.0.1")
		err := AppServer.Listen(":" + httpPort)
		if err != nil {
			httpLogger.Error("Failed to start HTTP server", "error", err)
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	httpLogger.Info("Context cancelled, shutting down HTTP server")
}

func ShutdownHttpServer() {
	logger := logger.GetDefault()
	logger.Info("HTTP server is shutting down")

	shutdownTimeout := viper.GetDuration("HttpServer.ShutdownTimeout")
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- AppServer.Shutdown()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error("HTTP server shutdown failed", "error", err)
			return
		}
		logger.Info("HTTP server shutdown completed")
	case <-ctx.Done():
		logger.Error("HTTP server shutdown timed out", "timeout", shutdownTimeout)
		return
	}
}

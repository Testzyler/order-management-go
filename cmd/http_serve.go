package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Testzyler/order-management-go/infrastructure/database"
	"github.com/Testzyler/order-management-go/infrastructure/http"
	"github.com/Testzyler/order-management-go/infrastructure/utils/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ServeCmd = &cobra.Command{
	Use:   "http-serve",
	Short: "serve http server",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize logger first
		if err := initLogger(); err != nil {
			logger.Fatalf("Failed to initialize logger: %v", err)
		}

		appLogger := logger.GetDefault()
		appLogger.Info("Starting order management application")

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize services
		initPostgresql()
		initHttpServer(ctx)

		appLogger.Info("All services initialized successfully")

		// Wait for interrupt signal to gracefully shut down the server
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-quit:
			appLogger.Info("Received shutdown signal")
		case <-ctx.Done():
			appLogger.Info("Application context cancelled")
		}

		appLogger.Info("Shutting down server...")

		// Cancel the main context to signal all services to stop
		cancel()

		// Create shutdown context with timeout
		shutdownTimeout := viper.GetDuration("HttpServer.ShutdownTimeout")
		if shutdownTimeout == 0 {
			shutdownTimeout = 30 * time.Second
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()

		// Shutdown services with timeout
		shutdownDone := make(chan struct{})
		go func() {
			defer close(shutdownDone)
			shutdownHttpServer()
			shutdownPostgresql()
			wg.Wait()
		}()

		select {
		case <-shutdownDone:
			appLogger.Info("Server gracefully stopped")
		case <-shutdownCtx.Done():
			appLogger.Error("Shutdown timed out, forcing exit")
		}
	},
}

func init() {
	rootCmd.AddCommand(ServeCmd)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath("./config")
		viper.SetConfigName("config")
	}

	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// Use fmt.Printf here since logger isn't initialized yet
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Verify database configuration
	if !viper.IsSet("Database.Username") || !viper.IsSet("Database.Password") {
		fmt.Println("Database configuration is missing or incomplete")
		os.Exit(1)
	}
}

func initLogger() error {
	var loggerConfig logger.LoggerConfig
	if err := viper.UnmarshalKey("Logger", &loggerConfig); err != nil {
		return fmt.Errorf("failed to unmarshal logger config: %w", err)
	}

	// Set defaults if not provided
	if loggerConfig.Level == "" {
		loggerConfig.Level = "info"
	}
	if loggerConfig.Format == "" {
		loggerConfig.Format = "json"
	}
	if loggerConfig.Output == "" {
		loggerConfig.Output = "stdout"
	}

	return logger.Initialize(loggerConfig)
}

func initHttpServer(ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		http.InitHttpServer(ctx)
	}()
}

func shutdownHttpServer() {
	http.ShutdownHttpServer()
}

func initPostgresql() {
	database.NewDatabaseConnection()
}

func shutdownPostgresql() {
	if database.DatabasePool != nil {
		if err := database.ShutdownDatabase(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing database connection: %v\n", err)
		}
	}
}

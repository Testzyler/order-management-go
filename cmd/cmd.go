package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	infrastructure "github.com/Testzyler/order-management-go/infrastructure/database"
	"github.com/Testzyler/order-management-go/infrastructure/http"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var wg sync.WaitGroup
var configFile string

var rootCmd = &cobra.Command{
	Use:   "order-cli",
	Short: "Order management CLI app",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve http server",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize logger first
		if err := initLogger(); err != nil {
			logger.Fatalf("Failed to initialize logger: %v", err)
		}

		appLogger := logger.WithComponent("main")
		appLogger.Info("Starting order management application")

		// Initialize services
		initPostgresql()
		initHttpServer()

		appLogger.Info("All services initialized successfully")

		// Wait for interrupt signal to gracefully shut down the server
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		appLogger.Info("Shutting down server...")

		// Shutdown services
		shutdownHttpServer()
		shutdownPostgresql()

		wg.Wait()

		appLogger.Info("Server gracefully stopped")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Warnf("Error executing root command: %v", err)
		os.Exit(1)
	}
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath("./config")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logger.Infof("Using config file: %s", viper.ConfigFileUsed())
	} else {
		logger.Error("Error reading config file: %v", err)
		os.Exit(1)
	}

	// Verify database configuration
	if !viper.IsSet("Database.Username") || !viper.IsSet("Database.Password") {
		logger.Warn("Database configuration is missing or incomplete")
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

func initHttpServer() {
	wg.Add(1)
	go func() {
		defer wg.Done()
		http.InitHttpServer()
	}()
}

func shutdownHttpServer() {
	http.ShutdownHttpServer()
}

func initPostgresql() {
	infrastructure.NewDatabaseConnection()
}

func shutdownPostgresql() {
	if infrastructure.DatabasePool != nil {
		if err := infrastructure.ShutdownDatabase(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing database connection: %v\n", err)
		}
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config/config.yaml", "config file")
	rootCmd.AddCommand(serveCmd)
}

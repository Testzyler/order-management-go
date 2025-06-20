package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	infrastructure "github.com/Testzyler/order-management-go/infrastructure/database"
	"github.com/Testzyler/order-management-go/infrastructure/http"
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
		// Initialize services
		initPostgresql()
		initHttpServer()

		// Wait for interrupt signal to gracefully shut down the server
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server...")

		// Shutdown services
		shutdownHttpServer()
		shutdownPostgresql()

		wg.Wait()

		log.Println("Server gracefully stopped")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
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
		log.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		log.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Verify database configuration
	if !viper.IsSet("Database.Username") || !viper.IsSet("Database.Password") {
		log.Println("Database configuration is missing or incomplete")
		os.Exit(1)
	}
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

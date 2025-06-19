package cmd

import (
	"fmt"
	"os"
	"sync"

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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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

func initHttpServer() {
	wg.Add(1)
	go func() {
		http.InitHttpServer()
		wg.Done()
	}()
}

func shutdownHttpServer() {
	http.ShutdownHttpServer()
	wg.Done()
}

func initPostgresql() {
	wg.Add(1)
	infrastructure.NewDatabaseConnection()
}

func shutdownPostgresql() {
	if infrastructure.DatabasePool != nil {
		if err := infrastructure.ShutdownDatabase(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing database connection: %v\n", err)
		}
	}
	wg.Done()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config/config.yaml", "config file")
}

package cmd

import (
	"os"
	"sync"

	"github.com/Testzyler/order-management-go/infrastructure/logger"
	"github.com/spf13/cobra"
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
		logger.Warnf("Error executing root command: %v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config/config.yaml", "config file")
	// rootCmd.AddCommand(serveCmd)
}

package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
)

func initListenOsSignal() {
	wg.Add(1)
	go func() {
		var count int
		chanOsSignal := make(chan os.Signal, 2)
		signal.Notify(chanOsSignal, syscall.SIGTERM, os.Interrupt)

		go func() {
			// Wait Os Signal
			for getSignal := range chanOsSignal {
				// Shutdown if Interrupt or SigTerm Signal is received
				if getSignal == os.Interrupt || getSignal == syscall.SIGTERM {
					count++
					// Get Twice Signal Force Exit without Waiting Close all Components
					if count == 2 {
						logger.Info("Forcefully exiting")
						os.Exit(1)
					}

					go func() {
						shutdownPostgresql()
					}()

					go func() {
						shutdownHttpServer()
					}()

					logger.Warn("signal SIGKILL caught. shutting down")
					logger.Warn("catching SIGKILL one more time will forcefully exit")

					wg.Done()
				}
			}
			close(chanOsSignal)
		}()
	}()
}

var ServeCmd = &cobra.Command{
	Use:   "http-serve",
	Short: "Start HTTP Server for Online Order Management System",
	Run: func(cmd *cobra.Command, args []string) {
		// Init Database Connection first
		initPostgresql()

		// Init Listeners
		initListenOsSignal()

		// Init HTTP Server
		initHttpServer()

		// Waiting for Component Shut Down
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(ServeCmd)
}

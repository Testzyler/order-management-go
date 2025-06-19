package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	v1 "github.com/Testzyler/order-management-go/infrastructure/http/api/v1"
	"github.com/spf13/cobra"
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
						fmt.Println("Forcefully exiting")
						os.Exit(1)
					}

					go func() {
						shutdownPostgresql()
					}()

					go func() {
						shutdownHttpServer()
					}()

					fmt.Println("signal SIGKILL caught. shutting down")
					fmt.Println("catching SIGKILL one more time will forcefully exit")

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

		// Initialize handlers after database is ready
		v1.InitializeOrderHandler()

		// Waiting for Component Shut Down
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(ServeCmd)
}

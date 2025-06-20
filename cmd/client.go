package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Testzyler/order-management-go/application/models"
	faker "github.com/bxcodec/faker/v4"
	"github.com/spf13/cobra"
)

var ClientStressTestCmd = &cobra.Command{
	Use:   "stress-test",
	Short: "Start Stress Test for Online Order Management System API",
	Run: func(cmd *cobra.Command, args []string) {
		RunStressTest(numOrdersFlag, batchSizeFlag, concurrencyFlag, apiURLFlag)
	},
}
var (
	numOrdersFlag   int
	batchSizeFlag   int
	concurrencyFlag int
	apiURLFlag      string
	totalTimeout    = 60 * time.Second
)

func init() {
	ClientStressTestCmd.Flags().IntVar(&numOrdersFlag, "num", 1000, "Total number of orders to create")
	ClientStressTestCmd.Flags().IntVar(&batchSizeFlag, "batch", 100, "Number of orders per request batch")
	ClientStressTestCmd.Flags().IntVar(&concurrencyFlag, "concurrency", 10, "Number of concurrent requests")
	ClientStressTestCmd.Flags().StringVar(&apiURLFlag, "url", "http://localhost:3333/api/v1/orders", "Target API endpoint")
	rootCmd.AddCommand(ClientStressTestCmd)
}

func RunStressTest(numOrders, batchSize, concurrency int, apiURL string) {
	log.Println("Starting stress test for Online Order Management System API...")

	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	ordersToCreate := generateDummyOrders(numOrders)
	log.Printf("Generated %d dummy orders.", len(ordersToCreate))

	var orderBatches [][]models.CreateOrderInput
	for i := 0; i < len(ordersToCreate); i += batchSize {
		end := i + batchSize
		if end > len(ordersToCreate) {
			end = len(ordersToCreate)
		}
		orderBatches = append(orderBatches, ordersToCreate[i:end])
	}
	log.Printf("Divided orders into %d batches.", len(orderBatches))

	var wg sync.WaitGroup
	results := make(chan error, numOrders)
	sem := make(chan struct{}, concurrency)

	startTime := time.Now()

	for i, order := range ordersToCreate {
		wg.Add(1)
		sem <- struct{}{}

		go func(index int, order models.CreateOrderInput) {
			defer wg.Done()
			defer func() { <-sem }()

			log.Printf("Sending order %d...", index+1)

			reqCtx, cancel := context.WithTimeout(ctx, totalTimeout)
			defer cancel()

			err := sendBulkOrderRequest(reqCtx, order, apiURL)
			if err != nil {
				log.Printf("Error sending order %d: %v", index+1, err)
				results <- err
			} else {
				log.Printf("Successfully sent order %d.", index+1)
				results <- nil
			}
		}(i, order)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	successCount, errorCount := 0, 0
	for err := range results {
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	duration := time.Since(startTime)

	log.Printf("\n--- Stress Test Summary ---")
	log.Printf("Total Orders Sent: %d", numOrders)
	log.Printf("Successful Orders: %d", successCount)
	log.Printf("Failed Orders: %d", errorCount)
	log.Printf("Total Duration: %s", duration)
}

func generateDummyOrders(count int) []models.CreateOrderInput {
	orders := make([]models.CreateOrderInput, count)
	productNames := []string{"Widget", "Gadget", "Thingamajig", "Doodad", "Gizmo", "Contraption"}

	for i := 0; i < count; i++ {
		items := make([]models.OrderItem, rand.Intn(3)+1) // 1-3 items per order
		for j := range items {
			items[j] = models.OrderItem{
				ProductName: productNames[rand.Intn(len(productNames))],
				Quantity:    rand.Intn(5) + 1,                    // 1-5
				Price:       float64(rand.Intn(9000)+1000) / 100, // 10.00 - 99.99
			}
		}

		orders[i] = models.CreateOrderInput{
			CustomerName: faker.Name(),
			Items:        items,
		}
	}

	return orders
}

func sendBulkOrderRequest(ctx context.Context, order models.CreateOrderInput, apiURL string) error {
	payload, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal orders: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     500,
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 500,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("request cancelled or timed out: %w", ctx.Err())
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var responseBody bytes.Buffer
		responseBody.ReadFrom(resp.Body)
		return fmt.Errorf("API returned non-2xx status: %d - %s", resp.StatusCode, responseBody.String())
	}
	return nil
}

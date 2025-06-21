package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OrderItem struct {
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type CreateOrderInput struct {
	CustomerName string      `json:"customer_name"`
	Items        []OrderItem `json:"items"`
}

func main() {
	// Test context cancellation
	testContextCancellation()
}

func testContextCancellation() {
	fmt.Println("Testing context cancellation...")

	// Create a test order
	order := CreateOrderInput{
		CustomerName: "Test Customer",
		Items: []OrderItem{
			{
				ProductName: "Test Product",
				Quantity:    1,
				Price:       10.99,
			},
		},
	}

	// Marshal the order
	payload, err := json.Marshal(order)
	if err != nil {
		fmt.Printf("Failed to marshal order: %v\n", err)
		return
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:3333/api/v1/orders", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("✅ SUCCESS: Request was cancelled due to timeout as expected")
		} else if ctx.Err() == context.Canceled {
			fmt.Println("✅ SUCCESS: Request was cancelled as expected")
		} else {
			fmt.Printf("❌ Unexpected error: %v\n", err)
		}
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}

	// Check the response
	if resp.StatusCode == 499 {
		fmt.Println("✅ SUCCESS: Server returned 499 (Client Closed Request) for cancelled request")
		fmt.Printf("Response: %s\n", string(body))
	} else if resp.StatusCode == 408 {
		fmt.Println("✅ SUCCESS: Server returned 408 (Request Timeout) for cancelled request")
		fmt.Printf("Response: %s\n", string(body))
	} else if resp.StatusCode == 201 {
		fmt.Println("❌ PROBLEM: Server still returned 201 (Created) even though request should have been cancelled")
		fmt.Printf("Response: %s\n", string(body))
	} else {
		fmt.Printf("🤔 UNEXPECTED: Server returned status %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

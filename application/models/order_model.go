package models

import (
	"time"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

type Order struct {
	ID           int       `json:"id"`
	CustomerName string    `json:"customer_name"`
	TotalAmount  float64   `json:"total_amount"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateOrderInput struct {
	CustomerName string      `json:"customer_name"`
	TotalAmount  float64     `json:"total_amount"`
	Status       Status      `json:"status"`
	Items        []OrderItem `json:"items"`
}

type UpdateOrderInput struct {
	ID           int     `json:"id"`
	CustomerName string  `json:"customer_name"`
	TotalAmount  float64 `json:"total_amount"`
	Status       Status  `json:"status"`
}

type ListOrdersInput struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

type ListPaginatedOrders struct {
	Data       []Order `json:"data"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	Size       int     `json:"size"`
	TotalPages int     `json:"total_pages"`
}

type OrderItem struct {
	ID          int       `json:"id,omitempty"`
	OrderID     int       `json:"order_id"`
	ProductName string    `json:"product_name"`
	Quantity    int       `json:"quantity"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

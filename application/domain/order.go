package domain

import (
	"context"

	"github.com/Testzyler/order-management-go/application/models"
)

type OrderService interface {
	CreateOrder(ctx context.Context, order models.CreateOrderInput) error
	GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error)
	UpdateOrder(ctx context.Context, order models.UpdateOrderInput) error
	DeleteOrder(ctx context.Context, id int) error
	ListOrders(ctx context.Context, input models.ListInput) (models.ListPaginatedOrders, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) error
	GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error)
	UpdateOrder(ctx context.Context, order models.Order) error
	DeleteOrder(ctx context.Context, id int) error
	ListOrders(ctx context.Context, input models.ListInput) (*models.ListPaginatedOrders, error)
}

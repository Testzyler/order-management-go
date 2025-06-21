package services

import (
	"context"
	"errors"
	"time"

	"github.com/Testzyler/order-management-go/application/domain"
	"github.com/Testzyler/order-management-go/application/models"
	"github.com/Testzyler/order-management-go/infrastructure/utils/logger"
)

type OrderService struct {
	repo domain.OrderRepository
}

func NewOrderService(repo domain.OrderRepository) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input models.CreateOrderInput) error {
	// Use logger with request ID from context
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx)

	// Validate input
	if input.CustomerName == "" {
		serviceLogger.Error("Customer name is required")
		return errors.New("customer name is required")
	}

	if len(input.Items) == 0 {
		serviceLogger.Error("Order must have at least one item")
		return errors.New("order must have at least one item")
	}

	order := models.Order{
		CustomerName: input.CustomerName,
		Status:       models.StatusPending,
	}

	items := make([]models.OrderItem, len(input.Items))
	totalAmount := 0.0

	for i, v := range input.Items {
		if v.Quantity <= 0 {
			serviceLogger.Error("Invalid item quantity", "product", v.ProductName, "quantity", v.Quantity)
			return errors.New("item quantity must be greater than 0")
		}

		if v.Price < 0 {
			serviceLogger.Error("Invalid item price", "product", v.ProductName, "price", v.Price)
			return errors.New("item price cannot be negative")
		}

		items[i] = models.OrderItem{
			ProductName: v.ProductName,
			Quantity:    v.Quantity,
			Price:       v.Price,
		}
		itemTotal := v.Price * float64(v.Quantity)
		totalAmount += itemTotal
	}

	order.TotalAmount = totalAmount
	err := s.repo.CreateOrder(ctx, order, items)

	if err != nil {
		// Check if error is due to context cancellation or timeout
		if err == context.Canceled {
			serviceLogger.Warn("Order creation cancelled", "customer", input.CustomerName)
			return context.Canceled
		}
		if err == context.DeadlineExceeded {
			serviceLogger.Warn("Order creation timed out", "customer", input.CustomerName)
			return context.DeadlineExceeded
		}

		serviceLogger.WithError(err).Error("Failed to create order", "customer", input.CustomerName, "total", order.TotalAmount)
		return err
	}

	return nil
}

func (s *OrderService) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx)
	// Validate input
	if id <= 0 {
		serviceLogger.Error("Invalid order ID", "order_id", id)
		return models.OrderWithItems{}, errors.New("order ID must be greater than 0")
	}

	order, err := s.repo.GetOrderById(ctx, id)

	if err != nil {
		serviceLogger.WithError(err).Error("Failed to get order", "order_id", id)
		return models.OrderWithItems{}, err
	}

	if order.ID == 0 {
		serviceLogger.Warn("Order not found", "order_id", id)
		return models.OrderWithItems{}, errors.New("order not found")
	}

	return order, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, order models.UpdateOrderInput) error {
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx)
	orderToUpdate := models.Order{
		ID:        order.ID,
		Status:    order.Status,
		UpdatedAt: time.Now(),
	}

	err := s.repo.UpdateOrder(ctx, orderToUpdate)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to update order", "order_id", order.ID)
		return err
	}

	return nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, id int) error {
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx)
	err := s.repo.DeleteOrder(ctx, id)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to delete order", "order_id", id)
		return err
	}

	return nil
}

func (s *OrderService) ListOrders(ctx context.Context, input models.ListInput) (models.ListPaginatedOrders, error) {
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx)
	orders, err := s.repo.ListOrders(ctx, input)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to list orders", "page", input.Page, "size", input.Size)
		return models.ListPaginatedOrders{}, err
	}

	return *orders, nil
}

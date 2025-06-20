package services

import (
	"context"
	"errors"
	"time"

	"github.com/Testzyler/order-management-go/application/domain"
	"github.com/Testzyler/order-management-go/application/models"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
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
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-service")
	serviceLogger.Info("Creating new order", "customer_name", input.CustomerName)

	order := models.Order{
		CustomerName: input.CustomerName,
		Status:       models.StatusPending,
	}

	items := make([]models.OrderItem, len(input.Items))
	for i, v := range input.Items {
		items[i] = models.OrderItem{
			ProductName: v.ProductName,
			Quantity:    v.Quantity,
			Price:       v.Price,
		}
		order.TotalAmount += v.Price * float64(v.Quantity)
	}

	serviceLogger.WithFields(map[string]interface{}{
		"total_amount": order.TotalAmount,
		"item_count":   len(items),
	}).Debug("Calculated order totals")

	err := s.repo.CreateOrder(ctx, order, items)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to create order in repository")
		return err
	}

	serviceLogger.Info("Order created successfully in service layer")
	return nil
}

func (s *OrderService) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	// Use logger with request ID from context
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-service")
	serviceLogger.Debug("Retrieving order by ID", "order_id", id)

	order, err := s.repo.GetOrderById(ctx, id)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to retrieve order from repository", "order_id", id)
		return models.OrderWithItems{}, err
	}
	if order.ID == 0 {
		serviceLogger.Warn("Order not found", "order_id", id)
		return models.OrderWithItems{}, errors.New("order not found")
	}

	serviceLogger.Info("Order retrieved successfully", "order_id", id)
	return order, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, order models.UpdateOrderInput) error {
	// Use logger with request ID from context
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-service")
	serviceLogger.Info("Updating order", "order_id", order.ID, "new_status", order.Status)

	orderToUpdate := models.Order{
		ID:        order.ID, // Assuming ID is part of UpdateOrderInput
		Status:    order.Status,
		UpdatedAt: time.Now(),
	}
	// Assuming Items are not updated in this case, if needed, handle accordingly
	err := s.repo.UpdateOrder(ctx, orderToUpdate)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to update order in repository", "order_id", order.ID)
		return err
	}

	serviceLogger.Info("Order updated successfully", "order_id", order.ID, "status", order.Status)
	return nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, id int) error {
	// Use logger with request ID from context
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-service")
	serviceLogger.Info("Deleting order", "order_id", id)

	err := s.repo.DeleteOrder(ctx, id)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to delete order in repository", "order_id", id)
		return err
	}

	serviceLogger.Info("Order deleted successfully", "order_id", id)
	return nil
}

func (s *OrderService) ListOrders(ctx context.Context, input models.ListInput) (models.ListPaginatedOrders, error) {
	// Use logger with request ID from context
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-service")
	serviceLogger.Debug("Listing orders", "page", input.Page, "size", input.Size)

	orders, err := s.repo.ListOrders(ctx, input)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to list orders from repository", "page", input.Page, "size", input.Size)
		return models.ListPaginatedOrders{}, err
	}

	serviceLogger.Info("Orders listed successfully", "page", input.Page, "size", input.Size, "total", orders.Total)
	return *orders, nil
}

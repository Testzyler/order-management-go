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

	// Validate input
	serviceLogger.Debug("Service: Validating order input")
	if input.CustomerName == "" {
		serviceLogger.Error("Service: Customer name is required")
		return errors.New("customer name is required")
	}

	if len(input.Items) == 0 {
		serviceLogger.Error("Service: Order must have at least one item")
		return errors.New("order must have at least one item")
	}

	order := models.Order{
		CustomerName: input.CustomerName,
		Status:       models.StatusPending,
	}

	items := make([]models.OrderItem, len(input.Items))
	totalAmount := 0.0

	serviceLogger.Debug("Service: Processing order items", "item_count", len(input.Items))
	for i, v := range input.Items {
		if v.Quantity <= 0 {
			serviceLogger.Errorf("Service: Invalid item quantity: %d", v.Quantity)
			return errors.New("item quantity must be greater than 0")
		}

		if v.Price < 0 {
			serviceLogger.Errorf("Service: Invalid item price: %d", v.Price)
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
	start := time.Now()
	err := s.repo.CreateOrder(ctx, order, items)
	duration := time.Since(start)
	logger.LogServiceCall(serviceLogger, "OrderRepository", "CreateOrder", duration, logger.RequestIDFromContext(ctx))

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			serviceLogger.Warn("Service: Repository call cancelled by client",
				"repository_duration_ms", duration.Milliseconds(),
				"customer_name", input.CustomerName,
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			serviceLogger.Warn("Service: Repository call timed out",
				"repository_duration_ms", duration.Milliseconds(),
				"customer_name", input.CustomerName,
			)
			return err
		}

		serviceLogger.WithError(err).Error("Service: Repository layer failed to create order",
			"repository_duration_ms", duration.Milliseconds(),
			"customer_name", input.CustomerName,
			"total_amount", order.TotalAmount,
		)
		return err
	}

	serviceLogger.Info("Service: Order created successfully in repository",
		"repository_duration_ms", duration.Milliseconds(),
		"customer_name", input.CustomerName,
		"total_amount", order.TotalAmount,
	)
	return nil
}

func (s *OrderService) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	// Use logger with request ID from context
	serviceLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-service")
	serviceLogger.Info("Service: GetOrderById started", "order_id", id)

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		serviceLogger.Warn("Service: Request cancelled before processing", "order_id", id, "error", ctx.Err())
		return models.OrderWithItems{}, ctx.Err()
	default:
		// Continue with operation
	}

	// Validate input
	if id <= 0 {
		serviceLogger.Error("Service: Invalid order ID", "order_id", id)
		return models.OrderWithItems{}, errors.New("order ID must be greater than 0")
	}

	// Check for cancellation before calling repository
	select {
	case <-ctx.Done():
		serviceLogger.Warn("Service: Request cancelled before repository call", "order_id", id, "error", ctx.Err())
		return models.OrderWithItems{}, ctx.Err()
	default:
		// Continue with operation
	}

	serviceLogger.Debug("Service: Calling repository layer to get order", "order_id", id)
	start := time.Now()

	order, err := s.repo.GetOrderById(ctx, id)

	duration := time.Since(start)
	logger.LogServiceCall(serviceLogger, "OrderRepository", "GetOrderById", duration, logger.RequestIDFromContext(ctx))

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			serviceLogger.Warn("Service: Repository call cancelled by client",
				"order_id", id,
				"repository_duration_ms", duration.Milliseconds(),
				"error", err,
			)
			return models.OrderWithItems{}, err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			serviceLogger.Warn("Service: Repository call timed out",
				"order_id", id,
				"repository_duration_ms", duration.Milliseconds(),
				"error", err,
			)
			return models.OrderWithItems{}, err
		}

		serviceLogger.WithError(err).Error("Service: Repository layer failed to retrieve order",
			"order_id", id,
			"repository_duration_ms", duration.Milliseconds(),
		)
		return models.OrderWithItems{}, err
	}

	if order.ID == 0 {
		serviceLogger.Warn("Service: Order not found in repository",
			"order_id", id,
			"repository_duration_ms", duration.Milliseconds(),
		)
		return models.OrderWithItems{}, errors.New("order not found")
	}

	serviceLogger.Info("Service: Order retrieved successfully from repository",
		"order_id", id,
		"customer_name", order.CustomerName,
		"status", order.Status,
		"total_amount", order.TotalAmount,
		"item_count", len(order.Items),
		"repository_duration_ms", duration.Milliseconds(),
	)
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

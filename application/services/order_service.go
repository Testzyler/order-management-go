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
	serviceLogger := logger.WithComponent("order-service")
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
		"customer_name": order.CustomerName,
		"total_amount":  order.TotalAmount,
		"item_count":    len(items),
	}).Debug("Order prepared for creation")

	err := s.repo.CreateOrder(ctx, order, items)
	if err != nil {
		serviceLogger.WithError(err).Error("Failed to create order in repository")
		return err
	}

	serviceLogger.WithFields(map[string]interface{}{
		"customer_name": order.CustomerName,
		"total_amount":  order.TotalAmount,
	}).Info("Order created successfully")

	return nil
}

func (s *OrderService) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	serviceLogger := logger.WithComponent("order-service")
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
	serviceLogger := logger.WithComponent("order-service")
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
	err := s.repo.DeleteOrder(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *OrderService) ListOrders(ctx context.Context, input models.ListInput) (models.ListPaginatedOrders, error) {
	orders, err := s.repo.ListOrders(ctx, input)
	if err != nil {
		return models.ListPaginatedOrders{}, err
	}
	return *orders, nil
}

package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Testzyler/order-management-go/application/domain"
	"github.com/Testzyler/order-management-go/application/models"
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

	err := s.repo.CreateOrder(ctx, order, items)
	if err != nil {
		return err
	}

	log.Printf("Order created successfully for customer %s with total amount %.2f", input.CustomerName, order.TotalAmount)
	return nil
}

func (s *OrderService) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	order, err := s.repo.GetOrderById(ctx, id)
	if err != nil {
		return models.OrderWithItems{}, err
	}
	if order.ID == 0 {
		log.Printf("Order with ID %d not found", id)
		return models.OrderWithItems{}, errors.New("order not found")
	}

	log.Printf("Order %d retrieved successfully", id)
	return order, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, order models.UpdateOrderInput) error {
	orderToUpdate := models.Order{
		ID:        order.ID, // Assuming ID is part of UpdateOrderInput
		Status:    order.Status,
		UpdatedAt: time.Now(),
	}
	// Assuming Items are not updated in this case, if needed, handle accordingly
	err := s.repo.UpdateOrder(ctx, orderToUpdate)
	if err != nil {
		return err
	}
	log.Printf("Order %d updated successfully with status %s", order.ID, order.Status)
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

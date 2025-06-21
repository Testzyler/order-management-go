package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Testzyler/order-management-go/application/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderRepository is a mock implementation of OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) error {
	args := m.Called(ctx, order, items)
	return args.Error(0)
}

func (m *MockOrderRepository) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return models.OrderWithItems{}, args.Error(1)
	}
	return args.Get(0).(models.OrderWithItems), args.Error(1)
}

func (m *MockOrderRepository) UpdateOrder(ctx context.Context, order models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) DeleteOrder(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrderRepository) ListOrders(ctx context.Context, input models.ListInput) (*models.ListPaginatedOrders, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ListPaginatedOrders), args.Error(1)
}

func TestNewOrderService(t *testing.T) {
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	input := models.CreateOrderInput{
		CustomerName: "John Doe",
		Status:       models.StatusPending,
		Items: []models.OrderItem{
			{
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
			},
		},
	}

	ctx := context.Background()

	// Set up mock expectation
	mockRepo.On("CreateOrder", ctx, mock.AnythingOfType("models.Order"), mock.AnythingOfType("[]models.OrderItem")).Return(nil)

	// Act
	err := service.CreateOrder(ctx, input)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_CreateOrder_EmptyCustomerName(t *testing.T) {
	// Arrange
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	input := models.CreateOrderInput{
		CustomerName: "",
		Status:       models.StatusPending,
		Items: []models.OrderItem{
			{
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
			},
		},
	}

	ctx := context.Background()

	// Act
	err := service.CreateOrder(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer name is required")
	mockRepo.AssertNotCalled(t, "CreateOrder")
}

func TestOrderService_CreateOrder_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	input := models.CreateOrderInput{
		CustomerName: "John Doe",
		Status:       models.StatusPending,
		Items: []models.OrderItem{
			{
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
			},
		},
	}

	ctx := context.Background()
	repoError := errors.New("database connection failed")

	// Set up mock expectation
	mockRepo.On("CreateOrder", ctx, mock.AnythingOfType("models.Order"), mock.AnythingOfType("[]models.OrderItem")).Return(repoError)

	// Act
	err := service.CreateOrder(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, repoError, err)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_GetOrderById_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	orderID := 1
	expectedOrder := models.OrderWithItems{
		Order: models.Order{
			ID:           orderID,
			CustomerName: "John Doe",
			TotalAmount:  100.50,
			Status:       models.StatusPending,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		Items: []models.OrderItem{
			{
				ID:          1,
				OrderID:     orderID,
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	}

	ctx := context.Background()

	// Set up mock expectation
	mockRepo.On("GetOrderById", ctx, orderID).Return(expectedOrder, nil)

	// Act
	result, err := service.GetOrderById(ctx, orderID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedOrder, result)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_GetOrderById_NotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	orderID := 999
	ctx := context.Background()

	// Set up mock expectation
	mockRepo.On("GetOrderById", ctx, orderID).Return(models.OrderWithItems{}, errors.New("order not found"))

	// Act
	result, err := service.GetOrderById(ctx, orderID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, models.OrderWithItems{}, result)
	assert.Contains(t, err.Error(), "order not found")
	mockRepo.AssertExpectations(t)
}

// Benchmark tests for performance profiling
func BenchmarkOrderService_CreateOrder(b *testing.B) {
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	input := models.CreateOrderInput{
		CustomerName: "John Doe",
		Status:       models.StatusPending,
		Items: []models.OrderItem{
			{
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
			},
		},
	}

	ctx := context.Background()
	mockRepo.On("CreateOrder", ctx, mock.AnythingOfType("models.Order"), mock.AnythingOfType("[]models.OrderItem")).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.CreateOrder(ctx, input)
	}
}

func BenchmarkOrderService_GetOrderById(b *testing.B) {
	mockRepo := &MockOrderRepository{}
	service := NewOrderService(mockRepo)

	orderID := 1
	expectedOrder := models.OrderWithItems{
		Order: models.Order{
			ID:           orderID,
			CustomerName: "John Doe",
			TotalAmount:  100.50,
			Status:       models.StatusPending,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		Items: []models.OrderItem{
			{
				ID:          1,
				OrderID:     orderID,
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	}

	ctx := context.Background()
	mockRepo.On("GetOrderById", ctx, orderID).Return(expectedOrder, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetOrderById(ctx, orderID)
	}
}

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Testzyler/order-management-go/application/models"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderService is a mock implementation of OrderService
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, input models.CreateOrderInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockOrderService) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.OrderWithItems), args.Error(1)
}

func (m *MockOrderService) UpdateOrder(ctx context.Context, input models.UpdateOrderInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockOrderService) DeleteOrder(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrderService) ListOrders(ctx context.Context, input models.ListInput) (models.ListPaginatedOrders, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(models.ListPaginatedOrders), args.Error(1)
}

func TestOrderHandler_CreateOrder_Success(t *testing.T) {
	// Arrange
	mockService := &MockOrderService{}
	handler := &OrderHandler{service: mockService}

	app := fiber.New()
	app.Post("/orders", handler.CreateOrder)

	orderInput := models.CreateOrderInput{
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

	requestBody, _ := json.Marshal(orderInput)
	mockService.On("CreateOrder", mock.Anything, orderInput).Return(nil)

	// Act
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestOrderHandler_CreateOrder_BadRequest(t *testing.T) {
	// Arrange
	mockService := &MockOrderService{}
	handler := &OrderHandler{service: mockService}

	app := fiber.New()
	app.Post("/orders", handler.CreateOrder)

	// Invalid JSON
	invalidJSON := `{"customer_name": "John Doe", "invalid_field": `

	// Act
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(invalidJSON)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockService.AssertNotCalled(t, "CreateOrder")
}

func TestOrderHandler_CreateOrder_ServiceError(t *testing.T) {
	// Arrange
	mockService := &MockOrderService{}
	handler := &OrderHandler{service: mockService}

	app := fiber.New()
	app.Post("/orders", handler.CreateOrder)

	orderInput := models.CreateOrderInput{
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

	requestBody, _ := json.Marshal(orderInput)
	mockService.On("CreateOrder", mock.Anything, orderInput).Return(errors.New("service error"))

	// Act
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestOrderHandler_GetOrder_Success(t *testing.T) {
	// Arrange
	mockService := &MockOrderService{}
	handler := &OrderHandler{service: mockService}

	app := fiber.New()
	app.Get("/orders/:id", handler.GetOrder)

	expectedOrder := models.OrderWithItems{
		Order: models.Order{
			ID:           1,
			CustomerName: "John Doe",
			TotalAmount:  100.50,
			Status:       models.StatusPending,
		},
		Items: []models.OrderItem{
			{
				ID:          1,
				OrderID:     1,
				ProductName: "Product 1",
				Quantity:    2,
				Price:       50.25,
			},
		},
	}

	mockService.On("GetOrderById", mock.Anything, 1).Return(expectedOrder, nil)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestOrderHandler_GetOrder_InvalidID(t *testing.T) {
	// Arrange
	mockService := &MockOrderService{}
	handler := &OrderHandler{service: mockService}

	app := fiber.New()
	app.Get("/orders/:id", handler.GetOrder)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/orders/invalid", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockService.AssertNotCalled(t, "GetOrderById")
}

// Benchmark tests for HTTP handlers
func BenchmarkOrderHandler_CreateOrder(b *testing.B) {
	mockService := &MockOrderService{}
	handler := &OrderHandler{service: mockService}

	app := fiber.New()
	app.Post("/orders", handler.CreateOrder)

	orderInput := models.CreateOrderInput{
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

	requestBody, _ := json.Marshal(orderInput)
	mockService.On("CreateOrder", mock.Anything, orderInput).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		_, _ = app.Test(req)
	}
}

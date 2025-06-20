package v1

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/application/domain"
	"github.com/Testzyler/order-management-go/application/models"
	"github.com/Testzyler/order-management-go/application/repositories"
	"github.com/Testzyler/order-management-go/application/services"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

type OrderHandler struct {
	service domain.OrderService
}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

// Initialize implements HandlerInitializer interface
func (h *OrderHandler) Initialize() {
	repo := repositories.NewOrderRepository(route.GetDatabasePool())
	service := services.NewOrderService(repo)
	h.service = service
}

// GetRouteDefinition implements HandlerInitializer interface
func (h *OrderHandler) GetRouteDefinition() route.RouteDefinition {
	return route.RouteDefinition{
		Routes: route.Routes{
			route.Route{
				Name:        "CreateOrder",
				Path:        "/",
				Method:      constants.METHOD_POST,
				HandlerFunc: h.CreateOrder,
			},
			route.Route{
				Name:        "GetOrder",
				Path:        "/:id",
				Method:      constants.METHOD_GET,
				HandlerFunc: h.GetOrder,
			},
			route.Route{
				Name:        "UpdateOrder",
				Path:        "/:id/status",
				Method:      constants.METHOD_PUT,
				HandlerFunc: h.UpdateOrder,
			},
			route.Route{
				Name:        "DeleteOrder",
				Path:        "/:id",
				Method:      constants.METHOD_DELETE,
				HandlerFunc: h.DeleteOrder,
			},
			route.Route{
				Name:        "ListOrders",
				Path:        "/",
				Method:      constants.METHOD_GET,
				HandlerFunc: h.ListOrders,
			},
		},
		Prefix: "orders",
	}
}

// Auto-register the handler
func init() {
	route.RegisterHandler(NewOrderHandler())
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context())
	var input models.CreateOrderInput
	ctx := c.Context()

	if err := c.BodyParser(&input); err != nil {
		requestLogger.WithError(err).Error("Failed to parse request body")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	start := time.Now()
	err := h.service.CreateOrder(ctx, input)
	duration := time.Since(start)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			requestLogger.Warn("Request cancelled by client", "duration_ms", duration.Milliseconds())
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message": "Request was cancelled by client",
			})
		}
		if errors.Is(err, context.DeadlineExceeded) {
			requestLogger.Warn("Request timed out", "duration_ms", duration.Milliseconds())
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message": "Request timed out",
			})
		}

		requestLogger.WithError(err).Error("Failed to create order", "duration_ms", duration.Milliseconds())
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	requestLogger.Info("Order created successfully", "duration_ms", duration.Milliseconds())
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Order created successfully",
	})
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context())
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		requestLogger.Error("Order ID is required")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		requestLogger.WithError(err).Error("Invalid Order ID format", "id", id)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID format",
		})
	}

	start := time.Now()
	order, err := h.service.GetOrderById(ctx, idInt)
	duration := time.Since(start)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			requestLogger.Warn("Request cancelled by client", "order_id", idInt, "duration_ms", duration.Milliseconds())
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message": "Request was cancelled by client",
			})
		}
		if errors.Is(err, context.DeadlineExceeded) {
			requestLogger.Warn("Request timed out", "order_id", idInt, "duration_ms", duration.Milliseconds())
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message": "Request timed out",
			})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			requestLogger.Warn("Order not found", "order_id", idInt)
			return c.Status(fiber.ErrNotFound.Code).JSON(fiber.Map{
				"message": "Order not found",
			})
		}
		requestLogger.WithError(err).Error("Failed to get order", "order_id", idInt, "duration_ms", duration.Milliseconds())
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": order,
	})
}

func (h *OrderHandler) UpdateOrder(c *fiber.Ctx) error {
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context())
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		requestLogger.Error("Order ID is required for update")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	var input models.UpdateOrderInput
	if err := c.BodyParser(&input); err != nil {
		requestLogger.WithError(err).Error("Failed to parse update request body")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		requestLogger.WithError(err).Error("Invalid Order ID format", "id", id)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}

	input.ID = idInt
	err = h.service.UpdateOrder(ctx, input)
	if err != nil {
		requestLogger.WithError(err).Error("Failed to update order", "order_id", idInt)
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	requestLogger.Info("Order updated successfully", "order_id", idInt, "status", input.Status)
	return c.JSON(fiber.Map{
		"message": "Order updated successfully",
	})
}

func (h *OrderHandler) DeleteOrder(c *fiber.Ctx) error {
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context())
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		requestLogger.Error("Order ID is required for deletion")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		requestLogger.WithError(err).Error("Invalid Order ID format", "id", id)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}

	err = h.service.DeleteOrder(ctx, idInt)
	if err != nil {
		requestLogger.WithError(err).Error("Failed to delete order", "order_id", idInt)
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	requestLogger.Info("Order deleted successfully", "order_id", idInt)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "Order deleted successfully",
	})
}

func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context())
	ctx := c.Context()
	page := c.Query("page", "1")
	size := c.Query("size", "10")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		requestLogger.WithError(err).Error("Invalid page parameter", "page", page)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid page number",
		})
	}
	sizeInt, err := strconv.Atoi(size)
	if err != nil || sizeInt < 1 {
		requestLogger.WithError(err).Error("Invalid size parameter", "size", size)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid size number",
		})
	}

	orders, err := h.service.ListOrders(ctx, models.ListInput{
		Page: pageInt,
		Size: sizeInt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			requestLogger.Warn("No orders found", "page", pageInt, "size", sizeInt)
			return c.Status(fiber.ErrNotFound.Code).JSON(fiber.Map{
				"message": "Order not found",
			})
		}

		requestLogger.WithError(err).Error("Failed to list orders", "page", pageInt, "size", sizeInt)
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.JSON(orders)
}

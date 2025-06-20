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
	// Get logger with request ID from context and enhanced metadata
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")
	requestLogger.Info("Handler: CreateOrder started",
		"endpoint", "POST /orders",
		"content_type", c.Get("Content-Type"),
	)

	var input models.CreateOrderInput
	// Use the context that already has request ID
	ctx := c.Context()

	// Check if request is already cancelled
	select {
	case <-ctx.Done():
		requestLogger.Warn("Handler: Request cancelled before processing", "error", ctx.Err())
		if errors.Is(ctx.Err(), context.Canceled) {
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message":    "Request was cancelled by client",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message":    "Request timed out",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"message":    "Request cancelled",
			"request_id": logger.RequestIDFromContext(ctx),
		})
	default:
		// Continue processing
	}

	// Log body parsing attempt
	requestLogger.Debug("Handler: Parsing request body")
	if err := c.BodyParser(&input); err != nil {
		requestLogger.WithError(err).Error("Handler: Failed to parse request body",
			"body_size", len(c.Body()),
			"content_type", c.Get("Content-Type"),
		)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message":    err.Error(),
			"request_id": logger.RequestIDFromContext(ctx),
		})
	}

	requestLogger.WithFields(map[string]interface{}{
		"customer_name": input.CustomerName,
		"item_count":    len(input.Items),
		"status":        input.Status,
	}).Info("Handler: Successfully parsed order input")

	// Log service call start
	requestLogger.Debug("Handler: Calling service layer to create order")
	start := time.Now()

	err := h.service.CreateOrder(ctx, input)

	duration := time.Since(start)
	logger.LogServiceCall(requestLogger, "OrderService", "CreateOrder", duration, logger.RequestIDFromContext(ctx))

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			requestLogger.Warn("Handler: Service call cancelled by client",
				"service_duration_ms", duration.Milliseconds(),
				"error", err,
			)
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message":    "Request was cancelled by client",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}
		if errors.Is(err, context.DeadlineExceeded) {
			requestLogger.Warn("Handler: Service call timed out",
				"service_duration_ms", duration.Milliseconds(),
				"error", err,
			)
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message":    "Request timed out",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}

		requestLogger.WithError(err).Error("Handler: Service layer failed to create order",
			"service_duration_ms", duration.Milliseconds(),
		)
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message":    err.Error(),
			"request_id": logger.RequestIDFromContext(ctx),
		})
	}

	requestLogger.Info("Handler: Order created successfully",
		"service_duration_ms", duration.Milliseconds(),
	)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Order created successfully",
		"request_id": logger.RequestIDFromContext(ctx),
	})
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")

	ctx := c.Context()
	id := c.Params("id")

	requestLogger.Info("Handler: GetOrder started",
		"endpoint", "GET /orders/:id",
		"order_id", id,
	)

	// Check if request is already cancelled
	select {
	case <-ctx.Done():
		requestLogger.Warn("Handler: Request cancelled before processing", "order_id", id, "error", ctx.Err())
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"message":    "Request cancelled",
			"request_id": logger.RequestIDFromContext(ctx),
		})
	default:
		// Continue processing
	}

	if id == "" {
		requestLogger.Error("Handler: Order ID is required but not provided")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message":    "Order ID is required",
			"request_id": logger.RequestIDFromContext(ctx),
		})
	}

	requestLogger.Debug("Handler: Validating order ID format", "provided_id", id)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		requestLogger.WithError(err).Error("Handler: Invalid Order ID format",
			"provided_id", id,
			"expected_format", "integer",
		)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message":    "Invalid Order ID format",
			"request_id": logger.RequestIDFromContext(ctx),
		})
	}

	// Check for cancellation before service call
	select {
	case <-ctx.Done():
		requestLogger.Warn("Handler: Request cancelled before service call", "order_id", idInt, "error", ctx.Err())
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"message":    "Request cancelled during processing",
			"request_id": logger.RequestIDFromContext(ctx),
		})
	default:
		// Continue processing
	}

	requestLogger.Debug("Handler: Calling service layer to get order", "order_id", idInt)
	start := time.Now()

	order, err := h.service.GetOrderById(ctx, idInt)

	duration := time.Since(start)
	logger.LogServiceCall(requestLogger, "OrderService", "GetByID", duration, logger.RequestIDFromContext(ctx))

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			requestLogger.Warn("Handler: Service call cancelled by client",
				"order_id", idInt,
				"service_duration_ms", duration.Milliseconds(),
				"error", err,
			)
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message":    "Request was cancelled by client",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}
		if errors.Is(err, context.DeadlineExceeded) {
			requestLogger.Warn("Handler: Service call timed out",
				"order_id", idInt,
				"service_duration_ms", duration.Milliseconds(),
				"error", err,
			)
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"message":    "Request timed out",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			requestLogger.WithError(err).Warn("Handler: Order not found",
				"order_id", idInt,
				"service_duration_ms", duration.Milliseconds(),
			)
			return c.Status(fiber.ErrNotFound.Code).JSON(fiber.Map{
				"message":    "Order not found",
				"request_id": logger.RequestIDFromContext(ctx),
			})
		}
		requestLogger.WithError(err).Error("Handler: Service layer failed to get order",
			"order_id", idInt,
			"service_duration_ms", duration.Milliseconds(),
		)
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message":    err.Error(),
			"request_id": logger.RequestIDFromContext(ctx),
		})
	}

	requestLogger.Info("Handler: Order retrieved successfully",
		"order_id", idInt,
		"customer_name", order.CustomerName,
		"status", order.Status,
		"service_duration_ms", duration.Milliseconds(),
	)

	return c.JSON(fiber.Map{
		"data":       order,
		"request_id": logger.RequestIDFromContext(ctx),
	})
}

func (h *OrderHandler) UpdateOrder(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")

	ctx := c.Context()
	id := c.Params("id")

	requestLogger.Debug("Updating order", "order_id", id)

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
		requestLogger.WithError(err).Error("Invalid Order ID format for update", "provided_id", id)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}

	input.ID = idInt
	requestLogger.WithFields(map[string]interface{}{
		"order_id":   idInt,
		"new_status": input.Status,
	}).Debug("Parsed update input")

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
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")

	ctx := c.Context()
	id := c.Params("id")

	requestLogger.Debug("Deleting order", "order_id", id)

	if id == "" {
		requestLogger.Error("Order ID is required for deletion")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		requestLogger.WithError(err).Error("Invalid Order ID format for deletion", "provided_id", id)
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
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")

	ctx := c.Context()
	page := c.Query("page", "1")
	size := c.Query("size", "10")

	requestLogger.Debug("Listing orders", "page", page, "size", size)

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

	requestLogger.Info("Orders listed successfully", "page", pageInt, "size", sizeInt, "count", len(orders.Data))
	return c.JSON(orders)
}

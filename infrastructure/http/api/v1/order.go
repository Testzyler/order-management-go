package v1

import (
	"errors"
	"strconv"

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
	// Get logger with request ID from context (automatically includes request_id)
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")
	requestLogger.Info("Creating new order")

	var input models.CreateOrderInput
	// Use the context that already has request ID
	ctx := c.Context()

	if err := c.BodyParser(&input); err != nil {
		requestLogger.WithError(err).Error("Failed to parse request body")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	requestLogger.WithFields(map[string]interface{}{
		"customer_name": input.CustomerName,
		"item_count":    len(input.Items),
		"status":        input.Status,
	}).Debug("Parsed order input")

	err := h.service.CreateOrder(ctx, input)
	if err != nil {
		requestLogger.WithError(err).Error("Failed to create order")
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	requestLogger.Info("Order created successfully")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Order created successfully",
	})
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	// Get logger with request ID from context
	requestLogger := logger.LoggerWithRequestIDFromContext(c.Context()).WithComponent("order-handler")

	ctx := c.Context()
	id := c.Params("id")

	requestLogger.Debug("Getting order by ID", "order_id", id)

	if id == "" {
		requestLogger.Error("Order ID is required")
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		requestLogger.WithError(err).Error("Invalid Order ID format", "provided_id", id)
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}

	order, err := h.service.GetOrderById(ctx, idInt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			requestLogger.Warn("Order not found", "order_id", idInt)
			return c.Status(fiber.ErrNotFound.Code).JSON(fiber.Map{
				"message": "Order not found",
			})
		}
		// Handle other errors
		requestLogger.WithError(err).Error("Failed to get order", "order_id", idInt)
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	requestLogger.Info("Order retrieved successfully", "order_id", idInt)
	return c.JSON(order)
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

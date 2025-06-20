package v1

import (
	"context"
	"errors"
	"strconv"

	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/application/domain"
	"github.com/Testzyler/order-management-go/application/models"
	"github.com/Testzyler/order-management-go/application/repositories"
	"github.com/Testzyler/order-management-go/application/services"
	"github.com/Testzyler/order-management-go/infrastructure/http/api/route"
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
	var input models.CreateOrderInput
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	err := h.service.CreateOrder(ctx, input)
	if err != nil {
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Order created successfully",
	})
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}

	order, err := h.service.GetOrderById(ctx, idInt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.ErrNotFound.Code).JSON(fiber.Map{
				"message": "Order not found",
			})
		}
		// Handle other errors
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.JSON(order)
}

func (h *OrderHandler) UpdateOrder(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	var input models.UpdateOrderInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}
	input.ID = idInt
	err = h.service.UpdateOrder(ctx, input)
	if err != nil {
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Order updated successfully",
	})
}

func (h *OrderHandler) DeleteOrder(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Order ID is required",
		})
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid Order ID",
		})
	}
	err = h.service.DeleteOrder(ctx, idInt)
	if err != nil {
		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "Order deleted successfully",
	})
}

func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()
	page := c.Query("page", "1")
	size := c.Query("size", "10")
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "Invalid page number",
		})
	}
	sizeInt, err := strconv.Atoi(size)
	if err != nil || sizeInt < 1 {
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
			return c.Status(fiber.ErrNotFound.Code).JSON(fiber.Map{
				"message": "Order not found",
			})
		}

		return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.JSON(orders)
}

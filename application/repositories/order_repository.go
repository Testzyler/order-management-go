package repositories

import (
	"context"
	"fmt"

	"github.com/Testzyler/order-management-go/application/models"
	"github.com/Testzyler/order-management-go/infrastructure/utils/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *orderRepository {
	return &orderRepository{
		db: db,
	}
}

func (r *orderRepository) ListOrders(ctx context.Context, input models.ListInput) (*models.ListPaginatedOrders, error) {
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx)

	if input.Page < 1 {
		input.Page = 1
	}
	if input.Size < 1 {
		input.Size = 10
	}
	offset := (input.Page - 1) * input.Size

	queryOrders := `
		SELECT COUNT(*) OVER() AS total_count, id, customer_name, total_amount, status, created_at, updated_at 
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, queryOrders, input.Size, offset)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to query orders")
		return nil, err
	}
	defer rows.Close()

	var (
		orderIDs []int
		total    int
		orderMap = make(map[int]*models.OrderWithItems)
	)

	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&total, &order.ID, &order.CustomerName, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
			repoLogger.WithError(err).Error("Failed to scan order row")
			return nil, err
		}
		orderIDs = append(orderIDs, order.ID)
		orderWithItems := &models.OrderWithItems{Order: order}
		orderMap[order.ID] = orderWithItems
	}

	if len(orderIDs) == 0 {
		return &models.ListPaginatedOrders{
			Data:       []models.OrderWithItems{},
			Total:      0,
			Page:       input.Page,
			Size:       input.Size,
			TotalPages: 0,
		}, nil
	}

	// Get items for all orders in the page
	queryItems := `SELECT id, order_id, product_name, quantity, price, created_at, updated_at
		FROM order_items
		WHERE order_id = ANY($1)`

	itemRows, err := r.db.Query(ctx, queryItems, orderIDs)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to query order items")
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item models.OrderItem
		if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductName, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt); err != nil {
			repoLogger.WithError(err).Error("Failed to scan order item")
			return nil, err
		}
		if orderMap[item.OrderID] != nil {
			orderMap[item.OrderID].Items = append(orderMap[item.OrderID].Items, item)
		}
	}

	// Combine into list
	var orderWithItems []models.OrderWithItems
	for _, oid := range orderIDs {
		orderWithItems = append(orderWithItems, *orderMap[oid])
	}

	totalPages := (total + input.Size - 1) / input.Size
	if err := itemRows.Err(); err != nil {
		repoLogger.WithError(err).Error("Error scanning order items")
		return nil, fmt.Errorf("error scanning order items: %w", err)
	}

	return &models.ListPaginatedOrders{
		Data:       orderWithItems,
		Total:      total,
		Page:       input.Page,
		Size:       input.Size,
		TotalPages: totalPages,
	}, nil
}

func (r *orderRepository) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx)
	var result models.OrderWithItems
	var order models.Order
	query := `
		SELECT id, customer_name, total_amount, status, created_at, updated_at 
		FROM orders 
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.CustomerName,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		repoLogger.WithError(err).Error("Failed to query order", "order_id", id)
		return models.OrderWithItems{}, err
	}

	// Fetch order items
	itemQuery := `SELECT id, order_id, product_name, quantity, price, created_at, updated_at
		FROM order_items
		WHERE order_id = $1`

	itemRows, err := r.db.Query(ctx, itemQuery, id)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to fetch order items", "order_id", id)
		return models.OrderWithItems{}, fmt.Errorf("failed to fetch order items: %w", err)
	}
	defer itemRows.Close()

	var items []models.OrderItem
	for itemRows.Next() {
		var item models.OrderItem
		if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductName, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt); err != nil {
			repoLogger.WithError(err).Error("Failed to scan order item", "order_id", id)
			return models.OrderWithItems{}, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	result.Order = order
	result.Items = items

	return result, nil
}

func (r *orderRepository) CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) (err error) {
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx)
	tx, err := r.db.Begin(ctx)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to begin transaction")
		err = errors.Wrap(err, "failed to begin transaction")
		return err
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				repoLogger.WithError(rollbackErr).Error("Failed to rollback transaction")
			}
		}
	}()

	// Insert order
	insertOrderQuery := "INSERT INTO orders (customer_name, total_amount, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id"

	var insertedOrderID int
	err = tx.QueryRow(ctx, insertOrderQuery, order.CustomerName, order.TotalAmount, order.Status, order.CreatedAt, order.UpdatedAt).Scan(&insertedOrderID)

	if err != nil {
		repoLogger.WithError(err).Error("Failed to insert order", "customer", order.CustomerName)
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert order items
	if len(items) > 0 {
		insertItemsQuery := "INSERT INTO order_items (order_id, product_name, quantity, price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"

		for i, item := range items {
			_, err = tx.Exec(ctx, insertItemsQuery, insertedOrderID, item.ProductName, item.Quantity, item.Price, item.CreatedAt, item.UpdatedAt)
			if err != nil {
				repoLogger.WithError(err).Error("Failed to insert order item", "order_id", insertedOrderID, "product", item.ProductName, "index", i)
				return fmt.Errorf("failed to insert order item: %w", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		repoLogger.WithError(err).Error("Failed to commit transaction", "order_id", insertedOrderID)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, order models.Order) (err error) {
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to begin transaction", "order_id", order.ID)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				repoLogger.WithError(rollbackErr).Error("Failed to rollback transaction", "order_id", order.ID)
			}
		}
	}()

	query := "UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3"
	result, err := tx.Exec(ctx, query, order.Status, order.UpdatedAt, order.ID)

	if err != nil {
		repoLogger.WithError(err).Error("Failed to update order", "order_id", order.ID)
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		repoLogger.Warn("Order not found", "order_id", order.ID)
		return fmt.Errorf("order with ID %d not found", order.ID)
	}

	if err = tx.Commit(ctx); err != nil {
		repoLogger.WithError(err).Error("Failed to commit transaction", "order_id", order.ID)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, id int) (err error) {
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to begin transaction", "order_id", id)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				repoLogger.WithError(rollbackErr).Error("Failed to rollback transaction", "order_id", id)
			}
		}
	}()

	// Delete order items first
	deleteItemsQuery := "DELETE FROM order_items WHERE order_id = $1"
	_, err = tx.Exec(ctx, deleteItemsQuery, id)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to delete order items", "order_id", id)
		return fmt.Errorf("failed to delete order items: %w", err)
	}

	// Delete the order
	deleteOrderQuery := "DELETE FROM orders WHERE id = $1"
	orderResult, err := tx.Exec(ctx, deleteOrderQuery, id)
	if err != nil {
		repoLogger.WithError(err).Error("Failed to delete order", "order_id", id)
		return fmt.Errorf("failed to delete order: %w", err)
	}

	orderRowsAffected := orderResult.RowsAffected()
	if orderRowsAffected == 0 {
		repoLogger.Warn("Order not found", "order_id", id)
		return fmt.Errorf("order with ID %d not found", id)
	}

	if err = tx.Commit(ctx); err != nil {
		repoLogger.WithError(err).Error("Failed to commit transaction", "order_id", id)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

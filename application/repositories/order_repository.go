package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Testzyler/order-management-go/application/models"
	"github.com/Testzyler/order-management-go/infrastructure/logger"
	"github.com/jackc/pgx/v5/pgxpool"
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
	// Use logger with request ID from context
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-repository")
	repoLogger.Debug("Fetching order from database")

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
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item models.OrderItem
		if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductName, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt); err != nil {
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
	repoLogger.Debug("Orders fetched successfully", "total", total, "page", input.Page, "size", input.Size, "total_pages", totalPages)
	if err := itemRows.Err(); err != nil {
		repoLogger.WithError(err).Error("Error scanning order items")
		return nil, fmt.Errorf("error scanning order items: %w", err)
	}
	repoLogger.Info("Order items fetched successfully", "item_count", len(orderWithItems))
	return &models.ListPaginatedOrders{
		Data:       orderWithItems,
		Total:      total,
		Page:       input.Page,
		Size:       input.Size,
		TotalPages: totalPages,
	}, nil
}

func (r *orderRepository) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	// Use logger with request ID from context
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-repository")
	repoLogger.Debug("Fetching order from database", "order_id", id)

	var result models.OrderWithItems
	var order models.Order
	query := `
		SELECT id, customer_name, total_amount, status, created_at, updated_at 
		FROM orders 
		WHERE id = $1`

	startTime := time.Now()
	// Context cancellation is automatically handled by pgx
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.CustomerName,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	queryDuration := time.Since(startTime)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Order query cancelled by client",
				"order_id", id,
				"query_duration_ms", queryDuration.Milliseconds(),
			)
			return models.OrderWithItems{}, err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Order query timed out",
				"order_id", id,
				"query_duration_ms", queryDuration.Milliseconds(),
			)
			return models.OrderWithItems{}, err
		}
		if err == sql.ErrNoRows {
			repoLogger.Warn("Order not found in database", "order_id", id, "query_duration_ms", queryDuration.Milliseconds())
			return models.OrderWithItems{}, err
		}
		repoLogger.WithError(err).Error("Failed to query order", "order_id", id, "query_duration_ms", queryDuration.Milliseconds())
		return models.OrderWithItems{}, err
	}

	repoLogger.Debug("Order fetched, now fetching items", "order_id", id, "query_duration_ms", queryDuration.Milliseconds())

	// Fetch order items - context cancellation handled automatically by pgx
	itemQuery := `SELECT id, order_id, product_name, quantity, price, created_at, updated_at
		FROM order_items
		WHERE order_id = $1`

	itemsStartTime := time.Now()
	itemRows, err := r.db.Query(ctx, itemQuery, id)
	if err != nil {
		itemsQueryDuration := time.Since(itemsStartTime)
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Order items query cancelled by client",
				"order_id", id,
				"items_query_duration_ms", itemsQueryDuration.Milliseconds(),
			)
			return models.OrderWithItems{}, err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Order items query timed out",
				"order_id", id,
				"items_query_duration_ms", itemsQueryDuration.Milliseconds(),
			)
			return models.OrderWithItems{}, err
		}
		repoLogger.WithError(err).Error("Failed to fetch order items", "order_id", id, "items_query_duration_ms", itemsQueryDuration.Milliseconds())
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

	itemsQueryDuration := time.Since(itemsStartTime)
	totalDuration := time.Since(startTime)

	result.Order = order
	result.Items = items

	repoLogger.Info("Order with items fetched successfully",
		"order_id", id,
		"item_count", len(items),
		"order_query_duration_ms", queryDuration.Milliseconds(),
		"items_query_duration_ms", itemsQueryDuration.Milliseconds(),
		"total_duration_ms", totalDuration.Milliseconds(),
	)
	return result, nil
}

func (r *orderRepository) CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) (err error) {
	// Use logger with request ID from context
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-repository")
	startTime := time.Now()

	repoLogger.Debug("Starting order creation transaction",
		"customer_name", order.CustomerName,
		"total_amount", order.TotalAmount,
		"item_count", len(items),
	)

	// Begin transaction - context cancellation handled automatically by pgx
	tx, err := r.db.Begin(ctx)
	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Transaction begin cancelled by client")
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Transaction begin timed out")
			return err
		}
		repoLogger.WithError(err).Error("Failed to begin transaction for order creation")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				repoLogger.WithError(rollbackErr).Error("Failed to rollback transaction")
			}
		}
	}()

	// Insert order - context cancellation handled automatically by pgx
	insertOrderQuery := "INSERT INTO orders (customer_name, total_amount, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id"

	repoLogger.Debug("Executing order insertion query")
	insertStart := time.Now()
	var insertedOrderID int
	err = tx.QueryRow(ctx, insertOrderQuery, order.CustomerName, order.TotalAmount, order.Status, order.CreatedAt, order.UpdatedAt).Scan(&insertedOrderID)
	insertDuration := time.Since(insertStart)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Order insertion cancelled by client",
				"query_duration_ms", insertDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Order insertion timed out",
				"query_duration_ms", insertDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to insert order",
			"customer_name", order.CustomerName,
			"query_duration_ms", insertDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to insert order: %w", err)
	}

	repoLogger.Debug("Order inserted successfully",
		"order_id", insertedOrderID,
		"customer_name", order.CustomerName,
		"insert_duration_ms", insertDuration.Milliseconds(),
	)

	// Batch insert order items - context cancellation handled by pgx
	if len(items) > 0 {
		insertItemsQuery := "INSERT INTO order_items (order_id, product_name, quantity, price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"

		repoLogger.Debug("Executing batch order items insertion", "item_count", len(items))
		batchStart := time.Now()

		for i, item := range items {
			_, err = tx.Exec(ctx, insertItemsQuery, insertedOrderID, item.ProductName, item.Quantity, item.Price, item.CreatedAt, item.UpdatedAt)
			if err != nil {
				// Check if error is due to context cancellation
				if errors.Is(err, context.Canceled) {
					repoLogger.Warn("Order items insertion cancelled by client",
						"order_id", insertedOrderID,
						"item_index", i,
					)
					return err
				}
				if errors.Is(err, context.DeadlineExceeded) {
					repoLogger.Warn("Order items insertion timed out",
						"order_id", insertedOrderID,
						"item_index", i,
					)
					return err
				}
				repoLogger.WithError(err).Error("Failed to insert order item",
					"order_id", insertedOrderID,
					"item_index", i,
					"product_name", item.ProductName,
				)
				return fmt.Errorf("failed to insert order item: %w", err)
			}
		}

		batchDuration := time.Since(batchStart)
		repoLogger.Debug("All order items inserted successfully",
			"order_id", insertedOrderID,
			"item_count", len(items),
			"batch_duration_ms", batchDuration.Milliseconds(),
		)
	}

	// Commit transaction - context cancellation handled by pgx
	commitStart := time.Now()
	if err = tx.Commit(ctx); err != nil {
		commitDuration := time.Since(commitStart)
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Transaction commit cancelled by client",
				"order_id", insertedOrderID,
				"commit_duration_ms", commitDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Transaction commit timed out",
				"order_id", insertedOrderID,
				"commit_duration_ms", commitDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to commit order creation transaction",
			"order_id", insertedOrderID,
			"commit_duration_ms", commitDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	commitDuration := time.Since(commitStart)
	totalDuration := time.Since(startTime)

	repoLogger.Info("Order and items created successfully",
		"order_id", insertedOrderID,
		"customer_name", order.CustomerName,
		"total_amount", order.TotalAmount,
		"item_count", len(items),
		"commit_duration_ms", commitDuration.Milliseconds(),
		"total_duration_ms", totalDuration.Milliseconds(),
	)
	return nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, order models.Order) (err error) {
	// Use logger with request ID from context
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-repository")
	startTime := time.Now()

	repoLogger.Debug("Starting order update transaction", "order_id", order.ID, "new_status", order.Status)

	// Begin transaction - context cancellation handled automatically by pgx
	tx, err := r.db.Begin(ctx)
	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Transaction begin cancelled by client", "order_id", order.ID)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Transaction begin timed out", "order_id", order.ID)
			return err
		}
		repoLogger.WithError(err).Error("Failed to begin transaction for order update", "order_id", order.ID)
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

	repoLogger.Debug("Executing order update query", "order_id", order.ID, "status", order.Status)
	updateStart := time.Now()
	// Context cancellation handled automatically by pgx
	result, err := tx.Exec(ctx, query, order.Status, order.UpdatedAt, order.ID)
	updateDuration := time.Since(updateStart)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Order update cancelled by client",
				"order_id", order.ID,
				"query_duration_ms", updateDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Order update timed out",
				"order_id", order.ID,
				"query_duration_ms", updateDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to execute order update query",
			"order_id", order.ID,
			"query_duration_ms", updateDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsAffected := result.RowsAffected()
	repoLogger.Debug("Order update query executed",
		"order_id", order.ID,
		"rows_affected", rowsAffected,
		"query_duration_ms", updateDuration.Milliseconds(),
	)

	if rowsAffected == 0 {
		repoLogger.Warn("No rows affected by order update", "order_id", order.ID)
		return fmt.Errorf("order with ID %d not found", order.ID)
	}

	// Commit transaction - context cancellation handled automatically by pgx
	commitStart := time.Now()
	if err = tx.Commit(ctx); err != nil {
		commitDuration := time.Since(commitStart)
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Transaction commit cancelled by client",
				"order_id", order.ID,
				"commit_duration_ms", commitDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Transaction commit timed out",
				"order_id", order.ID,
				"commit_duration_ms", commitDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to commit order update transaction",
			"order_id", order.ID,
			"commit_duration_ms", commitDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	commitDuration := time.Since(commitStart)
	totalDuration := time.Since(startTime)

	repoLogger.Info("Order updated successfully",
		"order_id", order.ID,
		"status", order.Status,
		"rows_affected", rowsAffected,
		"commit_duration_ms", commitDuration.Milliseconds(),
		"total_duration_ms", totalDuration.Milliseconds(),
	)
	return nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, id int) (err error) {
	// Use logger with request ID from context
	repoLogger := logger.LoggerWithRequestIDFromContext(ctx).WithComponent("order-repository")
	startTime := time.Now()

	repoLogger.Debug("Starting order deletion transaction", "order_id", id)

	// Begin transaction - context cancellation handled automatically by pgx
	tx, err := r.db.Begin(ctx)
	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Transaction begin cancelled by client", "order_id", id)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Transaction begin timed out", "order_id", id)
			return err
		}
		repoLogger.WithError(err).Error("Failed to begin transaction for order deletion", "order_id", id)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				repoLogger.WithError(rollbackErr).Error("Failed to rollback transaction", "order_id", id)
			}
		}
	}()

	// First, delete order items - context cancellation handled automatically by pgx
	deleteItemsQuery := "DELETE FROM order_items WHERE order_id = $1"

	repoLogger.Debug("Executing order items deletion query", "order_id", id)
	itemsDeleteStart := time.Now()
	itemsResult, err := tx.Exec(ctx, deleteItemsQuery, id)
	itemsDeleteDuration := time.Since(itemsDeleteStart)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Order items deletion cancelled by client",
				"order_id", id,
				"query_duration_ms", itemsDeleteDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Order items deletion timed out",
				"order_id", id,
				"query_duration_ms", itemsDeleteDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to delete order items",
			"order_id", id,
			"query_duration_ms", itemsDeleteDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to delete order items: %w", err)
	}

	itemsAffected := itemsResult.RowsAffected()
	repoLogger.Debug("Order items deleted",
		"order_id", id,
		"items_deleted", itemsAffected,
		"items_delete_duration_ms", itemsDeleteDuration.Milliseconds(),
	)

	// Then, delete the order - context cancellation handled automatically by pgx
	deleteOrderQuery := "DELETE FROM orders WHERE id = $1"

	repoLogger.Debug("Executing order deletion query", "order_id", id)
	orderDeleteStart := time.Now()
	orderResult, err := tx.Exec(ctx, deleteOrderQuery, id)
	orderDeleteDuration := time.Since(orderDeleteStart)

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Order deletion cancelled by client",
				"order_id", id,
				"query_duration_ms", orderDeleteDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Order deletion timed out",
				"order_id", id,
				"query_duration_ms", orderDeleteDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to delete order",
			"order_id", id,
			"query_duration_ms", orderDeleteDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to delete order: %w", err)
	}

	orderRowsAffected := orderResult.RowsAffected()
	repoLogger.Debug("Order deletion query executed",
		"order_id", id,
		"rows_affected", orderRowsAffected,
		"query_duration_ms", orderDeleteDuration.Milliseconds(),
	)

	if orderRowsAffected == 0 {
		repoLogger.Warn("No rows affected by order deletion", "order_id", id)
		return fmt.Errorf("order with ID %d not found", id)
	}

	// Commit transaction - context cancellation handled automatically by pgx
	commitStart := time.Now()
	if err = tx.Commit(ctx); err != nil {
		commitDuration := time.Since(commitStart)
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			repoLogger.Warn("Transaction commit cancelled by client",
				"order_id", id,
				"commit_duration_ms", commitDuration.Milliseconds(),
			)
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			repoLogger.Warn("Transaction commit timed out",
				"order_id", id,
				"commit_duration_ms", commitDuration.Milliseconds(),
			)
			return err
		}
		repoLogger.WithError(err).Error("Failed to commit order deletion transaction",
			"order_id", id,
			"commit_duration_ms", commitDuration.Milliseconds(),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	commitDuration := time.Since(commitStart)
	totalDuration := time.Since(startTime)

	repoLogger.Info("Order and items deleted successfully",
		"order_id", id,
		"items_deleted", itemsAffected,
		"order_rows_affected", orderRowsAffected,
		"commit_duration_ms", commitDuration.Milliseconds(),
		"total_duration_ms", totalDuration.Milliseconds(),
	)
	return nil
}

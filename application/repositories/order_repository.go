package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Testzyler/order-management-go/application/models"
	"github.com/jackc/pgx/v5"
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
	return &models.ListPaginatedOrders{
		Data:       orderWithItems,
		Total:      total,
		Page:       input.Page,
		Size:       input.Size,
		TotalPages: totalPages,
	}, nil
}

func (r *orderRepository) GetOrderById(ctx context.Context, id int) (models.OrderWithItems, error) {
	if err := ctx.Err(); err != nil {
		return models.OrderWithItems{}, err
	}
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
		if err == sql.ErrNoRows {
			return models.OrderWithItems{}, nil
		}
		return models.OrderWithItems{}, err
	}
	// Fetch order items
	itemQuery := `	SELECT id, order_id, product_name, quantity, price, created_at, updated_at
		FROM order_items
		WHERE order_id = $1`
	itemRows, err := r.db.Query(ctx, itemQuery, id)
	if err != nil {
		return models.OrderWithItems{}, fmt.Errorf("failed to fetch order items: %w", err)
	}
	defer itemRows.Close()
	var items []models.OrderItem
	for itemRows.Next() {
		var item models.OrderItem
		if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductName, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return models.OrderWithItems{}, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}
	result.Order = order
	result.Items = items
	return result, nil
}

func (r *orderRepository) CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) (err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	timeNow := time.Now()
	var insertedOrderID int

	// Insert the order and get its ID
	orderQuery := "INSERT INTO orders (customer_name, total_amount, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id"
	err = tx.QueryRow(ctx, orderQuery, order.CustomerName, order.TotalAmount, order.Status, timeNow, timeNow).Scan(&insertedOrderID)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Batch insert order items if any
	if len(items) > 0 {
		rows := make([][]interface{}, len(items))
		for i, item := range items {
			rows[i] = []interface{}{insertedOrderID, item.ProductName, item.Quantity, item.Price, timeNow, timeNow}
		}

		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"order_items"},
			[]string{"order_id", "product_name", "quantity", "price", "created_at", "updated_at"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("failed to batch insert order items: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, order models.Order) (err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := "UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3"
	_, err = tx.Exec(ctx, query, order.Status, order.UpdatedAt, order.ID)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, id int) (err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := "DELETE FROM orders WHERE id = $1"
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

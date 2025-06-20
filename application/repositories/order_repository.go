package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Testzyler/order-management-go/application/models"
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

func (r *orderRepository) GetOrderById(ctx context.Context, id int) (models.Order, error) {
	if err := ctx.Err(); err != nil {
		return models.Order{}, err
	}
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
			return models.Order{}, nil
		}
		return models.Order{}, err
	}
	return order, nil
}

func (r *orderRepository) CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	timeNow := time.Now()
	var insertedOrderID int

	query := "INSERT INTO orders (customer_name, total_amount, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id"
	err = tx.QueryRow(ctx, query, order.CustomerName, order.TotalAmount, order.Status, timeNow, timeNow).Scan(&insertedOrderID)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// create order items if any
	if len(items) > 0 {
		itemQuery := "INSERT INTO order_items (order_id, product_name, quantity, price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
		for _, item := range items {
			_, err = tx.Query(ctx, itemQuery, insertedOrderID, item.ProductName, item.Quantity, item.Price, timeNow, timeNow)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, order models.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	query := "UPDATE orders SET customer_name = $1, total_amount = $2, status = $3, updated_at = $4 WHERE id = $5"
	_, err = tx.Query(ctx, query, order.CustomerName, order.TotalAmount, order.Status, time.Now(), order.ID)
	if err != nil {
		return err
	}

	order.UpdatedAt = time.Now()
	return nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, id int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	query := "DELETE FROM orders WHERE id = ?"
	_, err = tx.Query(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

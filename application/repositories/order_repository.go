package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Testzyler/order-management-go/application/models"
)

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *orderRepository {
	return &orderRepository{
		db: db,
	}
}

func (r *orderRepository) ListOrders(ctx context.Context, input models.ListOrdersInput) (*models.ListPaginatedOrders, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Size < 1 {
		input.Size = 10
	}
	offset := (input.Page - 1) * input.Size

	// Query total count
	var total int
	query := `SELECT COUNT(*) OVER() AS total_count, id, customer_name, total_amount, status, created_at, updated_at 
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, input.Size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var order models.Order
		if err := rows.Scan(
			&total,
			&order.ID,
			&order.CustomerName,
			&order.TotalAmount,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := (total + input.Size - 1) / input.Size
	return &models.ListPaginatedOrders{
		Data:       orders,
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

	err := r.db.QueryRowContext(ctx, query, id).Scan(
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
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	timeNow := time.Now()
	var insertedOrderID int

	query := "INSERT INTO orders (customer_name, total_amount, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id"
	err = tx.QueryRowContext(ctx, query, order.CustomerName, order.TotalAmount, order.Status, timeNow, timeNow).Scan(&insertedOrderID)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// create order items if any
	if len(items) > 0 {
		itemQuery := "INSERT INTO order_items (order_id, product_name, quantity, price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
		for _, item := range items {
			_, err = tx.ExecContext(ctx, itemQuery, insertedOrderID, item.ProductName, item.Quantity, item.Price, timeNow, timeNow)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, order models.Order) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	query := "UPDATE orders SET customer_name = ?, total_amount = ?, status = ?, updated_at = ? WHERE id = ?"
	_, err = tx.ExecContext(ctx, query, order.CustomerName, order.TotalAmount, order.Status, time.Now(), order.ID)
	if err != nil {
		return err
	}
	order.UpdatedAt = time.Now()
	return nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	query := "DELETE FROM orders WHERE id = ?"
	_, err = tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

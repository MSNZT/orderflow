package orders

import (
	"context"
	"errors"
	"fmt"

	ordersapp "github.com/MSNZT/orderflow/internal/app/orders"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db postgres.DBTX
}

var _ ordersapp.Repository = (*Repository)(nil)

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID, offset int, limit int) ([]ordersapp.Order, error) {
	const op = "orders.repository.ListByUserID"

	query := `
		SELECT 
			id,
			user_id,
			status,
			total_price_cents,
			currency,
			expires_at,
			created_at,
			updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3;
	`

	orders := make([]ordersapp.Order, 0)

	db := postgres.ExecutorFromContext(ctx, r.db)

	rows, err := db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var o ordersapp.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Status, &o.TotalPriceCents, &o.Currency, &o.ExpiresAt,
			&o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return orders, nil

}

func (r *Repository) GetDetailsByIDAndUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (details *ordersapp.OrderDetails, err error) {
	const op = "orders.repository.GetDetailsByIDAndUserID"

	batch := &pgx.Batch{}

	batch.Queue(`
		SELECT 
			id,
			user_id,
			status,
			total_price_cents,
			currency,
			expires_at,
			created_at,
			updated_at
		FROM orders
		WHERE user_id = $1 AND id = $2
	`, userID, orderID)

	batch.Queue(`
		SELECT
			oi.id,
			oi.order_id,
			oi.product_id,
			oi.product_name,
			oi.unit_price_cents,
			oi.currency,
			oi.quantity,
			oi.line_total_price_cents,
			oi.created_at
		FROM order_items AS oi
		JOIN orders AS o ON o.id = oi.order_id
		WHERE oi.order_id = $2 AND o.user_id = $1 
		ORDER BY oi.created_at, oi.id;
	`, userID, orderID)

	db := postgres.ExecutorFromContext(ctx, r.db)
	br := db.SendBatch(ctx, batch)

	defer func() {
		if closeErr := br.Close(); closeErr != nil && err == nil {
			details = nil
			err = fmt.Errorf("%s: close batch: %w", op, closeErr)
		}
	}()

	var order ordersapp.Order
	err = br.QueryRow().Scan(&order.ID, &order.UserID, &order.Status, &order.TotalPriceCents, &order.Currency,
		&order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ordersapp.ErrOrderNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	orderItems := make([]ordersapp.OrderItem, 0)

	rows, err := br.Query()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var o ordersapp.OrderItem
		if err := rows.Scan(
			&o.ID, &o.OrderID, &o.ProductID, &o.ProductName, &o.UnitPriceCents,
			&o.Currency, &o.Quantity, &o.LineTotalPriceCents, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		orderItems = append(orderItems, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	details = &ordersapp.OrderDetails{
		Order: order,
		Items: orderItems,
	}

	return details, nil

}

func (r *Repository) CreateOrder(ctx context.Context, o *ordersapp.Order) error {
	const op = "orders.repository.CreateOrder"

	query := `
		INSERT INTO orders(
			id,
			user_id,
			status,
			total_price_cents,
			currency,
			expires_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	err := db.QueryRow(ctx, query, o.ID, o.UserID, o.Status, o.TotalPriceCents, o.Currency, o.ExpiresAt).Scan(
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) CreateOrderItems(ctx context.Context, orderItems []ordersapp.OrderItem) error {
	const op = "orders.repository.CreateOrderItems"

	if len(orderItems) == 0 {
		return nil
	}

	identifier := pgx.Identifier{"order_items"}
	columns := []string{"id", "order_id", "product_id", "product_name", "unit_price_cents",
		"currency", "quantity", "line_total_price_cents",
	}

	db := postgres.ExecutorFromContext(ctx, r.db)

	insertedRows, err := db.CopyFrom(ctx, identifier, columns, pgx.CopyFromSlice(len(orderItems), func(i int) ([]any, error) {
		item := orderItems[i]

		return []any{
			item.ID,
			item.OrderID,
			item.ProductID,
			item.ProductName,
			item.UnitPriceCents,
			item.Currency,
			item.Quantity,
			item.LineTotalPriceCents,
		}, nil
	}))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if insertedRows != int64(len(orderItems)) {
		return fmt.Errorf("%s: inserted %d order items, want %d", op, insertedRows, len(orderItems))
	}

	return nil
}

package orders

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) GetDetailsByID(ctx context.Context, orderID uuid.UUID) (details *ordersapp.OrderDetails, err error) {
	const op = "orders.repository.GetDetailsByID"

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
		WHERE id = $1
	`, orderID)

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
		WHERE oi.order_id = $1
		ORDER BY oi.created_at, oi.id;
	`, orderID)

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

func (r *Repository) MarkPaid(ctx context.Context, orderID uuid.UUID) error {
	const op = "orders.repository.MarkPaid"

	if err := r.markFromPending(ctx, orderID, ordersapp.StatusPaid, op); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) MarkCanceled(ctx context.Context, orderID uuid.UUID) error {
	const op = "orders.repository.MarkCanceled"

	if err := r.markFromPending(ctx, orderID, ordersapp.StatusCanceled, op); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) MarkExpired(ctx context.Context, orderID uuid.UUID) error {
	const op = "orders.repository.MarkExpired"

	if err := r.markFromPending(ctx, orderID, ordersapp.StatusExpired, op); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) FindExpiredPendingIDs(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	const op = "orders.repository.FindExpiredPending"

	query := `
		SELECT 
			id
		FROM orders
		WHERE status = 'pending' 
			AND expires_at < $1
		ORDER BY expires_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)
	overdueOrderIDs := make([]uuid.UUID, 0)

	rows, err := db.Query(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query overdue orders %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		var orderID uuid.UUID

		err := rows.Scan(&orderID)
		if err != nil {
			return nil, fmt.Errorf("%s: scan overdue order row: %w", op, err)
		}

		overdueOrderIDs = append(overdueOrderIDs, orderID)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: failed to iterate rows: %w", op, err)
	}

	return overdueOrderIDs, nil
}

func (r *Repository) markFromPending(ctx context.Context, orderID uuid.UUID, status ordersapp.Status, op string) error {

	if orderID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ordersapp.ErrOrderIDIsNil)
	}

	query := `
		WITH existing AS (
			SELECT id FROM orders WHERE id = $1
		),
		updated AS ( 
			UPDATE orders
			SET status = $2,
				updated_at = now()
			WHERE id = $1 AND status = 'pending'
			RETURNING id
		)
		SELECT
			EXISTS (SELECT 1 FROM existing) AS order_existing, 
			EXISTS (SELECT 1 FROM updated) AS order_updated
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var orderExisting, orderUpdated bool
	if err := db.QueryRow(ctx, query, orderID, status).Scan(&orderExisting, &orderUpdated); err != nil {
		return fmt.Errorf("%s: failed to query row order update: %w", op, err)
	}

	if !orderExisting {
		return fmt.Errorf("%s: %w", op, ordersapp.ErrOrderNotFound)
	}

	if !orderUpdated {
		return fmt.Errorf("%s: %w", op, ordersapp.ErrOrderStatusTransitionInvalid)
	}

	return nil
}

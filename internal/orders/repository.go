package orders

import (
	"context"
	"fmt"

	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db postgres.DBTX
}

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrder(ctx context.Context, o *Order) error {
	const op = "orders.repository.CreateOrder"

	query := `
		INSERT INTO orders(
			id,
			user_id,
			status,
			total_price_cents,
			currency
		) VALUES ($1, $2, $3, $4, $5)
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	_, err := db.Exec(ctx, query, o.ID, o.UserID, o.Status, o.TotalPriceCents, o.Currency)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) CreateOrderItems(ctx context.Context, orderItems []OrderItem) error {
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

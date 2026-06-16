package orders

import (
	"context"
	"fmt"

	"github.com/MSNZT/orderflow/internal/platform/postgres"
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
		INSERT orders(
			id,
			user_id,
			status,
			total_price_cents,
			currency
		) VALUES ($1, $2, $3, $4, $5)
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	res, err := db.Exec(ctx, query, &o.ID, &o.UserID, &o.Status, &o.TotalPriceCents, &o.Currency)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) CreateOrderItem(ctx context.Context, o *OrderItems) error {
	const op = "orders.repository.CreateOrderItem"

	query := `
		INSERT order_items(
			id,
			order_id,
			product_id,
			product_name,
			unit_price_cents,
			currency,
			quantity,
			total_price_cents
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	res, err := db.Exec(ctx, query,
		&o.ID, &o.OrderID, &o.ProductID, &o.ProductName, &o.UnitPriceCents,
		&o.Currency, &o.Quantity, &o.TotalPriceCents,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

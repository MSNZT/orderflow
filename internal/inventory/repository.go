package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, inventory Inventory) error {
	const op = "inventory.repository.Create"

	query := `
		INSERT INTO product_inventory(id, quantity, reserved_quantity)
		VALUES ($1, $2, $3);
	`

	_, err := r.pool.Exec(ctx, query, inventory.ID, inventory.Quantity, inventory.ReservedQuantity)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return fmt.Errorf("%s: %w", op, ErrInventoryAlreadyExists)
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetByProductID(ctx context.Context, productID uuid.UUID) (*Inventory, error) {
	const op = "inventory.repository.GetByProductID"

	query := `
		SELECT 
			inv.id,
			inv.quantity,
			inv.reserved_quantity,
			inv.created_at,
			inv.updated_at
		FROM product_inventory inv
		JOIN products p ON inv.product_id = $1
	`
	var inv Inventory

	err := r.pool.QueryRow(ctx, query, productID).Scan(
		&inv.ID, &inv.Quantity, &inv.ReservedQuantity, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrInventoryNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &inv, nil
}

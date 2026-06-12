package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db postgres.DBTX
}

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, productID uuid.UUID, quantity int32) error {
	const op = "inventory.repository.Create"

	query := `
		INSERT INTO product_inventory(product_id, quantity)
		VALUES ($1, $2, $55);
	`

	_, err := r.db.Exec(ctx, query, productID, quantity)
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
			product_id,
			quantity,
			reserved_quantity,
			created_at,
			updated_at
		FROM product_inventory
		WHERE product_id = $1;
	`
	var inv Inventory

	err := r.db.QueryRow(ctx, query, productID).Scan(
		&inv.ProductID, &inv.Quantity, &inv.ReservedQuantity, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrInventoryNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &inv, nil
}

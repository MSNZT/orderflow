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
		VALUES ($1, $2);
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	_, err := db.Exec(ctx, query, productID, quantity)
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

	db := postgres.ExecutorFromContext(ctx, r.db)

	err := db.QueryRow(ctx, query, productID).Scan(
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

func (r *Repository) GetByProductIDsForUpdate(ctx context.Context, productIDs []uuid.UUID) ([]Inventory, error) {
	const op = "inventory.repository.GetByProductIDsForUpdate"

	query := `
		SELECT 
			product_id, 
			quantity, 
			reserved_quantity, 
			created_at, 
			updated_at
		FROM product_inventory
		WHERE product_id = ANY($1)
		FOR UPDATE;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	rows, err := db.Query(ctx, query, productIDs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var inventories = make([]Inventory, 0)

	for rows.Next() {
		var inv Inventory

		if err := rows.Scan(
			&inv.ProductID, &inv.Quantity, &inv.ReservedQuantity, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		inventories = append(inventories, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return inventories, nil
}

func (r *Repository) DecreaseQuantity(ctx context.Context, productID uuid.UUID, requestedQuantity int) error {
	const op = "inventory.repository.DecreaseQuantity"

	query := `
		UPDATE product_inventory
		SET quantity = quantity - $2,
			updated_at = now()
		WHERE product_id = $1
			AND quantity - reserved_quantity >= $2;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	res, err := db.Exec(ctx, query, productID, requestedQuantity)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, ErrInsufficientStock)
	}

	return nil
}

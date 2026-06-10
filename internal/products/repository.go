package products

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListActive(ctx context.Context) ([]Product, error) {
	const op = "products.repository.ListActive"

	query := `
		SELECT 
			id,
			name,
			description,
			price_cents,
			currency,
			is_active,
			created_at,
			updated_at
		FROM products
		WHERE is_active = true
		LIMIT 20;
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var products = make([]Product, 0)

	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.Id, &p.Name, &p.Description, &p.PriceCents,
			&p.Currency, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return products, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	const op = "products.repository.GetByID"

	query := `
		SELECT 
			id,
			name,
			description,
			price_cents,
			currency,
			is_active,
			created_at,
			updated_at
		FROM products
		WHERE id = $1;
	`
	var p Product
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.Id, &p.Name, &p.Description, &p.PriceCents,
		&p.Currency, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

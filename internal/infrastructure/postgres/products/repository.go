package products

import (
	"context"
	"errors"
	"fmt"

	productsapp "github.com/MSNZT/orderflow/internal/app/products"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db postgres.DBTX
}

var _ productsapp.Repository = (*Repository)(nil)

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListActive(ctx context.Context) ([]productsapp.Product, error) {
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
		ORDER BY created_at DESC
		LIMIT 20;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var products = make([]productsapp.Product, 0)

	for rows.Next() {
		var p productsapp.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceCents,
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

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*productsapp.Product, error) {
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
		WHERE id = $1 AND is_active = true;
	`
	var p productsapp.Product

	db := postgres.ExecutorFromContext(ctx, r.db)

	if err := db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.PriceCents,
		&p.Currency, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, productsapp.ErrProductNotFound
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) Create(ctx context.Context, p *productsapp.Product) (*productsapp.Product, error) {
	const op = "products.repository.Create"

	query := `
		INSERT INTO products(id, name, description, price_cents, currency, is_active)
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, name, description, price_cents, currency, is_active, created_at, updated_at;
	`

	var product productsapp.Product
	db := postgres.ExecutorFromContext(ctx, r.db)

	err := db.QueryRow(ctx, query,
		p.ID, p.Name, p.Description, p.PriceCents, p.Currency, p.IsActive,
	).Scan(
		&product.ID, &product.Name, &product.Description, &product.PriceCents, &product.Currency,
		&product.IsActive, &product.CreatedAt, &product.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return nil, fmt.Errorf("%s: %w", op, productsapp.ErrProductAlreadyExists)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &product, nil
}

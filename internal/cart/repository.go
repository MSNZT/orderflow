package cart

import (
	"context"
	"fmt"

	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/google/uuid"
)

type Repository struct {
	db postgres.DBTX
}

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, userId uuid.UUID, limit int32, offset int32) ([]CartItem, error) {
	const op = "cart.repository.List"

	query := `
		SELECT 
			ci.product_id,
			p.name,
			ci.quantity,
			p.price_cents,
			ci.created_at,
			ci.updated_at
		FROM cart c
		JOIN cart_items ci ON c.id = ci.cart_id
		JOIN products p ON ci.product_id = p.id
		WHERE c.user_id = $1
		ORDER BY ci.created_at DESC
		LIMIT $2
		OFFSET $3;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, userId, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	var cartItems = make([]CartItem, 0, limit)

	for rows.Next() {
		var ci CartItem

		if err := rows.Scan(
			&ci.ProductID, &ci.Name, &ci.Quantity, &ci.PriceCents,
			&ci.CreatedAt, &ci.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		cartItems = append(cartItems, ci)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return cartItems, nil
}

func (r *Repository) CreateByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	const op = "cart.repository.CreateByUserID"

	query := `
		INSERT INTO cart (id, user_id)
		VALUES (gen_random_uuid(), $1)
		ON CONFLICT (user_id)
		DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var cartID uuid.UUID
	if err := db.QueryRow(ctx, query, userID).Scan(&cartID); err != nil {
		return cartID, fmt.Errorf("%s: %w", op, err)
	}

	return cartID, nil
}

func (r *Repository) AddItem(ctx context.Context, cartID uuid.UUID, productID uuid.UUID, quantity int32) error {
	const op = "cart.repository.AddItem"

	query := `
		INSERT INTO cart_items (cart_id, product_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (cart_id, product_id)
		DO UPDATE SET 
			quantity = cart_items.quantity + EXCLUDED.quantity,
			updated_at = now();
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	_, err := db.Exec(ctx, query, cartID, productID, quantity)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

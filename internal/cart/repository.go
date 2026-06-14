package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db postgres.DBTX
}

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetItems(ctx context.Context, userId uuid.UUID, limit int32, offset int32) ([]CartItem, error) {
	const op = "cart.repository.List"

	query := `
		SELECT 
			ci.product_id,
			p.name,
			ci.quantity,
			p.price_cents,
			ci.quantity * p.price_cents AS line_total_price_cents
		FROM carts c
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
			&ci.LineTotalPriceCents,
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

func (r *Repository) GetOrCreateByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	const op = "cart.repository.GetOrCreateByUserID"

	query := `
		INSERT INTO carts (id, user_id)
		VALUES (gen_random_uuid(), $1)
		ON CONFLICT (user_id)
		DO UPDATE SET updated_at = now()
		RETURNING id;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var cartID uuid.UUID
	if err := db.QueryRow(ctx, query, userID).Scan(&cartID); err != nil {
		return cartID, fmt.Errorf("%s: %w", op, err)
	}

	return cartID, nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	const op = "cart.repository.GetByUserID"

	query := `
		SELECT id FROM carts
		WHERE user_id = $1
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var cartID uuid.UUID
	if err := db.QueryRow(ctx, query, userID).Scan(&cartID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrCartNotFound)
		}
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
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

func (r *Repository) UpdateItemQuantity(
	ctx context.Context, cartID uuid.UUID, productID uuid.UUID, quantity int32) error {
	const op = "cart.repository.UpdateItemQuantity"

	query := `
		WITH updated_item AS (
			UPDATE cart_items
			SET quantity = $3,
				updated_at = now()
			WHERE cart_id = $1 
			  AND product_id = $2 
			RETURNING cart_id, product_id
		)
		UPDATE carts
		SET updated_at = now()
		WHERE id = (SELECT cart_id FROM updated_item)
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	res, err := db.Exec(ctx, query, cartID, productID, quantity)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, ErrCartItemNotFound)
	}

	return nil
}

func (r *Repository) DeleteItem(ctx context.Context, cartID uuid.UUID, productID uuid.UUID) error {
	const op = "cart.repository.DeleteItem"

	query := `
		WITH deleted AS (
			DELETE FROM cart_items
			WHERE cart_id = $1 AND product_id = $2
			RETURNING cart_id
		)
		UPDATE carts
		SET updated_at = now()
		WHERE id = (SELECT cart_id FROM deleted);
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	res, err := db.Exec(ctx, query, cartID, productID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		fmt.Println("remove=======")
		return fmt.Errorf("%s: %w", op, ErrCartItemNotFound)
	}

	return nil
}

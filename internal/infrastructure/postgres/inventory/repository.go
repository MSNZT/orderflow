package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	inventoryapp "github.com/MSNZT/orderflow/internal/app/inventory"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
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
				return fmt.Errorf("%s: %w", op, inventoryapp.ErrInventoryAlreadyExists)
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetByProductID(ctx context.Context, productID uuid.UUID) (*inventoryapp.Inventory, error) {
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
	var inv inventoryapp.Inventory

	db := postgres.ExecutorFromContext(ctx, r.db)

	err := db.QueryRow(ctx, query, productID).Scan(
		&inv.ProductID, &inv.Quantity, &inv.ReservedQuantity, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, inventoryapp.ErrInventoryNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &inv, nil
}

func (r *Repository) GetByProductIDsForUpdate(ctx context.Context, productIDs []uuid.UUID) ([]inventoryapp.Inventory, error) {
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
		ORDER BY product_id
		FOR UPDATE;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	rows, err := db.Query(ctx, query, productIDs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	var inventories = make([]inventoryapp.Inventory, 0, len(productIDs))

	for rows.Next() {
		var inv inventoryapp.Inventory

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

func (r *Repository) ReserveQuantity(ctx context.Context, productID uuid.UUID, quantity int) error {
	const op = "inventory.repository.ReserveQuantity"

	if quantity <= 0 {
		return fmt.Errorf("%s: %w", op, inventoryapp.ErrInventoryQuantityInvalid)
	}

	query := `
		UPDATE product_inventory
		SET reserved_quantity = reserved_quantity + $2,
			updated_at = now()
		WHERE product_id = $1 
		AND $2 > 0
		AND quantity - reserved_quantity >= $2
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	res, err := db.Exec(ctx, query, productID, quantity)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, inventoryapp.ErrInsufficientStock)
	}

	return nil
}

func (r *Repository) DecreaseQuantity(ctx context.Context, productID uuid.UUID, requestedQuantity int) error {
	const op = "inventory.repository.DecreaseQuantity"

	if requestedQuantity <= 0 {
		return fmt.Errorf("%s: %w", op, inventoryapp.ErrInventoryQuantityInvalid)
	}

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
		return fmt.Errorf("%s: %w", op, inventoryapp.ErrInsufficientStock)
	}

	return nil
}

func (r *Repository) CommitReservedQuantities(ctx context.Context, reservedItems []inventoryapp.ReservedItem) error {
	const op = "inventory.repository.CommitReservedQuantities"

	query := `
		WITH input AS (
			SELECT product_id, quantity
			FROM jsonb_to_recordset($1::jsonb) AS t(
				product_id UUID, quantity INTEGER
			)
		),
		existing AS (
			SELECT pi.product_id
			FROM product_inventory pi
			JOIN input AS i ON i.product_id = pi.product_id
		),
		updated AS (
			UPDATE product_inventory pi
			SET quantity = pi.quantity - input.quantity,
				reserved_quantity = pi.reserved_quantity - input.quantity,
				updated_at = now()
			FROM input
			WHERE pi.product_id = input.product_id 
				AND pi.reserved_quantity >= input.quantity
				AND pi.quantity >= input.quantity
			RETURNING pi.product_id
		)
		SELECT
			(SELECT COUNT(*) FROM input) AS input_count,
			(SELECT COUNT(*) FROM existing) AS existing_count,
			(SELECT COUNT(*) FROM updated) AS updated_count;
	`
	if err := r.applyReservedInventoryUpdate(ctx, reservedItems, query, op); err != nil {
		return err
	}

	return nil
}

func (r *Repository) ReleaseReservedQuantities(ctx context.Context, reservedItems []inventoryapp.ReservedItem) error {
	const op = "inventory.repository.ReleaseReservedQuantities"

	query := `
		WITH input AS (
			SELECT product_id, quantity
			FROM jsonb_to_recordset($1::jsonb) AS t(
				product_id UUID, quantity INTEGER
			)
		),
		existing AS (
			SELECT pi.product_id
			FROM product_inventory pi
			JOIN input AS i ON i.product_id = pi.product_id
		),
		updated AS (
			UPDATE product_inventory pi
			SET reserved_quantity = pi.reserved_quantity - input.quantity,
				updated_at = now()
			FROM input
			WHERE pi.product_id = input.product_id 
				AND pi.reserved_quantity >= input.quantity
			RETURNING pi.product_id
		)
		SELECT
			(SELECT COUNT(*) FROM input) AS input_count,
			(SELECT COUNT(*) FROM existing) AS existing_count,
			(SELECT COUNT(*) FROM updated) AS updated_count;
	`

	if err := r.applyReservedInventoryUpdate(ctx, reservedItems, query, op); err != nil {
		return err
	}

	return nil
}

func (r *Repository) applyReservedInventoryUpdate(ctx context.Context, reservedItems []inventoryapp.ReservedItem, query string, op string) error {
	if err := validateReservedItems(op, reservedItems); err != nil {
		return err
	}

	var inputCount, existingCount, updatedCount int64

	db := postgres.ExecutorFromContext(ctx, r.db)

	jsonBytes, err := json.Marshal(reservedItems)
	if err != nil {
		return fmt.Errorf("%s: marshal reserved items: %w", op, err)
	}

	if err := db.QueryRow(ctx, query, string(jsonBytes)).Scan(
		&inputCount, &existingCount, &updatedCount); err != nil {
		return fmt.Errorf("%s: scan reserved inventory update result: %w", op, err)
	}

	itemsCount := int64(len(reservedItems))

	if inputCount != itemsCount {
		return fmt.Errorf("%s: unexpected input count: got=%d want=%d", op, inputCount, itemsCount)
	}

	if existingCount != itemsCount {
		return fmt.Errorf(
			"%s: %w: input_count=%d existing_count=%d",
			op,
			inventoryapp.ErrInventoryNotFound,
			inputCount,
			existingCount,
		)
	}

	if updatedCount != itemsCount {
		return fmt.Errorf(
			"%s: %w: input_count=%d updated_count=%d",
			op,
			inventoryapp.ErrInsufficientStock,
			inputCount,
			updatedCount,
		)
	}
	return nil
}

func validateReservedItems(op string, reservedItems []inventoryapp.ReservedItem) error {
	if len(reservedItems) == 0 {
		return fmt.Errorf("%s: %w", op, inventoryapp.ErrReservedItemsEmpty)
	}

	var uniqueIDs = make(map[uuid.UUID]struct{}, len(reservedItems))
	for _, item := range reservedItems {
		if item.ProductID == uuid.Nil {
			return fmt.Errorf("%s: %w", op, inventoryapp.ErrProductIDIsNil)
		}

		if _, exists := uniqueIDs[item.ProductID]; exists {
			return fmt.Errorf("%s: %w", op, inventoryapp.ErrDuplicateProductID)
		}

		if item.Quantity <= 0 {
			return fmt.Errorf("%s: %w", op, inventoryapp.ErrInventoryQuantityInvalid)
		}

		uniqueIDs[item.ProductID] = struct{}{}
	}

	return nil
}

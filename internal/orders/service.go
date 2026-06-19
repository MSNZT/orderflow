package orders

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/MSNZT/orderflow/internal/cart"
	"github.com/MSNZT/orderflow/internal/inventory"
	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/google/uuid"
)

type CartProvider interface {
	GetSelectedItemsForCheckout(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) ([]cart.CheckoutItem, error)
	DeleteSelectedItems(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) error
}

type InventoryRepository interface {
	GetByProductIDsForUpdate(ctx context.Context, productIDs []uuid.UUID) ([]inventory.Inventory, error)
	DecreaseQuantity(ctx context.Context, productID uuid.UUID, requestedQuantity int) error
}

type Service struct {
	repo          *Repository
	inventoryRepo InventoryRepository
	cartService   CartProvider
	txManager     *postgres.TxManager
}

func NewService(
	repo *Repository,
	inventoryRepo InventoryRepository,
	cartService CartProvider,
	txManager *postgres.TxManager) *Service {
	return &Service{repo: repo, inventoryRepo: inventoryRepo, cartService: cartService, txManager: txManager}
}

func (s *Service) CreateOrder(ctx context.Context, userID uuid.UUID, productsIDs []uuid.UUID) (*Order, error) {
	const op = "orders.service.CreateOrder"

	if userID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if len(productsIDs) == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrProductIDsEmpty)
	}

	slices.SortFunc(productsIDs, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	var uniqueIDs = make(map[uuid.UUID]struct{}, len(productsIDs))

	for _, id := range productsIDs {
		if id == uuid.Nil {
			return nil, fmt.Errorf("%s: %w", op, ErrProductIDIsNil)
		}

		if _, exists := uniqueIDs[id]; exists {
			return nil, fmt.Errorf("%s: %w", op, ErrDuplicateProductID)
		}

		uniqueIDs[id] = struct{}{}
	}

	var order Order

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		lenProductsIDs := len(productsIDs)

		selectedItems, err := s.cartService.GetSelectedItemsForCheckout(txCtx, userID, productsIDs)
		if err != nil {
			return err
		}

		if len(selectedItems) != lenProductsIDs {
			return ErrCartChanged
		}

		var totalPriceCents int64
		var currency Currency = Currency(selectedItems[0].Currency)

		for _, selectedItem := range selectedItems {
			if !selectedItem.IsProductActive {
				return ErrProductInactive
			}

			if selectedItem.Quantity <= 0 {
				return ErrCartChanged
			}

			if currency != Currency(selectedItem.Currency) {
				return ErrCurrencyMismatch
			}

			totalPriceCents += int64(selectedItem.LineTotalPriceCents)
		}

		inventories, err := s.inventoryRepo.GetByProductIDsForUpdate(txCtx, productsIDs)
		if err != nil {
			return err
		}

		if len(inventories) < lenProductsIDs {
			return ErrInventoryNotFound
		}

		fmt.Println("====Inventories====", inventories)

		invMap := make(map[uuid.UUID]inventory.Inventory, len(inventories))

		for _, inv := range inventories {
			invMap[inv.ProductID] = inv
		}

		for _, item := range selectedItems {
			inv, exists := invMap[item.ProductID]
			if !exists {
				return ErrInventoryNotFound
			}

			available := inv.Quantity - inv.ReservedQuantity

			if available < int32(item.Quantity) {
				return ErrInsufficientStock
			}
		}

		id, err := uuid.NewV7()
		if err != nil {
			return ErrGenerateUUID
		}

		order = Order{
			ID:              id,
			UserID:          userID,
			Status:          StatusPending,
			Currency:        string(currency),
			TotalPriceCents: totalPriceCents,
		}

		var orderItems = make([]OrderItem, 0, len(selectedItems))

		for _, item := range selectedItems {
			id, err := uuid.NewV7()
			if err != nil {
				return ErrGenerateUUID
			}

			orderItem := OrderItem{
				ID:                  id,
				OrderID:             order.ID,
				ProductID:           item.ProductID,
				ProductName:         item.ProductName,
				UnitPriceCents:      item.UnitPriceCents,
				Currency:            string(currency),
				Quantity:            item.Quantity,
				LineTotalPriceCents: item.LineTotalPriceCents,
			}

			orderItems = append(orderItems, orderItem)
		}

		fmt.Println("====Order====", order, &order)

		if err := s.repo.CreateOrder(txCtx, &order); err != nil {
			return err
		}

		if err := s.repo.CreateOrderItems(txCtx, orderItems); err != nil {
			return err
		}

		for _, item := range selectedItems {
			if err := s.inventoryRepo.DecreaseQuantity(txCtx, item.ProductID, item.Quantity); err != nil {
				return err
			}
		}

		if err := s.cartService.DeleteSelectedItems(txCtx, userID, productsIDs); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &order, nil
}

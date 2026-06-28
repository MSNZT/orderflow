package orders

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/MSNZT/orderflow/internal/app/inventory"
	"github.com/MSNZT/orderflow/internal/cart"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

type CartProvider interface {
	GetSelectedItemsForCheckout(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) ([]cart.CheckoutItem, error)
	DeleteSelectedItems(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) error
}

type InventoryRepository interface {
	GetByProductIDsForUpdate(ctx context.Context, productIDs []uuid.UUID) ([]inventory.Inventory, error)
	ReserveQuantity(ctx context.Context, productID uuid.UUID, quantity int) error
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

func (s *Service) ListByUserID(ctx context.Context, userID uuid.UUID, page int, limit int) ([]Order, error) {
	const op = "orders.service.ListByUserID"

	if userID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	offset := (page - 1) * limit

	orders, err := s.repo.ListByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return orders, nil
}

func (s *Service) GetByID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*OrderDetails, error) {
	const op = "orders.service.GetByID"

	if userID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if orderID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrOrderIDIsNil)
	}

	orderDetails, err := s.repo.GetDetailsByIDAndUserID(ctx, userID, orderID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return orderDetails, nil
}

func (s *Service) CreateOrder(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) (*Order, error) {
	const op = "orders.service.CreateOrder"

	if userID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if len(productIDs) == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrProductIDsEmpty)
	}

	var sortedProductIDs = slices.Clone(productIDs)

	slices.SortFunc(sortedProductIDs, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	var uniqueIDs = make(map[uuid.UUID]struct{}, len(sortedProductIDs))

	for _, id := range sortedProductIDs {
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
		lenProductsIDs := len(sortedProductIDs)

		selectedItems, err := s.cartService.GetSelectedItemsForCheckout(txCtx, userID, sortedProductIDs)
		if err != nil {
			if errors.Is(err, cart.ErrCartNotFound) {
				return ErrCartChanged
			}
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

		inventories, err := s.inventoryRepo.GetByProductIDsForUpdate(txCtx, sortedProductIDs)
		if err != nil {
			return err
		}

		if len(inventories) < lenProductsIDs {
			return ErrInventoryNotFound
		}

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

		if err := s.repo.CreateOrder(txCtx, &order); err != nil {
			return err
		}

		if err := s.repo.CreateOrderItems(txCtx, orderItems); err != nil {
			return err
		}

		for _, item := range selectedItems {
			if err := s.inventoryRepo.ReserveQuantity(txCtx, item.ProductID, item.Quantity); err != nil {
				if errors.Is(err, inventory.ErrInsufficientStock) {
					return ErrInsufficientStock
				}
				return err
			}
		}

		if err := s.cartService.DeleteSelectedItems(txCtx, userID, sortedProductIDs); err != nil {
			if errors.Is(err, cart.ErrCartItemNotFound) || errors.Is(err, cart.ErrCartNotFound) {
				return ErrCartChanged
			}
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &order, nil
}

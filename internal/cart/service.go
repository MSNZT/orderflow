package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/MSNZT/orderflow/internal/products"
	"github.com/google/uuid"
)

type ProductsProvider interface {
	GetByID(ctx context.Context, productID uuid.UUID) (*products.Product, error)
}

type Service struct {
	repo            *Repository
	txManager       *postgres.TxManager
	productsService ProductsProvider
}

func NewService(repo *Repository, txManager *postgres.TxManager, productService ProductsProvider) *Service {
	return &Service{repo: repo, txManager: txManager, productsService: productService}
}

type getItemsInput struct {
	UserID uuid.UUID
	Limit  int32
	Page   int32
}

type addItemInput struct {
	UserID    uuid.UUID
	ProductID uuid.UUID
	Quantity  int32
}

type updateItemQuantityInput struct {
	UserID    uuid.UUID
	ProductID uuid.UUID
	Quantity  int32
}

func (s *Service) GetItems(ctx context.Context, input getItemsInput) (*Cart, error) {
	const op = "cart.service.GetItems"

	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if input.Limit <= 0 {
		input.Limit = 20
	}

	if input.Page <= 0 {
		input.Page = 1
	}

	pageOffset := (input.Page - 1) * input.Limit
	cartItems, err := s.repo.GetItems(ctx, input.UserID, input.Limit, pageOffset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	cartResponse := toCartResponse(cartItems)

	return cartResponse, nil
}

func (s *Service) AddItem(ctx context.Context, input addItemInput) error {
	const op = "cart.service.AddItem"

	if input.UserID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if input.ProductID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrProductIDIsNil)
	}

	if input.Quantity <= 0 {
		return fmt.Errorf("%s: %w", op, ErrQuantityInvalid)
	}

	product, err := s.productsService.GetByID(ctx, input.ProductID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			return fmt.Errorf("%s: %w", op, ErrProductNotAvailable)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	err = s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		cartID, err := s.repo.GetOrCreateByUserID(txCtx, input.UserID)
		if err != nil {
			return err
		}

		if err := s.repo.AddItem(txCtx, cartID, product.ID, input.Quantity); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) UpdateItemQuantity(ctx context.Context, input updateItemQuantityInput) error {
	const op = "cart.service.UpdateItemQuantity"

	if input.UserID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if input.ProductID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrProductIDIsNil)
	}

	if input.Quantity <= 0 {
		return fmt.Errorf("%s: %w", op, ErrQuantityInvalid)
	}

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		cartID, err := s.repo.GetByUserID(txCtx, input.UserID)
		if err != nil {
			if errors.Is(err, ErrCartNotFound) {
				return ErrCartItemNotFound
			}
			return err
		}

		err = s.repo.UpdateItemQuantity(txCtx, cartID, input.ProductID, input.Quantity)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) DeleteItem(ctx context.Context, userID uuid.UUID, productID uuid.UUID) error {
	const op = "cart.service.DeleteItem"

	if userID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if productID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrProductIDIsNil)
	}

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		cartID, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			if errors.Is(err, ErrCartNotFound) {
				return nil
			}

			return err
		}

		if err = s.repo.DeleteItem(txCtx, cartID, productID); err != nil {
			if errors.Is(err, ErrCartItemNotFound) {
				return nil
			}

			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func toCartResponse(cartItems []CartItem) *Cart {
	var totalPriceCents int64
	for _, item := range cartItems {
		totalPriceCents += item.LineTotalPriceCents
	}

	return &Cart{
		Items:           cartItems,
		TotalPriceCents: totalPriceCents,
	}
}

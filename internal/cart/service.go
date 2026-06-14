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

type listInput struct {
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

func (s *Service) List(ctx context.Context, input listInput) (*Cart, error) {
	const op = "cart.service.List"

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

func (s *Service) UpdateItemQuantity(ctx context.Context, input updateItemQuantityInput) (*Cart, error) {
	const op = "cart.service.UpdateItemQuantity"

	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if input.ProductID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrProductIDIsNil)
	}

	if input.Quantity <= 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrQuantityInvalid)
	}

	var cart *Cart

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		cartID, err := s.repo.GetOrCreateByUserID(txCtx, input.UserID)
		if err != nil {
			return err
		}

		err = s.repo.UpdateItemQuantity(txCtx, cartID, input.ProductID, input.Quantity)
		if err != nil {
			return err
		}

		cartItems, err := s.repo.GetItems(txCtx, input.UserID, int32(100), int32(0))

		cart = toCartResponse(cartItems)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return cart, nil
}

func toCartResponse(cartItems []CartItem) *Cart {
	var totalPriceCents int64
	for _, item := range cartItems {
		totalPriceCents += item.PriceCents * int64(item.Quantity)
	}

	return &Cart{
		Items:           cartItems,
		TotalPriceCents: totalPriceCents,
	}
}

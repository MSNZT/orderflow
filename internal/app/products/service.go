package products

import (
	"context"
	"fmt"
	"strings"

	"github.com/MSNZT/orderflow/internal/app/transaction"
	"github.com/google/uuid"
)

type Service struct {
	productRepo   Repository
	inventoryRepo InventoryRepository
	txManager     transaction.Manager
}

type CreateInput struct {
	Name            string
	Description     *string
	PriceCents      int64
	Currency        string
	InitialQuantity int32
}

func NewService(
	productRepo Repository,
	inventoryRepo InventoryRepository,
	txManager transaction.Manager,
) *Service {
	return &Service{productRepo: productRepo, inventoryRepo: inventoryRepo, txManager: txManager}
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	const op = "products.Service.List"

	products, err := s.productRepo.ListActive(ctx)
	if err != nil {
		return products, fmt.Errorf("%s: %w", op, err)
	}

	return products, nil
}

func (s *Service) GetByID(ctx context.Context, productID uuid.UUID) (*Product, error) {
	const op = "products.Service.GetByID"

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return product, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Product, error) {
	const op = "products.service.Create"

	if input.Currency == "" {
		input.Currency = "RUB"
	}

	name := strings.TrimSpace(input.Name)
	if len(name) <= 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrProductNameInvalid)
	}

	currency := strings.TrimSpace(input.Currency)
	if input.Currency != "RUB" {
		return nil, fmt.Errorf("%s: %w", op, ErrProductCurrencyInvalid)
	}

	if input.PriceCents <= 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrProductPriceCentsInvalid)
	}

	if input.InitialQuantity < 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrInitialQuantityInvalid)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to generate uuid: %w", op, err)
	}

	product := Product{
		ID:          id,
		Name:        name,
		Description: input.Description,
		Currency:    currency,
		PriceCents:  input.PriceCents,
		IsActive:    true,
	}

	var createdProduct *Product

	err = s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		var err error

		createdProduct, err = s.productRepo.Create(txCtx, &product)
		if err != nil {
			return err
		}

		err = s.inventoryRepo.Create(txCtx, createdProduct.ID, input.InitialQuantity)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return createdProduct, nil
}

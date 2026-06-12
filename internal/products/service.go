package products

import (
	"context"
	"fmt"

	"github.com/MSNZT/orderflow/internal/inventory"
	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/google/uuid"
)

type Service struct {
	productRepo   *Repository
	inventoryRepo *inventory.Repository
	txManager     *postgres.TxManager
}

func NewService(
	productRepo *Repository,
	inventoryRepo *inventory.Repository,
	txManager *postgres.TxManager,
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

func (s *Service) Create(ctx context.Context, product *Product, quantity int32) (*Product, error) {
	const op = "products.service.Create"

	if product.PriceCents <= 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrProductPriceCentsInvalid)
	}

	if product.Currency != "" && product.Currency != "RUB" && product.Currency != "USD" {
		return nil, fmt.Errorf("%s: %w", op, ErrProductCurrencyInvalid)
	}

	var createdProduct *Product

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		var err error

		createdProduct, err = s.productRepo.Create(txCtx, product)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = s.inventoryRepo.Create(txCtx, createdProduct.ID, quantity)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return createdProduct, nil
}

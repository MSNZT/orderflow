package products

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	const op = "products.Service.List"

	products, err := s.repo.ListActive(ctx)
	if err != nil {
		return products, fmt.Errorf("%s: %w", op, err)
	}

	return products, nil
}

func (s *Service) GetByID(ctx context.Context, productID uuid.UUID) (*Product, error) {
	const op = "products.Service.GetByID"

	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return product, fmt.Errorf("%s: %w", op, err)
	}

	return product, nil
}

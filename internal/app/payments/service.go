package payments

import (
	"github.com/MSNZT/orderflow/internal/app/transaction"
)

type Service struct {
	repo              Repository
	ordersProvider    OrdersProvider
	paymentProvider   PaymentProvider
	inventoryProvider InventoryProvider
	txManager         transaction.Manager
}

func NewService(
	repo Repository, ordersProvider OrdersProvider, paymentProvider PaymentProvider,
	inventoryProvider InventoryProvider, txManager transaction.Manager,
) *Service {
	return &Service{
		repo: repo, ordersProvider: ordersProvider, paymentProvider: paymentProvider,
		inventoryProvider: inventoryProvider, txManager: txManager,
	}
}

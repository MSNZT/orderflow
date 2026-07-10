package payments

import (
	"github.com/MSNZT/orderflow/internal/app/inventory"
	"github.com/MSNZT/orderflow/internal/app/orders"
)

func reservedItemsFromOrder(orderDetails *orders.OrderDetails) []inventory.ReservedItem {
	reservedItems := make([]inventory.ReservedItem, 0, len(orderDetails.Items))
	for _, item := range orderDetails.Items {
		reservedItem := inventory.ReservedItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}

		reservedItems = append(reservedItems, reservedItem)
	}

	return reservedItems
}

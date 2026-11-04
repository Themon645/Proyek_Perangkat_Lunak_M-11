package resource

import (
	"testing"

	"m11/model"
)

func TestNewPurchaseOrderMapsTotalsAndReceivedQuantity(t *testing.T) {
	result := NewPurchaseOrder(model.PurchaseOrder{
		ID: "PO-1",
		Items: []model.PurchaseOrderItem{{
			ID: "ITEM-1", Name: "Laptop", Quantity: 2, UnitCost: 100, Received: 1,
		}},
	})

	if result.Total != 200 || len(result.Items) != 1 || result.Items[0].QtyReceived != 1 {
		t.Fatalf("unexpected purchase order resource: %+v", result)
	}
}

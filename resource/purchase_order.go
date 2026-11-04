package resource

import "m11/model"

type PurchaseOrder struct {
	ID       string              `json:"id"`
	Supplier string              `json:"supplier"`
	Status   string              `json:"status"`
	Payment  string              `json:"payment"`
	Proof    string              `json:"proof,omitempty"`
	Total    int64               `json:"total"`
	Items    []PurchaseOrderItem `json:"items"`
}

type PurchaseOrderItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	UnitCost    int64  `json:"unit_cost"`
	QtyReceived int    `json:"qty_received"`
}

func NewPurchaseOrder(po model.PurchaseOrder) PurchaseOrder {
	result := PurchaseOrder{
		ID:       po.ID,
		Supplier: po.Supplier,
		Status:   po.Status,
		Payment:  po.Payment,
		Proof:    po.Proof,
		Items:    make([]PurchaseOrderItem, 0, len(po.Items)),
	}
	for _, item := range po.Items {
		result.Total += int64(item.Quantity) * item.UnitCost
		result.Items = append(result.Items, PurchaseOrderItem{
			ID: item.ID, Name: item.Name, Quantity: item.Quantity,
			UnitCost: item.UnitCost, QtyReceived: item.Received,
		})
	}
	return result
}

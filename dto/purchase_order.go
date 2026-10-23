package dto

type CreatePurchaseOrderRequest struct {
	ID       string                    `json:"id"`
	Supplier string                    `json:"supplier"`
	Status   string                    `json:"status"`
	Items    []CreatePurchaseOrderItem `json:"items"`
}

type CreatePurchaseOrderItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	UnitCost int64  `json:"unit_cost"`
}

type ReceivePurchaseOrderRequest struct {
	Items []ReceiveItem `json:"items"`
}

type ReceiveItem struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity"`
}

type PayPurchaseOrderRequest struct {
	Proof string `json:"proof"`
}

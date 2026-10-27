package dto

type CreateGoodsReceiptRequest struct {
	PurchaseOrderID string                   `json:"purchase_order_id"`
	ReceiptNo       string                   `json:"receipt_no"`
	ReceivedDate    string                   `json:"received_date"`
	Receiver        string                   `json:"receiver"`
	Notes           string                   `json:"notes"`
	Items           []CreateGoodsReceiptItem `json:"items"`
}

type CreateGoodsReceiptItem struct {
	ItemID      string `json:"item_id"`
	QtyReceived int    `json:"qty_received"`
}

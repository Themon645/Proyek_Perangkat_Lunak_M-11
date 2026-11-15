package resource

import "m11/model"

type GoodsReceiptHistory struct {
	PurchaseOrderID string                `json:"purchase_order_id"`
	Receipts        []GoodsReceiptSummary `json:"receipts"`
}

type GoodsReceiptSummary struct {
	ReceiptNo    string                    `json:"receipt_no"`
	ReceivedDate string                    `json:"received_date"`
	Receiver     string                    `json:"receiver"`
	Notes        string                    `json:"notes,omitempty"`
	Items        []GoodsReceiptItemSummary `json:"items"`
}

type GoodsReceiptItemSummary struct {
	ItemID      string `json:"item_id"`
	ItemName    string `json:"item_name"`
	QtyReceived int    `json:"qty_received"`
}

func NewGoodsReceiptHistory(po *model.PurchaseOrder, receipts []model.GoodsReceipt) GoodsReceiptHistory {
	result := GoodsReceiptHistory{
		PurchaseOrderID: po.ID,
		Receipts:        make([]GoodsReceiptSummary, 0, len(receipts)),
	}
	for _, receipt := range receipts {
		summary := GoodsReceiptSummary{
			ReceiptNo:    receipt.ReceiptNo,
			ReceivedDate: receipt.ReceivedDate.Format("2006-01-02"),
			Receiver:     receipt.Receiver,
			Notes:        receipt.Notes,
			Items:        make([]GoodsReceiptItemSummary, 0, len(receipt.Items)),
		}
		for _, item := range receipt.Items {
			summary.Items = append(summary.Items, GoodsReceiptItemSummary{
				ItemID: item.Item.ID, ItemName: item.Item.Name, QtyReceived: item.QtyReceived,
			})
		}
		result.Receipts = append(result.Receipts, summary)
	}
	return result
}

type GoodsReceiptResponse struct {
	ReceiptNo       string                    `json:"receipt_no"`
	PurchaseOrderID string                    `json:"purchase_order_id"`
	ReceivedDate    string                    `json:"received_date"`
	Receiver        string                    `json:"receiver"`
	Notes           string                    `json:"notes,omitempty"`
	Items           []GoodsReceiptItemSummary `json:"items"`
}

func NewGoodsReceiptResponse(receipt model.GoodsReceipt) GoodsReceiptResponse {
	poID := receipt.PurchaseOrderID
	if poID == "" && receipt.PurchaseOrder.ID != "" {
		poID = receipt.PurchaseOrder.ID
	}
	res := GoodsReceiptResponse{
		ReceiptNo:       receipt.ReceiptNo,
		PurchaseOrderID: poID,
		ReceivedDate:    receipt.ReceivedDate.Format("2006-01-02"),
		Receiver:        receipt.Receiver,
		Notes:           receipt.Notes,
		Items:           make([]GoodsReceiptItemSummary, 0, len(receipt.Items)),
	}
	for _, item := range receipt.Items {
		itemID := item.Item.ID
		itemName := item.Item.Name
		res.Items = append(res.Items, GoodsReceiptItemSummary{
			ItemID:      itemID,
			ItemName:    itemName,
			QtyReceived: item.QtyReceived,
		})
	}
	return res
}

package resource

import (
	"fmt"

	"m11/model"
)

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
	NamaBarang  string `json:"nama_barang,omitempty"`
	QtyReceived int    `json:"qty_received"`
	QtyMasuk    int    `json:"qty_masuk"`
	Unit        string `json:"unit"`
	Satuan      string `json:"satuan,omitempty"`
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
				ItemID:      item.Item.ID,
				ItemName:    item.Item.Name,
				NamaBarang:  item.Item.Name,
				QtyReceived: item.QtyReceived,
				QtyMasuk:    item.QtyReceived,
				Unit:        "Unit",
				Satuan:      "Unit",
			})
		}
		result.Receipts = append(result.Receipts, summary)
	}
	return result
}

type GoodsReceiptResponse struct {
	ID                string                    `json:"id"`
	ReceiptNo         string                    `json:"receipt_no"`
	NomorGRN          string                    `json:"nomor_grn,omitempty"`
	NomorSuratJalan   string                    `json:"nomor_surat_jalan,omitempty"`
	PurchaseOrderID   string                    `json:"purchase_order_id"`
	POID              string                    `json:"po_id,omitempty"`
	ReceivedDate      string                    `json:"received_date"`
	TanggalPenerimaan string                    `json:"tanggal_penerimaan,omitempty"`
	Receiver          string                    `json:"receiver"`
	Penerima          string                    `json:"penerima,omitempty"`
	Notes             string                    `json:"notes,omitempty"`
	Catatan           string                    `json:"catatan,omitempty"`
	Items             []GoodsReceiptItemSummary `json:"items"`
}

func NewGoodsReceiptResponse(receipt model.GoodsReceipt) GoodsReceiptResponse {
	poID := receipt.PurchaseOrderID
	if poID == "" && receipt.PurchaseOrder.ID != "" {
		poID = receipt.PurchaseOrder.ID
	}
	dateStr := receipt.ReceivedDate.Format("2006-01-02")
	idStr := receipt.ReceiptNo
	if idStr == "" {
		idStr = fmt.Sprintf("GRN-%s", receipt.UUID.String()[:8])
	}
	res := GoodsReceiptResponse{
		ID:                idStr,
		ReceiptNo:         receipt.ReceiptNo,
		NomorGRN:          receipt.ReceiptNo,
		NomorSuratJalan:   receipt.ReceiptNo,
		PurchaseOrderID:   poID,
		POID:              poID,
		ReceivedDate:      dateStr,
		TanggalPenerimaan: dateStr,
		Receiver:          receipt.Receiver,
		Penerima:          receipt.Receiver,
		Notes:             receipt.Notes,
		Catatan:           receipt.Notes,
		Items:             make([]GoodsReceiptItemSummary, 0, len(receipt.Items)),
	}
	for _, item := range receipt.Items {
		itemID := item.Item.ID
		itemName := item.Item.Name
		res.Items = append(res.Items, GoodsReceiptItemSummary{
			ItemID:      itemID,
			ItemName:    itemName,
			NamaBarang:  itemName,
			QtyReceived: item.QtyReceived,
			QtyMasuk:    item.QtyReceived,
			Unit:        "Unit",
			Satuan:      "Unit",
		})
	}
	return res
}

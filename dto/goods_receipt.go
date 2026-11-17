package dto

import (
	"errors"
	"strconv"
	"strings"
)

type StrictInt int

func (s *StrictInt) UnmarshalJSON(b []byte) error {
	raw := strings.Trim(string(b), "\"")
	if strings.Contains(raw, ".") || strings.Contains(raw, ",") || strings.ContainsAny(raw, "eE") {
		return errors.New("qty harus bilangan bulat, tidak boleh desimal")
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return errors.New("qty harus berupa angka")
	}
	*s = StrictInt(val)
	return nil
}

func (s StrictInt) Int() int {
	return int(s)
}

type CreateGoodsReceiptRequest struct {
	PurchaseOrderID string                   `json:"purchase_order_id"`
	ReceiptNo       string                   `json:"receipt_no"`
	ReceivedDate    string                   `json:"received_date"`
	Receiver        string                   `json:"receiver"`
	Notes           string                   `json:"notes"`
	Items           []CreateGoodsReceiptItem `json:"items"`
}

type CreateGoodsReceiptItem struct {
	ItemID      string    `json:"item_id"`
	QtyReceived StrictInt `json:"qty_received"`
}

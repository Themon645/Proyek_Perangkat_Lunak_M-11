package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"m11/application"
	"m11/common/helper"
	"m11/dto"
)

type GoodsReceiptController struct {
	service *application.PurchaseOrderService
}

func NewGoodsReceiptController(service *application.PurchaseOrderService) http.Handler {
	c := &GoodsReceiptController{service: service}
	return http.HandlerFunc(c.create)
}

func (c *GoodsReceiptController) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req dto.CreateGoodsReceiptRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.PurchaseOrderID) == "" ||
		strings.TrimSpace(req.ReceiptNo) == "" ||
		len(req.Items) == 0 {
		validationErrorJSON(w, "REQUIRED_FIELDS", "purchase_order_id, receipt_no, and items are required", map[string]string{
			"purchase_order_id": "purchase_order_id wajib diisi",
			"receipt_no":        "receipt_no wajib diisi",
			"items":             "items wajib diisi",
		})
		return
	}
	receivedDate, err := time.Parse("2006-01-02", req.ReceivedDate)
	if err != nil {
		validationErrorJSON(w, "INVALID_DATE", "received_date harus menggunakan format YYYY-MM-DD", map[string]string{
			"received_date": "received_date harus menggunakan format YYYY-MM-DD",
		})
		return
	}
	seen := make(map[string]bool, len(req.Items))
	for _, item := range req.Items {
		if strings.TrimSpace(item.ItemID) == "" || item.QtyReceived <= 0 || seen[item.ItemID] {
			validationErrorJSON(w, "INVALID_ITEMS", "item_id wajib unik dan qty_received harus lebih besar dari 0", map[string]string{
				"items": "item_id wajib unik dan qty_received harus lebih besar dari 0",
			})
			return
		}
		seen[item.ItemID] = true
	}
	receipt, err := c.service.CreateGoodsReceipt(&req, receivedDate)
	if err != nil {
		helper.LogTransaction("ERROR", "GOODS_RECEIPT", "create failed receipt_no=%s: %v", req.ReceiptNo, err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			errorJSON(w, http.StatusNotFound, "purchase order not found")
		case strings.Contains(err.Error(), "must be approved"):
			errorJSON(w, http.StatusConflict, err.Error())
		case strings.HasPrefix(err.Error(), "Jumlah diterima melebihi sisa pesanan barang"):
			validationErrorJSON(w, "QUANTITY_EXCEEDS_REMAINING", err.Error(), map[string]string{"items": err.Error()})
		default:
			errorJSON(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	helper.LogTransaction("CREATE", "GOODS_RECEIPT", "receipt_no=%s purchase_order_id=%s", receipt.ReceiptNo, req.PurchaseOrderID)
	jsonResponse(w, http.StatusCreated, receipt)
}

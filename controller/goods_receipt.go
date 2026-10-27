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
		errorJSON(w, http.StatusBadRequest, "purchase_order_id, receipt_no, and items are required")
		return
	}
	receivedDate, err := time.Parse("2006-01-02", req.ReceivedDate)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "received_date must use YYYY-MM-DD")
		return
	}
	seen := make(map[string]bool, len(req.Items))
	for _, item := range req.Items {
		if strings.TrimSpace(item.ItemID) == "" || item.QtyReceived <= 0 || seen[item.ItemID] {
			errorJSON(w, http.StatusBadRequest, "each item_id must be unique and qty_received must be greater than zero")
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
		default:
			errorJSON(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	helper.LogTransaction("CREATE", "GOODS_RECEIPT", "receipt_no=%s purchase_order_id=%s", receipt.ReceiptNo, req.PurchaseOrderID)
	jsonResponse(w, http.StatusCreated, receipt)
}

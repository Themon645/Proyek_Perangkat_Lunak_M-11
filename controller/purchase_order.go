package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"m11/application"
	"m11/common"
	"m11/common/helper"
	"m11/dto"
	"m11/model"
	"m11/resource"

	"github.com/google/uuid"
)

type PurchaseOrderController struct {
	service *application.PurchaseOrderService
}

func NewPurchaseOrderController(service *application.PurchaseOrderService) *PurchaseOrderController {
	return &PurchaseOrderController{service: service}
}

func (c *PurchaseOrderController) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	var orders []model.PurchaseOrder
	var err error
	if status != "" {
		orders, err = c.service.ListByStatus(status)
	} else {
		orders, err = c.service.List()
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]resource.PurchaseOrder, 0, len(orders))
	for _, order := range orders {
		result = append(result, resource.NewPurchaseOrder(order))
	}
	jsonValue(w, http.StatusOK, result)
}

func (c *PurchaseOrderController) Approved(w http.ResponseWriter, r *http.Request) {
	orders, err := c.service.Approved()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]resource.PurchaseOrder, 0, len(orders))
	for _, order := range orders {
		result = append(result, resource.NewPurchaseOrder(order))
	}
	jsonValue(w, http.StatusOK, result)
}

func (c *PurchaseOrderController) Find(w http.ResponseWriter, r *http.Request) {
	order, err := c.service.Find(r.PathValue("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		errorJSON(w, http.StatusNotFound, "purchase order not found")
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonValue(w, http.StatusOK, resource.NewPurchaseOrder(*order))
}

func (c *PurchaseOrderController) ReceiptHistory(w http.ResponseWriter, r *http.Request) {
	po, receipts, err := c.service.ReceiptHistory(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorJSON(w, http.StatusNotFound, "purchase order not found")
			return
		}
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, resource.NewGoodsReceiptHistory(po, receipts))
}

func (c *PurchaseOrderController) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePurchaseOrderRequest
	if !decodeJSON(w, r, &req) || req.Supplier == "" || len(req.Items) == 0 {
		return
	}
	if req.ID == "" {
		req.ID = "PO-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	if req.Status == "" {
		req.Status = common.StatusPending
	}
	for i := range req.Items {
		if req.Items[i].ID == "" || req.Items[i].Quantity <= 0 || req.Items[i].UnitCost < 0 {
			errorJSON(w, http.StatusBadRequest, "invalid purchase order item")
			return
		}
	}
	poUUID := uuid.New()
	items := make([]model.PurchaseOrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = model.PurchaseOrderItem{
			ID: item.ID, PurchaseOrderUUID: poUUID, Name: item.Name,
			Quantity: item.Quantity, UnitCost: item.UnitCost,
		}
	}
	po := &model.PurchaseOrder{ID: req.ID, Supplier: req.Supplier, Status: req.Status, Payment: common.PaymentUnpaid, Items: items}
	po.UUID = poUUID
	if err := c.service.Create(po); err != nil {
		helper.LogTransaction("ERROR", "PURCHASE_ORDER", "create failed: %v", err)
		errorJSON(w, http.StatusConflict, "purchase order already exists")
		return
	}
	helper.LogTransaction("CREATE", "PURCHASE_ORDER", "id=%s supplier=%s", po.ID, po.Supplier)
	jsonResponse(w, http.StatusCreated, po)
}

func (c *PurchaseOrderController) Receive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req dto.ReceivePurchaseOrderRequest
	if !decodeJSON(w, r, &req) || len(req.Items) == 0 {
		return
	}
	received := make(map[string]int, len(req.Items))
	for _, item := range req.Items {
		received[item.ID] += item.Quantity
	}
	po, err := c.service.Receive(id, received)
	if err != nil {
		helper.LogTransaction("ERROR", "PURCHASE_ORDER", "receive failed id=%s: %v", id, err)
		c.repositoryError(w, err)
		return
	}
	helper.LogTransaction("UPDATE", "PURCHASE_ORDER", "received id=%s", id)
	jsonResponse(w, http.StatusOK, po)
}

func (c *PurchaseOrderController) Pay(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var proofPath string
	var uploaded bool
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(helper.MaxPaymentProofSize + (1 << 20)); err != nil {
			paymentProofError(w, "format multipart/form-data tidak valid")
			return
		}
		file, header, err := r.FormFile("proof")
		if err != nil {
			paymentProofError(w, "berkas bukti pembayaran wajib diisi")
			return
		}
		proofPath, err = helper.SavePaymentProof(file, header)
		_ = file.Close()
		if err != nil {
			paymentProofError(w, err.Error())
			return
		}
		uploaded = true
	} else {
		var req dto.PayPurchaseOrderRequest
		if !decodeJSON(w, r, &req) || strings.TrimSpace(req.Proof) == "" {
			return
		}
		proofPath = req.Proof
	}
	po, err := c.service.Pay(id, proofPath)
	if err != nil {
		if uploaded {
			helper.DeletePaymentProof(proofPath)
		}
		helper.LogTransaction("ERROR", "PAYMENT", "payment failed id=%s: %v", id, err)
		c.repositoryError(w, err)
		return
	}
	helper.LogTransaction("UPDATE", "PAYMENT", "paid purchase_order_id=%s", id)
	jsonResponse(w, http.StatusOK, po)
}

func paymentProofError(w http.ResponseWriter, message string) {
	writeErrorJSON(w, http.StatusBadRequest, "INVALID_PAYMENT_PROOF", message, map[string]string{"proof": message})
}

func (c *PurchaseOrderController) Unpaid(w http.ResponseWriter, r *http.Request) {
	rows, total, err := c.service.Unpaid()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonValue(w, http.StatusOK, map[string]any{"purchase_orders": rows, "total": total})
}

func (c *PurchaseOrderController) Report(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "from must use YYYY-MM-DD")
		return
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "to must use YYYY-MM-DD")
		return
	}
	totals, err := c.service.Report(from, to)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonValue(w, http.StatusOK, map[string]any{"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"), "suppliers": totals})
}

func (c *PurchaseOrderController) repositoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		errorJSON(w, http.StatusNotFound, "purchase order not found")
	case strings.Contains(err.Error(), "must be approved"),
		strings.Contains(err.Error(), "must be received"),
		strings.Contains(err.Error(), "sudah dibayar"):
		errorJSON(w, http.StatusConflict, err.Error())
	default:
		errorJSON(w, http.StatusBadRequest, err.Error())
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		errorJSON(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    value,
	})
}

func jsonValue(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func errorJSON(w http.ResponseWriter, status int, message string) {
	code := "BAD_REQUEST"
	switch status {
	case http.StatusNotFound:
		code = "NOT_FOUND"
	case http.StatusConflict:
		code = "CONFLICT"
	case http.StatusUnsupportedMediaType:
		code = "UNSUPPORTED_MEDIA_TYPE"
	case http.StatusUnprocessableEntity:
		code = "VALIDATION_ERROR"
	case http.StatusInternalServerError:
		code = "INTERNAL_SERVER_ERROR"
	}
	writeErrorJSON(w, status, code, message, nil)
}

func validationErrorJSON(w http.ResponseWriter, code, message string, fields map[string]string) {
	writeErrorJSON(w, http.StatusUnprocessableEntity, code, message, fields)
}

func writeErrorJSON(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	err := map[string]any{
		"code":    code,
		"message": message,
	}
	if fields != nil {
		err["fields"] = fields
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   err,
	})
}

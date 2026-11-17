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
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		q = strings.TrimSpace(r.URL.Query().Get("search"))
	}

	orders, err := c.service.ListFiltered(status, date, from, to, q)
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

	// Pastikan PO ada sebelum memproses bukti bayar
	poExisting, err := c.service.Find(id)
	if err != nil || poExisting == nil {
		errorJSON(w, http.StatusNotFound, "purchase order not found")
		return
	}

	// Poin 1: Pelunasan hanya dapat diproses setelah seluruh barang pesanan 100% lengkap diterima di gudang (status: received)
	if poExisting.Status != common.StatusReceived {
		writeErrorJSON(w, http.StatusUnprocessableEntity, "PO_NOT_FULLY_RECEIVED", "Pelunasan tagihan hanya dapat diproses setelah seluruh barang pesanan 100% lengkap diterima di gudang.", map[string]string{
			"status": "Pelunasan tagihan hanya dapat diproses setelah seluruh barang pesanan 100% lengkap diterima di gudang.",
		})
		return
	}

	var proofPath string
	var uploaded bool
	var paidAtStr string
	var bank string

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

		paidAtStr = r.FormValue("paid_at")
		if paidAtStr == "" {
			paidAtStr = r.FormValue("payment_date")
		}
		if paidAtStr == "" {
			paidAtStr = r.FormValue("date")
		}
		bank = r.FormValue("bank")
		if bank == "" {
			bank = r.FormValue("bank_tujuan")
		}
	} else {
		var req dto.PayPurchaseOrderRequest
		if !decodeJSON(w, r, &req) || strings.TrimSpace(req.Proof) == "" {
			return
		}
		proofPath = req.Proof
		paidAtStr = req.PaidAt
		if paidAtStr == "" {
			paidAtStr = req.PaymentDate
		}
		bank = req.Bank
		if bank == "" {
			bank = req.BankTujuan
		}
	}

	// D08: Validasi tanggal pelunasan (tidak boleh masa depan, tidak boleh sebelum PO)
	paidTime := time.Now().UTC()
	if strings.TrimSpace(paidAtStr) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(paidAtStr))
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, strings.TrimSpace(paidAtStr))
		}
		if err != nil {
			if uploaded {
				helper.DeletePaymentProof(proofPath)
			}
			paymentProofError(w, "format tanggal pembayaran harus YYYY-MM-DD")
			return
		}
		paidTime = parsed
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	poOrderDate := time.Date(poExisting.CreatedAt.Year(), poExisting.CreatedAt.Month(), poExisting.CreatedAt.Day(), 0, 0, 0, 0, time.UTC)

	// Poin 3: Validasi tanggal pembayaran
	if paidTime.After(today) || paidTime.Before(poOrderDate) {
		if uploaded {
			helper.DeletePaymentProof(proofPath)
		}
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_PAYMENT_DATE", "Tanggal pembayaran tidak boleh melebihi hari ini atau lebih awal dari tanggal PO.", map[string]string{
			"paid_at": "Tanggal pembayaran tidak boleh melebihi hari ini atau lebih awal dari tanggal PO.",
		})
		return
	}

	if bank == "" {
		bank = "Bank Mandiri (1310-0982-4412 a.n. Vendor)"
	}

	po, err := c.service.Pay(id, proofPath, paidTime, bank)
	if err != nil {
		if uploaded {
			helper.DeletePaymentProof(proofPath)
		}
		helper.LogTransaction("ERROR", "PAYMENT", "payment failed id=%s: %v", id, err)
		c.repositoryError(w, err)
		return
	}
	helper.LogTransaction("UPDATE", "PAYMENT", "paid purchase_order_id=%s", id)
	jsonResponse(w, http.StatusOK, resource.NewPurchaseOrder(*po))
}

func (c *PurchaseOrderController) Proof(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	po, err := c.service.Find(id)
	if err != nil || po == nil {
		errorJSON(w, http.StatusNotFound, "purchase order not found")
		return
	}
	proof := strings.TrimSpace(po.Proof)
	if proof == "" && len(po.Payments) > 0 {
		proof = strings.TrimSpace(po.Payments[len(po.Payments)-1].ProofPath)
	}
	if proof == "" {
		errorJSON(w, http.StatusNotFound, "bukti pembayaran belum tersedia")
		return
	}
	_, _, cloudURL := helper.ResolveProofURLs(proof)
	http.Redirect(w, r, cloudURL, http.StatusFound)
}

func (c *PurchaseOrderController) StorageFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if strings.TrimSpace(filename) == "" {
		errorJSON(w, http.StatusNotFound, "file not found")
		return
	}
	_, _, cloudURL := helper.ResolveProofURLs(filename)
	http.Redirect(w, r, cloudURL, http.StatusFound)
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
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		errorJSON(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		if strings.Contains(err.Error(), "tidak boleh desimal") {
			validationErrorJSON(w, "INVALID_INTEGER_QTY", "Kuantitas penerimaan barang harus berupa bilangan bulat positif.", map[string]string{
				"items":        "Kuantitas penerimaan barang harus berupa bilangan bulat positif.",
				"qty_received": "Kuantitas penerimaan barang harus berupa bilangan bulat positif.",
			})
			return false
		}
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

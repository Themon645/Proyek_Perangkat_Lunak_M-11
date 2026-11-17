package controller

import (
	"errors"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"m11/application"
	"m11/common/helper"
	"m11/dto"
	"m11/resource"
)

var reScriptTag = regexp.MustCompile(`(?i)<script[\s\S]*?>[\s\S]*?</script>|<script[\s\S]*?>`)

func sanitizeInput(s string) string {
	cleaned := reScriptTag.ReplaceAllString(s, "")
	cleaned = html.EscapeString(strings.TrimSpace(cleaned))
	return cleaned
}

type GoodsReceiptController struct {
	service *application.PurchaseOrderService
}

func NewGoodsReceiptController(service *application.PurchaseOrderService) *GoodsReceiptController {
	return &GoodsReceiptController{service: service}
}

func (c *GoodsReceiptController) Create(w http.ResponseWriter, r *http.Request) {
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

	// B07: Pembatasan nomor surat jalan maksimal 20 karakter
	rawReceiptNo := strings.TrimSpace(req.ReceiptNo)
	if len([]rune(rawReceiptNo)) > 20 {
		validationErrorJSON(w, "INVALID_RECEIPT_NO", "nomor surat jalan maksimal 20 karakter", map[string]string{
			"receipt_no": "nomor surat jalan maksimal 20 karakter",
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

	// Poin 3: Validasi kronologi tanggal penerimaan (tidak boleh masa depan, tidak boleh sebelum PO)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	if receivedDate.After(today) {
		validationErrorJSON(w, "INVALID_RECEIPT_DATE", "Tanggal penerimaan tidak boleh lebih awal dari tanggal PO atau melebihi hari ini.", map[string]string{
			"received_date": "Tanggal penerimaan tidak boleh lebih awal dari tanggal PO atau melebihi hari ini.",
		})
		return
	}

	po, err := c.service.Find(req.PurchaseOrderID)
	if err != nil || po == nil {
		errorJSON(w, http.StatusNotFound, "purchase order not found")
		return
	}
	poOrderDate := time.Date(po.CreatedAt.Year(), po.CreatedAt.Month(), po.CreatedAt.Day(), 0, 0, 0, 0, time.UTC)
	if receivedDate.Before(poOrderDate) {
		validationErrorJSON(w, "INVALID_RECEIPT_DATE", "Tanggal penerimaan tidak boleh lebih awal dari tanggal PO atau melebihi hari ini.", map[string]string{
			"received_date": "Tanggal penerimaan tidak boleh lebih awal dari tanggal PO atau melebihi hari ini.",
		})
		return
	}

	// Poin 4 (B08): Validasi kuantitas bulat integer positif > 0 dan keunikan item
	seen := make(map[string]bool, len(req.Items))
	for _, item := range req.Items {
		qty := item.QtyReceived.Int()
		if strings.TrimSpace(item.ItemID) == "" || qty <= 0 || seen[item.ItemID] {
			validationErrorJSON(w, "INVALID_INTEGER_QTY", "Kuantitas penerimaan barang harus berupa bilangan bulat positif.", map[string]string{
				"items":        "Kuantitas penerimaan barang harus berupa bilangan bulat positif.",
				"qty_received": "Kuantitas penerimaan barang harus berupa bilangan bulat positif.",
			})
			return
		}
		seen[item.ItemID] = true
	}

	// F05: Sanitasi XSS pada catatan dan identitas penerima
	req.ReceiptNo = sanitizeInput(req.ReceiptNo)
	req.Receiver = sanitizeInput(req.Receiver)
	req.Notes = sanitizeInput(req.Notes)

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

func (c *GoodsReceiptController) List(w http.ResponseWriter, r *http.Request) {
	poID := strings.TrimSpace(r.URL.Query().Get("purchase_order_id"))
	if poID == "" {
		poID = strings.TrimSpace(r.URL.Query().Get("po"))
	}
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		q = strings.TrimSpace(r.URL.Query().Get("search"))
	}

	receipts, err := c.service.ListGoodsReceipts(poID, date, from, to, q)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorJSON(w, http.StatusNotFound, "purchase order not found")
			return
		}
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]resource.GoodsReceiptResponse, 0, len(receipts))
	for _, receipt := range receipts {
		result = append(result, resource.NewGoodsReceiptResponse(receipt))
	}
	// Poin 5: Mengembalikan data object berisi goods_receipts
	jsonResponse(w, http.StatusOK, map[string]any{
		"goods_receipts": result,
	})
}

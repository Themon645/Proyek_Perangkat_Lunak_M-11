package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"m11/application"
	"m11/common"
	"m11/database"
	"m11/repository"
	"m11/routes"
)

func setupTestApp(t *testing.T) (http.Handler, *gorm.DB) {
	_ = godotenv.Load("../.env")
	db, err := database.GetPostgresDB(false)
	if err != nil {
		t.Skipf("Skipping database integration test: %v", err)
	}

	repo := repository.NewPurchaseOrderRepository(db)
	service := application.NewPurchaseOrderService(repo)
	handler := routes.Register(service)
	return handler, db
}

// Poin 1: Pelunasan ditolak jika PO belum 100% diterima (status != received)
func TestPoin1_PO_NOT_FULLY_RECEIVED(t *testing.T) {
	handler, _ := setupTestApp(t)

	// PO-2026-007 is received partial (5/10 units)
	body := map[string]any{
		"proof":   "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf",
		"paid_at": "2026-11-13",
		"bank":    "Bank Mandiri",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/purchase-orders/PO-2026-007/payment", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for payment on partial PO, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "PO_NOT_FULLY_RECEIVED") {
		t.Fatalf("expected error code PO_NOT_FULLY_RECEIVED, got: %s", w.Body.String())
	}

	// Verify GET /purchase-orders/unpaid does not return PO-2026-007
	reqUnpaid := httptest.NewRequest("GET", "/purchase-orders/unpaid", nil)
	wUnpaid := httptest.NewRecorder()
	handler.ServeHTTP(wUnpaid, reqUnpaid)

	var unpaidResp map[string]any
	_ = json.Unmarshal(wUnpaid.Body.Bytes(), &unpaidResp)
	unpaidPOs, ok := unpaidResp["purchase_orders"].([]any)
	if !ok {
		t.Fatalf("expected purchase_orders array in unpaid response")
	}
	for _, p := range unpaidPOs {
		poMap := p.(map[string]any)
		if poMap["id"] == "PO-2026-007" {
			t.Fatalf("PO-2026-007 (partial) must NOT appear in /purchase-orders/unpaid")
		}
	}
}

// B07: Pembatasan nomor surat jalan maksimal 20 karakter
func TestB07_ReceiptNoMaxLength(t *testing.T) {
	handler, _ := setupTestApp(t)

	body := map[string]any{
		"purchase_order_id": "PO-2026-001",
		"receipt_no":        "SJ-2026-EXTRA-LONG-NUMBER-EXCEEDING-20-CHARS",
		"received_date":     "2026-11-12",
		"receiver":          "Petugas Gudang",
		"items": []map[string]any{
			{"item_id": "ITEM-101", "qty_received": 10},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/goods-receipts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for receipt_no > 20 chars, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "maksimal 20 karakter") {
		t.Fatalf("expected error message about 20 characters, got: %s", w.Body.String())
	}
}

// B08 & Poin 4: Kuantitas penerimaan harus bilangan bulat, tidak boleh desimal
func TestB08_DecimalQuantityRejected(t *testing.T) {
	handler, _ := setupTestApp(t)

	bodyStr := `{
		"purchase_order_id": "PO-2026-001",
		"receipt_no": "SJ-2026-001",
		"received_date": "2026-11-12",
		"receiver": "Petugas Gudang",
		"items": [
			{"item_id": "ITEM-101", "qty_received": 2.5}
		]
	}`

	req := httptest.NewRequest("POST", "/goods-receipts", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for decimal quantity, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "INVALID_INTEGER_QTY") {
		t.Fatalf("expected error code INVALID_INTEGER_QTY, got: %s", w.Body.String())
	}
}

// B13 & Poin 3: Tanggal diterima tidak boleh di masa depan (maksimal hari ini)
func TestB13_FutureReceivedDateRejected(t *testing.T) {
	handler, _ := setupTestApp(t)

	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	body := map[string]any{
		"purchase_order_id": "PO-2026-001",
		"receipt_no":        "SJ-2026-001",
		"received_date":     futureDate,
		"receiver":          "Petugas Gudang",
		"items": []map[string]any{
			{"item_id": "ITEM-101", "qty_received": 10},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/goods-receipts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for future received date, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "INVALID_RECEIPT_DATE") {
		t.Fatalf("expected error code INVALID_RECEIPT_DATE, got: %s", w.Body.String())
	}
}

// B13 & Poin 3: Tanggal diterima tidak boleh sebelum tanggal order PO (backdated)
func TestB13_BackdatedReceivedDateRejected(t *testing.T) {
	handler, _ := setupTestApp(t)

	// PO-2026-001 order date is 2026-11-10; testing received date 2026-11-06
	body := map[string]any{
		"purchase_order_id": "PO-2026-001",
		"receipt_no":        "SJ-2026-001",
		"received_date":     "2026-11-06",
		"receiver":          "Petugas Gudang",
		"items": []map[string]any{
			{"item_id": "ITEM-101", "qty_received": 10},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/goods-receipts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for backdated received date, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "INVALID_RECEIPT_DATE") {
		t.Fatalf("expected error code INVALID_RECEIPT_DATE, got: %s", w.Body.String())
	}
}

// D08 & Poin 3: Tanggal pembayaran tidak boleh di masa depan & tidak boleh sebelum PO
func TestD08_PaymentDateValidation(t *testing.T) {
	handler, _ := setupTestApp(t)

	// Test 1: Future payment date rejected (PO-2026-003 is received)
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	bodyFuture := map[string]any{
		"proof":   "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf",
		"paid_at": futureDate,
	}
	b, _ := json.Marshal(bodyFuture)
	req := httptest.NewRequest("POST", "/purchase-orders/PO-2026-003/payment", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for future payment date, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "INVALID_PAYMENT_DATE") {
		t.Fatalf("expected error code INVALID_PAYMENT_DATE, got: %s", w.Body.String())
	}

	// Test 2: Backdated payment date before PO creation rejected
	bodyPast := map[string]any{
		"proof":   "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf",
		"paid_at": "2026-11-05",
	}
	bPast, _ := json.Marshal(bodyPast)
	reqPast := httptest.NewRequest("POST", "/purchase-orders/PO-2026-003/payment", bytes.NewReader(bPast))
	reqPast.Header.Set("Content-Type", "application/json")
	wPast := httptest.NewRecorder()
	handler.ServeHTTP(wPast, reqPast)

	if wPast.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for payment date earlier than PO, got %d: %s", wPast.Code, wPast.Body.String())
	}
	if !strings.Contains(wPast.Body.String(), "INVALID_PAYMENT_DATE") {
		t.Fatalf("expected error code INVALID_PAYMENT_DATE, got: %s", wPast.Body.String())
	}
}

// D10 & Poin 2: Bukti pelunasan menampilkan tanggal pelunasan dan tautan bukti bayar
func TestD10_ProofDetailsAndRedirect(t *testing.T) {
	handler, _ := setupTestApp(t)

	// 1. Check GET /purchase-orders/PO-2026-002 includes paid_at, tanggal_bayar, proof_url
	reqGet := httptest.NewRequest("GET", "/purchase-orders/PO-2026-002", nil)
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, reqGet)

	var res map[string]any
	_ = json.Unmarshal(wGet.Body.Bytes(), &res)
	if res["paid_at"] != "2026-11-13" || res["tanggal_bayar"] != "2026-11-13" {
		t.Fatalf("expected paid_at '2026-11-13', got %v", res["paid_at"])
	}
	if res["proof_url"] == "" || res["attachment_url"] == "" {
		t.Fatalf("expected non-empty proof_url and attachment_url")
	}

	// 2. Check GET /purchase-orders/PO-2026-002/proof redirects (302) to S3 URL
	reqProof := httptest.NewRequest("GET", "/purchase-orders/PO-2026-002/proof", nil)
	wProof := httptest.NewRecorder()
	handler.ServeHTTP(wProof, reqProof)

	if wProof.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect for proof download, got %d", wProof.Code)
	}
	target := wProof.Header().Get("Location")
	if !strings.Contains(target, "sample-receipt.pdf") {
		t.Fatalf("expected redirect target to contain sample-receipt.pdf, got: %s", target)
	}
}

// Poin 5: GET /goods-receipts mengembalikan data.goods_receipts
func TestPoin5_GlobalGoodsReceiptsResponse(t *testing.T) {
	handler, _ := setupTestApp(t)

	req := httptest.NewRequest("GET", "/goods-receipts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /goods-receipts, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Fatalf("expected success: true")
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be an object with goods_receipts key")
	}
	receipts, ok := data["goods_receipts"].([]any)
	if !ok || len(receipts) == 0 {
		t.Fatalf("expected non-empty goods_receipts array in data")
	}
}

// F05: Sanitasi XSS pada catatan kondisi barang
func TestF05_SanitizeXSS(t *testing.T) {
	handler, db := setupTestApp(t)

	// Reset database to ensure clean state
	_ = database.ResetAndSeedPostgres(db)

	body := map[string]any{
		"purchase_order_id": "PO-2026-001",
		"receipt_no":        "SJ-2026-XSS",
		"received_date":     "2026-11-12",
		"receiver":          "Petugas Gudang",
		"notes":             "<script>alert(1)</script>Barang dalam kondisi baik",
		"items": []map[string]any{
			{"item_id": "ITEM-101", "qty_received": 10},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/goods-receipts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid GRN, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "<script>") {
		t.Fatalf("notes should have script tags stripped, got: %s", w.Body.String())
	}

	// Reset back to original seed
	_ = database.ResetAndSeedPostgres(db)
}

// Test PO Status variety (matches QA Checklist)
func TestPOStatusVariety(t *testing.T) {
	handler, _ := setupTestApp(t)

	req := httptest.NewRequest("GET", "/purchase-orders", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var list []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 20 {
		t.Fatalf("expected exactly 20 POs, got %d", len(list))
	}

	statusCounts := map[string]int{}
	for _, po := range list {
		statusCounts[po["status"].(string)]++
	}

	if statusCounts[common.StatusApproved] != 7 {
		t.Fatalf("expected 7 approved, got %d", statusCounts[common.StatusApproved])
	}
	if statusCounts[common.StatusReceived] != 7 {
		t.Fatalf("expected 7 received, got %d", statusCounts[common.StatusReceived])
	}
	if statusCounts[common.StatusReceivedPartial] != 3 {
		t.Fatalf("expected 3 received partial, got %d", statusCounts[common.StatusReceivedPartial])
	}
	if statusCounts[common.StatusPending] != 3 {
		t.Fatalf("expected 3 pending (diajukan), got %d", statusCounts[common.StatusPending])
	}

	// Test filter selain disetujui (?status=selain-disetujui atau ?exclude_status=approved)
	for _, queryParam := range []string{"?status=selain-disetujui", "?exclude_status=approved", "?status=!approved"} {
		reqFilter := httptest.NewRequest("GET", "/purchase-orders"+queryParam, nil)
		wFilter := httptest.NewRecorder()
		handler.ServeHTTP(wFilter, reqFilter)

		if wFilter.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", queryParam, wFilter.Code)
		}
		var nonApprovedList []map[string]any
		_ = json.Unmarshal(wFilter.Body.Bytes(), &nonApprovedList)
		if len(nonApprovedList) != 13 {
			t.Fatalf("expected 13 non-approved POs for %s, got %d", queryParam, len(nonApprovedList))
		}
		for _, item := range nonApprovedList {
			if item["status"] == common.StatusApproved {
				t.Fatalf("expected no approved status in non-approved filter for %s", queryParam)
			}
		}
	}
}

// Test multi-item seeder (1 PO memiliki lebih dari 1 barang)
func TestPOMultiItemSeeder(t *testing.T) {
	handler, _ := setupTestApp(t)

	// Check PO-2026-001 (has 3 items)
	req := httptest.NewRequest("GET", "/purchase-orders/PO-2026-001", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var po map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &po)
	items, ok := po["items"].([]any)
	if !ok || len(items) < 2 {
		t.Fatalf("expected PO-2026-001 to have more than 1 item, got %d", len(items))
	}
	if len(items) != 3 {
		t.Fatalf("expected PO-2026-001 to have 3 items, got %d", len(items))
	}

	// Check PO-2026-007 (has 2 items with partial receipt)
	req2 := httptest.NewRequest("GET", "/purchase-orders/PO-2026-007", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var po7 map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &po7)
	items7, ok := po7["items"].([]any)
	if !ok || len(items7) != 2 {
		t.Fatalf("expected PO-2026-007 to have 2 items, got %d", len(items7))
	}
}


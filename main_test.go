package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"m11/api"
	"m11/application"
	"m11/common"
	"m11/database"
	"m11/model"
	"m11/repository"
)

func TestM11Flow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required for PostgreSQL integration tests")
	}
	database, err := database.Register(dsn, true)
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewPurchaseOrderService(repository.NewPurchaseOrderRepository(database))
	handler := api.Register(service)
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}

	res := request(http.MethodPost, "/purchase-orders", map[string]any{
		"id": "PO-1", "supplier": "Supplier A", "status": "approved",
		"items": []map[string]any{{"id": "item-1", "name": "Laptop", "quantity": 50, "unit_cost": 100}},
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create PO: got %d", res.Code)
	}
	res = request(http.MethodPost, "/purchase-orders/PO-1/receipts", map[string]any{"items": []map[string]any{{"id": "item-1", "quantity": 30}}})
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte(`"received partial"`)) {
		t.Fatalf("partial receipt: got %d %s", res.Code, res.Body.String())
	}
	res = request(http.MethodPost, "/purchase-orders/PO-1/receipts", map[string]any{"items": []map[string]any{{"id": "item-1", "quantity": 21}}})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("over-receipt: got %d", res.Code)
	}
	res = request(http.MethodPost, "/purchase-orders/PO-1/payment", map[string]string{"proof": "transfer-1"})
	if res.Code != http.StatusOK {
		t.Fatalf("payment: got %d", res.Code)
	}

	res = request(http.MethodPost, "/purchase-orders", map[string]any{
		"id": "PO-2", "supplier": "Supplier B", "status": "approved",
		"items": []map[string]any{{"id": "item-2", "name": "Monitor", "quantity": 10, "unit_cost": 200}},
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create PO for GRN: got %d", res.Code)
	}
	res = request(http.MethodPost, "/goods-receipts", map[string]any{
		"purchase_order_id": "PO-2", "receipt_no": "SJ-2", "received_date": "2026-10-27",
		"items": []map[string]any{{"item_id": "item-2", "qty_received": 0}},
	})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("invalid GRN: got %d", res.Code)
	}
	res = request(http.MethodPost, "/goods-receipts", map[string]any{
		"purchase_order_id": "PO-2", "receipt_no": "SJ-2", "received_date": "2026-10-27",
		"items": []map[string]any{{"item_id": "item-2", "qty_received": 10}},
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create GRN: got %d %s", res.Code, res.Body.String())
	}
	var po model.PurchaseOrder
	if err := database.First(&po, "id = ?", "PO-2").Error; err != nil || po.Status != common.StatusReceived {
		t.Fatalf("GRN status: got %q, err %v", po.Status, err)
	}
}

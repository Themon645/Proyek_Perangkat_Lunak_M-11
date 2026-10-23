package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"m11/api"
	"m11/application"
	"m11/database"
	"m11/repository"
)

func TestM11Flow(t *testing.T) {
	database, err := database.Register("file::memory:?cache=shared")
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
}

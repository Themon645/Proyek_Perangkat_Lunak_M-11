package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}
}

func TestHandlerCORSPreflight(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost:5173")
	req := httptest.NewRequest(http.MethodOptions, "/purchase-orders", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("unexpected origin header: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}


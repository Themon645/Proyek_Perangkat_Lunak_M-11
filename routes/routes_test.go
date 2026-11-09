package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAllowsFrontendPreflight(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost:5173")
	req := httptest.NewRequest(http.MethodOptions, "/purchase-orders", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	Register(nil).ServeHTTP(res, req)

	if res.Code != http.StatusNoContent ||
		res.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("preflight: got status %d and origin %q", res.Code, res.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRegisterHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	Register(nil).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.Code)
	}
	if res.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %q", res.Header().Get("Content-Type"))
	}
}

func TestRegisterPathNormalization(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	res := httptest.NewRecorder()

	Register(nil).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/health, got %d", res.Code)
	}
}


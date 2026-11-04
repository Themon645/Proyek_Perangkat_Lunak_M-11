package api

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

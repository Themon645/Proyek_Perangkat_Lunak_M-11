package api

import (
	"net/http"
	"os"
	"strings"

	"m11/api/routes"
	"m11/application"
	"m11/common/helper"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func Register(service *application.PurchaseOrderService) http.Handler {
	route := http.NewServeMux()
	routes.RegisterPurchaseOrderRoutes(route, service)
	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions && strings.EqualFold(r.Header.Get("Origin"), allowedOrigin) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		helper.LogIncomingRequest(r)

		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		route.ServeHTTP(rec, r)

		if rec.statusCode >= 400 {
			helper.LogHTTPError(r, rec.statusCode)
		}
	})
}

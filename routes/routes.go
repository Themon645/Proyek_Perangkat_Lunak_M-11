package routes

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"m11/application"
	"m11/common/helper"
	"m11/database"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Register mengonfigurasi multiplexer routing dan middleware CORS untuk Vercel Serverless Function.
func Register(service *application.PurchaseOrderService) http.Handler {
	mux := http.NewServeMux()
	RegisterPurchaseOrderRoutes(mux, service)

	// Favicon handler agar browser tidak memicu error 404/500 saat meminta favicon
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Root endpoint: mencegah error saat preview Vercel membuka root /
	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "M11 Procurement API Serverless on Vercel is online",
			"endpoints": []string{
				"/health",
				"/purchase-orders",
				"/purchase-orders/unpaid",
				"/reports/procurement",
				"/goods-receipts",
			},
		})
	}
	mux.HandleFunc("GET /{$}", rootHandler)
	mux.HandleFunc("GET /index.go", rootHandler)
	mux.HandleFunc("GET /api/index.go", rootHandler)

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		dbStatus := map[string]string{
			"serverless": "running",
		}

		if os.Getenv("MONGODB_URI") != "" {
			if _, err := database.GetMongoClient(); err != nil {
				dbStatus["mongodb"] = "error: " + err.Error()
			} else {
				dbStatus["mongodb"] = "connected"
			}
		} else {
			dbStatus["mongodb"] = "not_configured (MONGODB_URI is empty in Vercel)"
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    dbStatus,
		})
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. CORS Configuration
		origin := r.Header.Get("Origin")
		allowedOrigin := os.Getenv("CORS_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "*"
		}

		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin == allowedOrigin || strings.Contains(allowedOrigin, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept")

		// 2. Preflight OPTIONS Request Handling
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 3. Normalisasi Path Vercel Rewrite
		originalPath := r.URL.Path
		realPath := r.URL.Path

		if matched := r.Header.Get("x-matched-path"); matched != "" {
			realPath = matched
		} else if u, err := url.Parse(r.RequestURI); err == nil && u.Path != "" && u.Path != "/api/index.go" {
			realPath = u.Path
		}

		if strings.HasPrefix(realPath, "/api/") {
			realPath = strings.TrimPrefix(realPath, "/api")
		} else if realPath == "/api" || realPath == "/api/index.go" || realPath == "/index.go" {
			realPath = "/"
		}

		r.URL.Path = realPath

		helper.LogIncomingRequest(r)

		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		mux.ServeHTTP(rec, r)

		if rec.statusCode >= 400 {
			helper.LogHTTPError(r, rec.statusCode)
		}

		// Restore path
		r.URL.Path = originalPath
	})
}

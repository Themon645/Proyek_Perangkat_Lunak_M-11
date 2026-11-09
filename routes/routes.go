package routes

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

	// Favicon handler
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Root endpoint: sambutan & info endpoint
	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "M11 Procurement API Serverless on Vercel is online",
			"database": "PostgreSQL (Neon)",
			"endpoints": []string{
				"/health",
				"/purchase-orders",
				"/purchase-orders/approved",
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

		if os.Getenv("DATABASE_URL") != "" || os.Getenv("DB_HOST") != "" {
			if db, err := database.GetPostgresDB(false); err != nil {
				dbStatus["postgres"] = "error: " + err.Error()
			} else {
				sqlDB, err := db.DB()
				if err != nil || sqlDB.Ping() != nil {
					dbStatus["postgres"] = "ping_failed"
				} else {
					dbStatus["postgres"] = "connected (Neon)"
				}
			}
		} else {
			dbStatus["postgres"] = "not_configured (DATABASE_URL is empty)"
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    dbStatus,
		})
	})

	// Endpoint untuk membersihkan dan me-reset data database
	mux.HandleFunc("POST /database/reset-and-seed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "DATABASE_URL is not configured"})
			return
		}

		db, err := database.GetPostgresDB(true)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
			return
		}
		if err := database.ResetAndSeedPostgres(db); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "PostgreSQL (Neon) berhasil dibersihkan dan diisi 10 purchase orders & goods receipts bersih",
		})
	})

	// Static file handler untuk bukti pembayaran (storage lokal atau fallback /tmp di serverless)
	mux.HandleFunc("GET /storage/payment-proofs/{filename}", func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.PathValue("filename"))
		primary := filepath.Join("storage", "payment-proofs", filename)
		if _, err := os.Stat(primary); err == nil {
			http.ServeFile(w, r, primary)
			return
		}
		fallback := filepath.Join(os.TempDir(), "storage", "payment-proofs", filename)
		if _, err := os.Stat(fallback); err == nil {
			http.ServeFile(w, r, fallback)
			return
		}
		fallbackTemp := filepath.Join(os.TempDir(), filename)
		if _, err := os.Stat(fallbackTemp); err == nil {
			http.ServeFile(w, r, fallbackTemp)
			return
		}
		http.NotFound(w, r)
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

package routes

import (
	"encoding/json"
	"net/http"
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

	// Health check endpoint untuk memverifikasi deployment serverless & koneksi database
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
			allowedOrigin = "http://localhost:5173"
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

		// 3. Normalisasi Path: dukung pemanggilan via /api/* maupun direct route /*
		originalPath := r.URL.Path
		if strings.HasPrefix(r.URL.Path, "/api/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		} else if r.URL.Path == "/api" {
			r.URL.Path = "/"
		}

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


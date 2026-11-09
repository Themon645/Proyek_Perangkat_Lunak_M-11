package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"m11/application"
	"m11/common/helper"
	"m11/database"
	"m11/integration"
	"m11/repository"
	"m11/routes"
)

var (
	appHandler http.Handler
	initOnce   sync.Once
)

// initApp menginisialisasi database singleton, repository, service, dan router.
func initApp() {
	_ = godotenv.Load()

	// 1. Inisialisasi koneksi pool singleton MongoDB jika MONGODB_URI dikonfigurasi
	if os.Getenv("MONGODB_URI") != "" {
		if _, err := database.GetMongoClient(); err != nil {
			helper.LogError("MongoDB init error: %v", err)
		} else {
			helper.LogInfo("MongoDB connected successfully")
		}
	}

	// 2. Inisialisasi PostgreSQL hanya jika remote host dikonfigurasi (bukan localhost di Vercel)
	var service *application.PurchaseOrderService
	dbHost := os.Getenv("DB_HOST")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" || (dbHost != "" && dbHost != "localhost" && dbHost != "127.0.0.1") {
		autoMigrate, _ := strconv.ParseBool(os.Getenv("DB_AUTO_MIGRATE"))
		db, err := database.GetPostgresDB(autoMigrate)
		if err != nil {
			helper.LogError("PostgreSQL init error: %v", err)
		} else {
			repo := repository.NewPurchaseOrderRepository(db)

			var publisher integration.GRNPublisher
			if endpoint := os.Getenv("M9_GRN_ENDPOINT"); endpoint != "" {
				m9Queue := integration.NewM9Queue(endpoint, nil)
				m9Queue.Start()
				publisher = m9Queue
			}

			service = application.NewPurchaseOrderService(repo, publisher)
		}
	}

	appHandler = routes.Register(service)
}

// Handler adalah entrypoint standar Vercel Serverless Function.
func Handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			helper.LogError("panic recovered in Handler: %v", rec)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "INTERNAL_SERVER_ERROR",
					"message": fmt.Sprintf("%v", rec),
				},
			})
		}
	}()

	initOnce.Do(initApp)

	if appHandler == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "online",
			"message": "Handler initialized without active service",
		})
		return
	}

	appHandler.ServeHTTP(w, r)
}

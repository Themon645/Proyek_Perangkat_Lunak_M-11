package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"m11/application"
	"m11/database"
	"m11/integration"
	"m11/repository"
	"m11/routes"
)

var (
	appHandler http.Handler
	initOnce   sync.Once
	initErr    error
)

// initApp menginisialisasi database singleton, repository, service, dan router sekali saja
// selama siklus hidup (lifecycle) instance container serverless Vercel (warm start reuse).
func initApp() {
	_ = godotenv.Load()

	// 1. Inisialisasi koneksi pool singleton MongoDB jika MONGODB_URI dikonfigurasi
	if os.Getenv("MONGODB_URI") != "" {
		if _, err := database.GetMongoClient(); err != nil {
			initErr = err
			return
		}
	}

	// 2. Inisialisasi koneksi PostgreSQL singleton jika kredensial relational DB disediakan
	var service *application.PurchaseOrderService
	if os.Getenv("DB_HOST") != "" || os.Getenv("DATABASE_URL") != "" {
		autoMigrate, _ := strconv.ParseBool(os.Getenv("DB_AUTO_MIGRATE"))
		db, err := database.GetPostgresDB(autoMigrate)
		if err != nil {
			initErr = err
			return
		}

		repo := repository.NewPurchaseOrderRepository(db)

		var publisher integration.GRNPublisher
		if endpoint := os.Getenv("M9_GRN_ENDPOINT"); endpoint != "" {
			m9Queue := integration.NewM9Queue(endpoint, nil)
			m9Queue.Start()
			publisher = m9Queue
		}

		service = application.NewPurchaseOrderService(repo, publisher)
	}

	appHandler = routes.Register(service)
}

// Handler adalah entrypoint standar Vercel Serverless Function.
// Signature: func(w http.ResponseWriter, r *http.Request) dengan package handler.
// Tidak ada pemanggilan http.ListenAndServe atau server loop permanen.
func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(initApp)

	if initErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error": map[string]string{
				"code":    "INITIALIZATION_ERROR",
				"message": initErr.Error(),
			},
		})
		return
	}

	appHandler.ServeHTTP(w, r)
}


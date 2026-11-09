package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"m11/application"
	"m11/common/helper"
	"m11/database"
	"m11/integration"
	"m11/repository"
	"m11/routes"
)

func main() {
	_ = godotenv.Load()

	// Vercel memberikan environment variable PORT saat menjalankan server
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}

	var repo *repository.PurchaseOrderRepository

	// Inisialisasi PostgreSQL (Neon) jika DATABASE_URL atau DB_HOST diset
	dbURL := os.Getenv("DATABASE_URL")
	dbHost := os.Getenv("DB_HOST")
	if dbURL != "" || dbHost != "" {
		db, err := database.GetPostgresDB(true)
		if err != nil {
			helper.LogError("PostgreSQL (Neon) init error: %v", err)
		} else {
			_ = database.SeedDummy(db)
			repo = repository.NewPurchaseOrderRepository(db)
			helper.LogInfo("PostgreSQL (Neon) repository connected and active")
		}
	}

	var service *application.PurchaseOrderService
	if repo != nil {
		var publisher integration.GRNPublisher
		if endpoint := os.Getenv("M9_GRN_ENDPOINT"); endpoint != "" {
			m9Queue := integration.NewM9Queue(endpoint, nil)
			m9Queue.Start()
			publisher = m9Queue
		}
		service = application.NewPurchaseOrderService(repo, publisher)
	}

	handler := routes.Register(service)

	helper.LogInfo("Server running on port :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

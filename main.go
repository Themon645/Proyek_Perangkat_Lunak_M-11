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

	// 1. Inisialisasi MongoDB jika MONGODB_URI dikonfigurasi
	if os.Getenv("MONGODB_URI") != "" {
		if mongoDB, err := database.GetMongoDatabase(); err != nil {
			helper.LogError("MongoDB init error: %v", err)
		} else {
			_ = database.SeedMongo(mongoDB)
			repo = repository.NewMongoPurchaseOrderRepository(mongoDB)
			helper.LogInfo("MongoDB repository connected and active")
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

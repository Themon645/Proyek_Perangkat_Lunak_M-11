package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"m11/application"
	"m11/common/helper"
	"m11/database"
	"m11/integration"
	"m11/repository"
	"m11/routes"
)

// Main entrypoint khusus untuk development server standalone lokal.
// Pada deployment produksi ke Vercel, entrypoint yang digunakan adalah api/index.go (Serverless Function).
func main() {
	_ = godotenv.Load()
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	autoMigrate, _ := strconv.ParseBool(os.Getenv("DB_AUTO_MIGRATE"))
	db, err := database.GetPostgresDB(autoMigrate)
	if err != nil {
		log.Fatal(err)
	}

	seedDummy, _ := strconv.ParseBool(os.Getenv("DB_SEED_DUMMY"))
	if seedDummy {
		if err := database.SeedDummy(db); err != nil {
			log.Fatal(err)
		}
		helper.LogInfo("dummy data ready: 10 purchase orders, 10 items, 10 goods receipts")
	}

	repo := repository.NewPurchaseOrderRepository(db)
	var publisher integration.GRNPublisher
	if endpoint := os.Getenv("M9_GRN_ENDPOINT"); endpoint != "" {
		m9Queue := integration.NewM9Queue(endpoint, nil)
		m9Queue.Start()
		publisher = m9Queue
	}

	service := application.NewPurchaseOrderService(repo, publisher)
	address := host + ":" + port
	helper.LogInfo("backend running locally at http://%s:%s", host, port)
	log.Fatal(http.ListenAndServe(address, routes.Register(service)))
}


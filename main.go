package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"m11/api"
	"m11/application"
	"m11/common/helper"
	"m11/database"
	"m11/integration"
	"m11/repository"
)

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

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	autoMigrate, err := strconv.ParseBool(os.Getenv("DB_AUTO_MIGRATE"))
	if os.Getenv("DB_AUTO_MIGRATE") == "" {
		autoMigrate = false
	} else if err != nil {
		log.Fatal("DB_AUTO_MIGRATE must be true or false")
	}
	dsn := "host=" + os.Getenv("DB_HOST") +
		" port=" + dbPort +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" sslmode=" + sslMode
	db, err := database.Register(dsn, autoMigrate)
	if err != nil {
		log.Fatal(err)
	}
	seedDummy, err := strconv.ParseBool(os.Getenv("DB_SEED_DUMMY"))
	if os.Getenv("DB_SEED_DUMMY") == "" {
		seedDummy = false
	} else if err != nil {
		log.Fatal("DB_SEED_DUMMY must be true or false")
	}
	if seedDummy {
		if err := database.SeedDummy(db); err != nil {
			log.Fatal(err)
		}
		helper.LogInfo("dummy data ready: 10 purchase orders, 10 items, 10 goods receipts")
	}

	repo := repository.NewPurchaseOrderRepository(db)
	m9Queue := integration.NewM9Queue(os.Getenv("M9_GRN_ENDPOINT"), nil)
	m9Queue.Start()
	service := application.NewPurchaseOrderService(repo, m9Queue)
	address := host + ":" + port
	displayHost := host
	if host == "0.0.0.0" {
		displayHost = "localhost"
	}
	helper.LogInfo("database connected: %s", os.Getenv("DB_NAME"))
	helper.LogInfo("backend running at http://%s:%s", displayHost, port)
	helper.LogInfo("auto migration: %t", autoMigrate)
	log.Fatal(http.ListenAndServe(address, api.Register(service)))
}

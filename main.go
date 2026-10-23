package main

import (
	"log"
	"net/http"

	"m11/api"
	"m11/application"
	"m11/database"
	"m11/repository"
)

func main() {
	database, err := database.Register("m11.db")
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPurchaseOrderRepository(database)
	service := application.NewPurchaseOrderService(repo)
	log.Fatal(http.ListenAndServe(":8080", api.Register(service)))
}

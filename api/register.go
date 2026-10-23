package api

import (
	"net/http"

	"m11/api/routes"
	"m11/application"
)

func Register(service *application.PurchaseOrderService) http.Handler {
	mux := http.NewServeMux()
	routes.RegisterPurchaseOrderRoutes(mux, service)
	return mux
}

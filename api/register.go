package api

import (
	"net/http"

	"m11/api/routes"
	"m11/application"
)

func Register(service *application.PurchaseOrderService) http.Handler {
	route := http.NewServeMux()
	routes.RegisterPurchaseOrderRoutes(route, service)
	return route
}

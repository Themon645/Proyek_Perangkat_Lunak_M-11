package routes

import (
	"net/http"

	"m11/application"
	"m11/controller"
)

func RegisterPurchaseOrderRoutes(mux *http.ServeMux, service *application.PurchaseOrderService) {
	handler := controller.NewPurchaseOrderController(service)
	mux.Handle("/purchase-orders", handler)
	mux.Handle("/purchase-orders/", handler)
	mux.Handle("/reports/procurement", handler)
}

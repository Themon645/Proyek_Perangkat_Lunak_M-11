package routes

import (
	"net/http"

	"m11/application"
	"m11/controller"
)

func RegisterPurchaseOrderRoutes(route *http.ServeMux, service *application.PurchaseOrderService) {
	handler := controller.NewPurchaseOrderController(service)
	receiptHandler := controller.NewGoodsReceiptController(service)
	route.Handle("/purchase-orders", handler)
	route.Handle("/purchase-orders/", handler)
	route.Handle("/reports/procurement", handler)
	route.Handle("/goods-receipts", receiptHandler)
}

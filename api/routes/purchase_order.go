package routes

import (
	"net/http"

	"m11/application"
	"m11/controller"
)

func RegisterPurchaseOrderRoutes(route *http.ServeMux, service *application.PurchaseOrderService) {
	handler := controller.NewPurchaseOrderController(service)
	receiptHandler := controller.NewGoodsReceiptController(service)
	route.HandleFunc("GET /purchase-orders", handler.List)
	route.HandleFunc("POST /purchase-orders", handler.Create)
	route.HandleFunc("GET /purchase-orders/unpaid", handler.Unpaid)
	route.HandleFunc("GET /purchase-orders/{id}", handler.Find)
	route.HandleFunc("GET /purchase-orders/{id}/receipts", handler.ReceiptHistory)
	route.HandleFunc("POST /purchase-orders/{id}/receipts", handler.Receive)
	route.HandleFunc("POST /purchase-orders/{id}/payment", handler.Pay)
	route.HandleFunc("GET /reports/procurement", handler.Report)
	route.HandleFunc("POST /goods-receipts", receiptHandler.Create)
}

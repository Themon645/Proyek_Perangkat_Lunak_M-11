package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"m11/application"
	"m11/common"
	"m11/dto"
	"m11/model"
)

type PurchaseOrderController struct {
	service *application.PurchaseOrderService
}

func NewPurchaseOrderController(service *application.PurchaseOrderService) http.Handler {
	c := &PurchaseOrderController{service: service}
	return http.HandlerFunc(c.handle)
}

func (c *PurchaseOrderController) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/purchase-orders" {
		c.create(w, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/purchase-orders/unpaid" {
		c.unpaid(w)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/reports/procurement" {
		c.report(w, r)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "purchase-orders" && r.Method == http.MethodPost {
		if parts[2] == "receipts" {
			c.receive(w, r, parts[1])
			return
		}
		if parts[2] == "payment" {
			c.pay(w, r, parts[1])
			return
		}
	}
	errorJSON(w, http.StatusNotFound, "route not found")
}

func (c *PurchaseOrderController) create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePurchaseOrderRequest
	if !decodeJSON(w, r, &req) || req.Supplier == "" || len(req.Items) == 0 {
		return
	}
	if req.ID == "" {
		req.ID = "PO-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	if req.Status == "" {
		req.Status = common.StatusPending
	}
	for i := range req.Items {
		if req.Items[i].ID == "" || req.Items[i].Quantity <= 0 || req.Items[i].UnitCost < 0 {
			errorJSON(w, http.StatusBadRequest, "invalid purchase order item")
			return
		}
	}
	items := make([]model.PurchaseOrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = model.PurchaseOrderItem{
			ID: item.ID, PurchaseOrderID: req.ID, Name: item.Name,
			Quantity: item.Quantity, UnitCost: item.UnitCost,
		}
	}
	po := &model.PurchaseOrder{ID: req.ID, Supplier: req.Supplier, Status: req.Status, Payment: common.PaymentUnpaid, Items: items}
	if err := c.service.Create(po); err != nil {
		errorJSON(w, http.StatusConflict, "purchase order already exists")
		return
	}
	jsonResponse(w, http.StatusCreated, po)
}

func (c *PurchaseOrderController) receive(w http.ResponseWriter, r *http.Request, id string) {
	var req dto.ReceivePurchaseOrderRequest
	if !decodeJSON(w, r, &req) || len(req.Items) == 0 {
		return
	}
	received := make(map[string]int, len(req.Items))
	for _, item := range req.Items {
		received[item.ID] += item.Quantity
	}
	po, err := c.service.Receive(id, received)
	if err != nil {
		c.repositoryError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, po)
}

func (c *PurchaseOrderController) pay(w http.ResponseWriter, r *http.Request, id string) {
	var req dto.PayPurchaseOrderRequest
	if !decodeJSON(w, r, &req) || strings.TrimSpace(req.Proof) == "" {
		return
	}
	po, err := c.service.Pay(id, req.Proof)
	if err != nil {
		c.repositoryError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, po)
}

func (c *PurchaseOrderController) unpaid(w http.ResponseWriter) {
	rows, total, err := c.service.Unpaid()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"purchase_orders": rows, "total": total})
}

func (c *PurchaseOrderController) report(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "from must use YYYY-MM-DD")
		return
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "to must use YYYY-MM-DD")
		return
	}
	totals, err := c.service.Report(from, to)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"), "suppliers": totals})
}

func (c *PurchaseOrderController) repositoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		errorJSON(w, http.StatusNotFound, "purchase order not found")
	case strings.Contains(err.Error(), "must be approved"), strings.Contains(err.Error(), "must be received"):
		errorJSON(w, http.StatusConflict, err.Error())
	default:
		errorJSON(w, http.StatusBadRequest, err.Error())
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		errorJSON(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func errorJSON(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

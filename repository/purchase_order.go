package repository

import (
	"errors"
	"strings"
	"time"

	"m11/common"
	"m11/common/validation"
	"m11/dto"
	"m11/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = gorm.ErrRecordNotFound

type PurchaseOrderRepository struct {
	db *gorm.DB
}

func NewPurchaseOrderRepository(db *gorm.DB) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{db: db}
}

func (r *PurchaseOrderRepository) Create(po *model.PurchaseOrder) error {
	return r.db.Create(po).Error
}

func (r *PurchaseOrderRepository) Find(id string) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	if err := r.db.Preload("Items").Preload("Payments").First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *PurchaseOrderRepository) List() ([]model.PurchaseOrder, error) {
	var orders []model.PurchaseOrder
	if err := r.db.Preload("Items").Preload("Payments").Order("created_at ASC").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PurchaseOrderRepository) ListFiltered(status, date, from, to, q string) ([]model.PurchaseOrder, error) {
	var orders []model.PurchaseOrder
	query := r.db.Preload("Items").Preload("Payments")

	if status != "" && status != "semua" {
		query = query.Where("status = ?", status)
	}
	if date != "" {
		if parsedDate, err := time.Parse("2006-01-02", date); err == nil {
			query = query.Where("created_at >= ? AND created_at < ?", parsedDate, parsedDate.Add(24*time.Hour))
		}
	}
	if from != "" {
		if parsedFrom, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("created_at >= ?", parsedFrom)
		}
	}
	if to != "" {
		if parsedTo, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("created_at < ?", parsedTo.Add(24*time.Hour))
		}
	}
	if q != "" {
		like := "%" + strings.TrimSpace(q) + "%"
		query = query.Where("id ILIKE ? OR supplier ILIKE ?", like, like)
	}

	if err := query.Order("created_at ASC").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PurchaseOrderRepository) Approved() ([]model.PurchaseOrder, error) {
	var orders []model.PurchaseOrder
	if err := r.db.Preload("Items").Preload("Payments").
		Where("status IN ?", []string{common.StatusApproved, common.StatusReceivedPartial}).
		Order("created_at ASC").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PurchaseOrderRepository) ListByStatus(status string) ([]model.PurchaseOrder, error) {
	var orders []model.PurchaseOrder
	if err := r.db.Preload("Items").Preload("Payments").Where("status = ?", status).Order("created_at ASC").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PurchaseOrderRepository) ListGoodsReceipts(poID string, filters ...string) ([]model.GoodsReceipt, error) {
	var receipts []model.GoodsReceipt
	query := r.db.Preload("Items.Item").Preload("PurchaseOrder")
	if poID != "" {
		var po model.PurchaseOrder
		if err := r.db.First(&po, "id = ?", poID).Error; err != nil {
			return nil, err
		}
		query = query.Where("purchase_order_uuid = ?", po.UUID)
	}
	// filters optional: [date, from, to, q]
	if len(filters) > 0 && filters[0] != "" {
		if parsedDate, err := time.Parse("2006-01-02", filters[0]); err == nil {
			query = query.Where("received_date >= ? AND received_date < ?", parsedDate, parsedDate.Add(24*time.Hour))
		}
	}
	if len(filters) > 1 && filters[1] != "" {
		if parsedFrom, err := time.Parse("2006-01-02", filters[1]); err == nil {
			query = query.Where("received_date >= ?", parsedFrom)
		}
	}
	if len(filters) > 2 && filters[2] != "" {
		if parsedTo, err := time.Parse("2006-01-02", filters[2]); err == nil {
			query = query.Where("received_date < ?", parsedTo.Add(24*time.Hour))
		}
	}
	if len(filters) > 3 && filters[3] != "" {
		like := "%" + strings.TrimSpace(filters[3]) + "%"
		query = query.Where("receipt_no ILIKE ? OR receiver ILIKE ? OR notes ILIKE ?", like, like, like)
	}

	if err := query.Order("received_date DESC, created_at DESC").Find(&receipts).Error; err != nil {
		return nil, err
	}
	for i := range receipts {
		if receipts[i].PurchaseOrder.ID != "" {
			receipts[i].PurchaseOrderID = receipts[i].PurchaseOrder.ID
		}
	}
	return receipts, nil
}

func (r *PurchaseOrderRepository) ReceiptHistory(id string) (*model.PurchaseOrder, []model.GoodsReceipt, error) {
	var po model.PurchaseOrder
	if err := r.db.First(&po, "id = ?", id).Error; err != nil {
		return nil, nil, err
	}
	var receipts []model.GoodsReceipt
	if err := r.db.Preload("Items.Item").
		Where("purchase_order_uuid = ?", po.UUID).
		Order("received_date ASC, created_at ASC").
		Find(&receipts).Error; err != nil {
		return nil, nil, err
	}
	return &po, receipts, nil
}

func (r *PurchaseOrderRepository) Receive(id string, received map[string]int) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
			return err
		}
		if po.Status != common.StatusApproved && po.Status != common.StatusReceivedPartial {
			return errors.New("purchase order must be approved")
		}
		items := make(map[string]*model.PurchaseOrderItem, len(po.Items))
		for i := range po.Items {
			items[po.Items[i].ID] = &po.Items[i]
		}
		for itemID, quantity := range received {
			item, ok := items[itemID]
			if !ok || quantity <= 0 {
				return errors.New("received quantity exceeds ordered quantity")
			}
			if err := validation.ValidatePartialQuantity(item.Quantity, item.Received, quantity); err != nil {
				return err
			}
		}
		for itemID, quantity := range received {
			items[itemID].Received += quantity
			if err := tx.Model(&model.PurchaseOrderItem{}).Where("uuid = ?", items[itemID].UUID).
				Update("received", items[itemID].Received).Error; err != nil {
				return err
			}
		}
		complete := true
		for _, item := range po.Items {
			if item.Received < item.Quantity {
				complete = false
				break
			}
		}
		po.Status = common.StatusReceivedPartial
		if complete {
			po.Status = common.StatusReceived
		}
		now := time.Now().UTC()
		po.ReceivedAt = &now
		return tx.Model(&po).Updates(map[string]any{"status": po.Status, "received_at": po.ReceivedAt}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.Find(id)
}

func (r *PurchaseOrderRepository) CreateGoodsReceipt(req *dto.CreateGoodsReceiptRequest, receivedDate time.Time) (*model.GoodsReceipt, error) {
	var receipt model.GoodsReceipt
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var po model.PurchaseOrder
		if err := tx.Preload("Items").First(&po, "id = ?", req.PurchaseOrderID).Error; err != nil {
			return err
		}
		if po.Status != common.StatusApproved && po.Status != common.StatusReceivedPartial {
			return errors.New("purchase order must be approved")
		}

		items := make(map[string]*model.PurchaseOrderItem, len(po.Items))
		for i := range po.Items {
			items[po.Items[i].ID] = &po.Items[i]
		}
		totals, err := receiptTotals(tx, po.Items)
		if err != nil {
			return err
		}
		receipt = model.GoodsReceipt{
			ReceiptNo:         req.ReceiptNo,
			PurchaseOrderUUID: po.UUID,
			ReceivedDate:      receivedDate,
			Receiver:          req.Receiver,
			Notes:             req.Notes,
			Items:             make([]model.GoodsReceiptItem, 0, len(req.Items)),
		}
		if err := tx.Create(&receipt).Error; err != nil {
			return err
		}
		for _, item := range req.Items {
			orderItem, ok := items[item.ItemID]
			if !ok {
				return errors.New("received quantity exceeds ordered quantity")
			}
			qty := item.QtyReceived.Int()
			if err := validation.ValidatePartialQuantity(orderItem.Quantity, totals[orderItem.UUID], qty); err != nil {
				return err
			}
			if err := tx.Create(&model.GoodsReceiptItem{
				ReceiptUUID: receipt.UUID, ItemUUID: orderItem.UUID, QtyReceived: qty,
			}).Error; err != nil {
				return err
			}
			receipt.Items = append(receipt.Items, model.GoodsReceiptItem{
				ReceiptUUID: receipt.UUID, ItemUUID: orderItem.UUID, QtyReceived: qty,
			})
		}
		totals, err = receiptTotals(tx, po.Items)
		if err != nil {
			return err
		}
		complete := true
		for _, item := range items {
			item.Received = totals[item.UUID]
			if err := tx.Model(&model.PurchaseOrderItem{}).Where("uuid = ?", item.UUID).
				Update("received", item.Received).Error; err != nil {
				return err
			}
			if item.Received < item.Quantity {
				complete = false
			}
		}
		status := common.StatusReceivedPartial
		if complete {
			status = common.StatusReceived
		}
		now := time.Now().UTC()
		return tx.Model(&po).Updates(map[string]any{"status": status, "received_at": &now}).Error
	})
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

type receiptTotal struct {
	ItemUUID string
	Total    int
}

func receiptTotals(tx *gorm.DB, items []model.PurchaseOrderItem) (map[uuid.UUID]int, error) {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.UUID)
	}
	var rows []receiptTotal
	if err := tx.Model(&model.GoodsReceiptItem{}).
		Select("item_uuid, COALESCE(SUM(qty_received), 0) AS total").
		Where("item_uuid IN ?", ids).
		Group("item_uuid").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	totals := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ItemUUID)
		if err != nil {
			return nil, err
		}
		totals[id] = row.Total
	}
	return totals, nil
}

func (r *PurchaseOrderRepository) Pay(id, proof string, paidAt time.Time, bank string) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
			return err
		}
		// Poin 1: Wajib 100% barang diterima sebelum pelunasan
		if po.Status != common.StatusReceived {
			return errors.New("PO_NOT_FULLY_RECEIVED: Pelunasan tagihan hanya dapat diproses setelah seluruh barang pesanan 100% lengkap diterima di gudang.")
		}
		var amount int64
		for _, item := range po.Items {
			amount += int64(item.Received) * item.UnitCost
		}

		result := tx.Model(&po).
			Where("payment = ?", common.PaymentUnpaid).
			Updates(map[string]any{"payment": common.PaymentPaid, "proof": proof})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("purchase order sudah dibayar")
		}

		if paidAt.IsZero() {
			paidAt = time.Now().UTC()
		}
		if bank == "" {
			bank = "Bank Mandiri (1310-0982-4412 a.n. Vendor)"
		}

		if err := tx.Create(&model.PaymentRecord{
			PurchaseOrderUUID: po.UUID,
			Amount:            amount,
			PaidAt:            paidAt,
			ProofPath:         proof,
			Bank:              bank,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.Find(id)
}

type UnpaidRow struct {
	ID       string `json:"id"`
	Supplier string `json:"supplier"`
	Total    int64  `json:"total"`
}

func (r *PurchaseOrderRepository) Unpaid() ([]UnpaidRow, int64, error) {
	var orders []model.PurchaseOrder
	if err := r.db.Preload("Items").
		Where("payment = ? AND status = ?", common.PaymentUnpaid, common.StatusReceived).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]UnpaidRow, 0, len(orders))
	var total int64
	for _, po := range orders {
		var amount int64
		for _, item := range po.Items {
			amount += int64(item.Received) * item.UnitCost
		}
		rows = append(rows, UnpaidRow{po.ID, po.Supplier, amount})
		total += amount
	}
	return rows, total, nil
}

func (r *PurchaseOrderRepository) Report(from, to time.Time) (map[string]int64, error) {
	var orders []model.PurchaseOrder
	if err := r.db.Preload("Items").
		Where("received_at >= ? AND received_at < ?", from, to.Add(24*time.Hour)).
		Find(&orders).Error; err != nil {
		return nil, err
	}
	totals := map[string]int64{}
	for _, po := range orders {
		for _, item := range po.Items {
			totals[po.Supplier] += int64(item.Received) * item.UnitCost
		}
	}
	return totals, nil
}

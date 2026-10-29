package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m11/common"
	"m11/common/validation"
	"m11/dto"
	"m11/model"
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
	if err := r.db.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &po, nil
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
			if err := validation.ValidatePartialQuantity(orderItem.Quantity, totals[orderItem.UUID], item.QtyReceived); err != nil {
				return err
			}
			if err := tx.Create(&model.GoodsReceiptItem{
				ReceiptUUID: receipt.UUID, ItemUUID: orderItem.UUID, QtyReceived: item.QtyReceived,
			}).Error; err != nil {
				return err
			}
			receipt.Items = append(receipt.Items, model.GoodsReceiptItem{
				ReceiptUUID: receipt.UUID, ItemUUID: orderItem.UUID, QtyReceived: item.QtyReceived,
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
		return tx.Model(&po).Update("status", status).Error
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

func (r *PurchaseOrderRepository) Pay(id, proof string) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
			return err
		}
		if po.Status != common.StatusReceived && po.Status != common.StatusReceivedPartial {
			return errors.New("purchase order must be received before payment")
		}
		var amount int64
		for _, item := range po.Items {
			amount += int64(item.Received) * item.UnitCost
		}
		if err := tx.Model(&po).Updates(map[string]any{"payment": common.PaymentPaid, "proof": proof}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.PaymentRecord{
			PurchaseOrderUUID: po.UUID,
			Amount:            amount,
			PaidAt:            time.Now().UTC(),
			ProofPath:         proof,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	po.Payment, po.Proof = common.PaymentPaid, proof
	return &po, nil
}

type UnpaidRow struct {
	ID       string `json:"id"`
	Supplier string `json:"supplier"`
	Total    int64  `json:"total"`
}

func (r *PurchaseOrderRepository) Unpaid() ([]UnpaidRow, int64, error) {
	var orders []model.PurchaseOrder
	if err := r.db.Preload("Items").
		Where("payment = ? AND status IN ?", common.PaymentUnpaid, []string{common.StatusReceived, common.StatusReceivedPartial}).
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

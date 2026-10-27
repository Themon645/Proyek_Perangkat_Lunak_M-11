package model

import (
	"time"

	"m11/common/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurchaseOrder struct {
	ID            string              `json:"id" gorm:"primaryKey"`
	Supplier      string              `json:"supplier"`
	Status        string              `json:"status"`
	Payment       string              `json:"payment"`
	Proof         string              `json:"proof,omitempty"`
	Items         []PurchaseOrderItem `json:"items" gorm:"foreignKey:PurchaseOrderUUID;references:UUID;constraint:OnDelete:CASCADE"`
	GoodsReceipts []GoodsReceipt      `json:"goods_receipts,omitempty" gorm:"foreignKey:PurchaseOrderUUID;references:UUID;-:migration"`
	ReceivedAt    *time.Time          `json:"received_at,omitempty"`
	helper.AuditFields
}

func (p *PurchaseOrder) BeforeCreate(*gorm.DB) error {
	if p.UUID == uuid.Nil {
		p.UUID = uuid.New()
	}
	return nil
}

type PurchaseOrderItem struct {
	ID                string             `json:"id" gorm:"primaryKey"`
	PurchaseOrderUUID uuid.UUID          `json:"-"`
	Name              string             `json:"name"`
	Quantity          int                `json:"quantity"`
	UnitCost          int64              `json:"unit_cost"`
	Received          int                `json:"received"`
	GoodsReceiptItems []GoodsReceiptItem `json:"goods_receipt_items,omitempty" gorm:"foreignKey:ItemUUID;references:UUID;-:migration"`
	helper.AuditFields
}

func (p *PurchaseOrderItem) BeforeCreate(*gorm.DB) error {
	if p.UUID == uuid.Nil {
		p.UUID = uuid.New()
	}
	return nil
}

package model

import (
	"time"

	"m11/common/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurchaseOrder struct {
	ID            string              `json:"id" gorm:"primaryKey" bson:"_id"`
	Supplier      string              `json:"supplier" bson:"supplier"`
	Status        string              `json:"status" bson:"status"`
	Payment       string              `json:"payment" bson:"payment"`
	Proof         string              `json:"proof,omitempty" bson:"proof,omitempty"`
	Items         []PurchaseOrderItem `json:"items" gorm:"foreignKey:PurchaseOrderUUID;references:UUID;constraint:OnDelete:CASCADE" bson:"items"`
	GoodsReceipts []GoodsReceipt      `json:"goods_receipts,omitempty" gorm:"foreignKey:PurchaseOrderUUID;references:UUID;-:migration" bson:"-"`
	Payments      []PaymentRecord     `json:"payments,omitempty" gorm:"foreignKey:PurchaseOrderUUID;references:UUID;constraint:OnDelete:CASCADE" bson:"-"`
	ReceivedAt    *time.Time          `json:"received_at,omitempty" bson:"received_at,omitempty"`
	helper.AuditFields                `bson:"inline"`
}

func (p *PurchaseOrder) BeforeCreate(*gorm.DB) error {
	if p.UUID == uuid.Nil {
		p.UUID = uuid.New()
	}
	return nil
}

type PurchaseOrderItem struct {
	ID                string             `json:"id" gorm:"primaryKey" bson:"id"`
	PurchaseOrderUUID uuid.UUID          `json:"-" bson:"purchase_order_uuid,omitempty"`
	Name              string             `json:"name" bson:"name"`
	Quantity          int                `json:"quantity" bson:"quantity"`
	UnitCost          int64              `json:"unit_cost" bson:"unit_cost"`
	Received          int                `json:"received" bson:"received"`
	GoodsReceiptItems []GoodsReceiptItem `json:"goods_receipt_items,omitempty" gorm:"foreignKey:ItemUUID;references:UUID;-:migration" bson:"-"`
	helper.AuditFields                   `bson:"inline"`
}

func (p *PurchaseOrderItem) BeforeCreate(*gorm.DB) error {
	if p.UUID == uuid.Nil {
		p.UUID = uuid.New()
	}
	return nil
}

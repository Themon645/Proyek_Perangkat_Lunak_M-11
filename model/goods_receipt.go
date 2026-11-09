package model

import (
	"time"

	"m11/common/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoodsReceipt struct {
	ReceiptNo         string             `json:"receipt_no" gorm:"primaryKey" bson:"_id"`
	PurchaseOrderID   string             `json:"purchase_order_id,omitempty" gorm:"-" bson:"purchase_order_id,omitempty"`
	PurchaseOrderUUID uuid.UUID          `json:"-" bson:"purchase_order_uuid,omitempty"`
	ReceivedDate      time.Time          `json:"received_date" bson:"received_date"`
	Receiver          string             `json:"receiver" bson:"receiver"`
	Notes             string             `json:"notes" bson:"notes"`
	PurchaseOrder     PurchaseOrder      `json:"-" gorm:"foreignKey:PurchaseOrderUUID;references:UUID" bson:"-"`
	Items             []GoodsReceiptItem `json:"items" gorm:"foreignKey:ReceiptUUID;references:UUID;constraint:OnDelete:CASCADE" bson:"items"`
	helper.AuditFields                   `bson:"inline"`
}

func (g *GoodsReceipt) BeforeCreate(*gorm.DB) error {
	if g.UUID == uuid.Nil {
		g.UUID = uuid.New()
	}
	return nil
}

type GoodsReceiptItem struct {
	ID          string            `json:"id" gorm:"primaryKey" bson:"id"`
	ReceiptUUID uuid.UUID         `json:"-" bson:"receipt_uuid,omitempty"`
	ItemUUID    uuid.UUID         `json:"-" bson:"item_uuid,omitempty"`
	QtyReceived int               `json:"qty_received" bson:"qty_received"`
	Receipt     GoodsReceipt      `json:"-" gorm:"foreignKey:ReceiptUUID;references:UUID" bson:"-"`
	Item        PurchaseOrderItem `json:"-" gorm:"foreignKey:ItemUUID;references:UUID" bson:"item,omitempty"`
	helper.AuditFields                  `bson:"inline"`
}

func (g *GoodsReceiptItem) BeforeCreate(*gorm.DB) error {
	if g.UUID == uuid.Nil {
		g.UUID = uuid.New()
	}
	if g.ID == "" {
		g.ID = uuid.NewString()
	}
	return nil
}

package model

import (
	"time"

	"m11/common/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoodsReceipt struct {
	ReceiptNo         string             `json:"receipt_no" gorm:"primaryKey"`
	PurchaseOrderUUID uuid.UUID          `json:"-"`
	ReceivedDate      time.Time          `json:"received_date"`
	Receiver          string             `json:"receiver"`
	Notes             string             `json:"notes"`
	PurchaseOrder     PurchaseOrder      `json:"-" gorm:"foreignKey:PurchaseOrderUUID;references:UUID"`
	Items             []GoodsReceiptItem `json:"items" gorm:"foreignKey:ReceiptUUID;references:UUID;constraint:OnDelete:CASCADE"`
	helper.AuditFields
}

func (g *GoodsReceipt) BeforeCreate(*gorm.DB) error {
	if g.UUID == uuid.Nil {
		g.UUID = uuid.New()
	}
	return nil
}

type GoodsReceiptItem struct {
	ID          string            `json:"id" gorm:"primaryKey"`
	ReceiptUUID uuid.UUID         `json:"-"`
	ItemUUID    uuid.UUID         `json:"-"`
	QtyReceived int               `json:"qty_received"`
	Receipt     GoodsReceipt      `json:"-" gorm:"foreignKey:ReceiptUUID;references:UUID"`
	Item        PurchaseOrderItem `json:"-" gorm:"foreignKey:ItemUUID;references:UUID"`
	helper.AuditFields
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

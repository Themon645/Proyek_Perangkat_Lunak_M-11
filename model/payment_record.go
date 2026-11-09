package model

import (
	"time"

	"m11/common/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRecord struct {
	ID                uint          `json:"id" gorm:"primaryKey" bson:"id,omitempty"`
	PurchaseOrderID   string        `json:"purchase_order_id,omitempty" gorm:"-" bson:"purchase_order_id,omitempty"`
	PurchaseOrderUUID uuid.UUID     `json:"-" bson:"purchase_order_uuid,omitempty"`
	Amount            int64         `json:"amount" gorm:"not null" bson:"amount"`
	PaidAt            time.Time     `json:"paid_at" gorm:"not null" bson:"paid_at"`
	ProofPath         string        `json:"proof_path" gorm:"not null" bson:"proof_path"`
	PurchaseOrder     PurchaseOrder `json:"-" gorm:"foreignKey:PurchaseOrderUUID;references:UUID" bson:"-"`
	helper.AuditFields              `bson:"inline"`
}

func (p *PaymentRecord) BeforeCreate(*gorm.DB) error {
	if p.UUID == uuid.Nil {
		p.UUID = uuid.New()
	}
	return nil
}

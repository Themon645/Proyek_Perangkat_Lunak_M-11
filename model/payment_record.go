package model

import (
	"time"

	"m11/common/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRecord struct {
	ID                uint          `json:"id" gorm:"primaryKey"`
	PurchaseOrderUUID uuid.UUID     `json:"-"`
	Amount            int64         `json:"amount" gorm:"not null"`
	PaidAt            time.Time     `json:"paid_at" gorm:"not null"`
	ProofPath         string        `json:"proof_path" gorm:"not null"`
	PurchaseOrder     PurchaseOrder `json:"-" gorm:"foreignKey:PurchaseOrderUUID;references:UUID"`
	helper.AuditFields
}

func (p *PaymentRecord) BeforeCreate(*gorm.DB) error {
	if p.UUID == uuid.Nil {
		p.UUID = uuid.New()
	}
	return nil
}

package model

import "time"

type PurchaseOrder struct {
	ID         string              `json:"id" gorm:"primaryKey"`
	Supplier   string              `json:"supplier"`
	Status     string              `json:"status"`
	Payment    string              `json:"payment"`
	Proof      string              `json:"proof,omitempty"`
	Items      []PurchaseOrderItem `json:"items" gorm:"foreignKey:PurchaseOrderID;constraint:OnDelete:CASCADE"`
	ReceivedAt *time.Time          `json:"received_at,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
}

type PurchaseOrderItem struct {
	ID              string `json:"id" gorm:"primaryKey"`
	PurchaseOrderID string `json:"-"`
	Name            string `json:"name"`
	Quantity        int    `json:"quantity"`
	UnitCost        int64  `json:"unit_cost"`
	Received        int    `json:"received"`
}

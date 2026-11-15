package resource

import (
	"strings"

	"m11/common/helper"
	"m11/model"
)

type PurchaseOrder struct {
	ID            string              `json:"id"`
	Supplier      string              `json:"supplier"`
	Status        string              `json:"status"`
	Payment       string              `json:"payment"`
	Proof         string              `json:"proof,omitempty"`
	ProofURL      string              `json:"proof_url,omitempty"`
	AttachmentURL string              `json:"attachment_url,omitempty"`
	Total         int64               `json:"total"`
	Items         []PurchaseOrderItem `json:"items"`
}

type PurchaseOrderItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	UnitCost    int64  `json:"unit_cost"`
	QtyReceived int    `json:"qty_received"`
}

func NewPurchaseOrder(po model.PurchaseOrder) PurchaseOrder {
	proof := strings.TrimSpace(po.Proof)
	if proof == "" && len(po.Payments) > 0 {
		proof = strings.TrimSpace(po.Payments[len(po.Payments)-1].ProofPath)
	}

	proofURL, attachmentURL, cloudURL := helper.ResolveProofURLs(proof)
	formattedProof := cloudURL
	if formattedProof == "" {
		formattedProof = attachmentURL
	}

	result := PurchaseOrder{
		ID:            po.ID,
		Supplier:      po.Supplier,
		Status:        po.Status,
		Payment:       po.Payment,
		Proof:         formattedProof,
		ProofURL:      proofURL,
		AttachmentURL: attachmentURL,
		Items:         make([]PurchaseOrderItem, 0, len(po.Items)),
	}
	for _, item := range po.Items {
		result.Total += int64(item.Quantity) * item.UnitCost
		result.Items = append(result.Items, PurchaseOrderItem{
			ID: item.ID, Name: item.Name, Quantity: item.Quantity,
			UnitCost: item.UnitCost, QtyReceived: item.Received,
		})
	}
	return result
}

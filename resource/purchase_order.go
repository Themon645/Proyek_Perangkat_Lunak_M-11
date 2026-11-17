package resource

import (
	"path/filepath"
	"strings"

	"m11/common/helper"
	"m11/model"
)

type PaymentRecordResponse struct {
	ID        uint   `json:"id"`
	Amount    int64  `json:"amount"`
	PaidAt    string `json:"paid_at"`
	ProofPath string `json:"proof_path"`
	ProofURL  string `json:"proof_url"`
	Bank      string `json:"bank,omitempty"`
}

type PurchaseOrder struct {
	ID                   string                  `json:"id"`
	NomorPO              string                  `json:"nomor_po,omitempty"`
	Supplier             string                  `json:"supplier"`
	SupplierName         string                  `json:"supplier_name,omitempty"`
	Status               string                  `json:"status"`
	Payment              string                  `json:"payment"`
	Proof                string                  `json:"proof,omitempty"`
	ProofURL             string                  `json:"proof_url,omitempty"`
	AttachmentURL        string                  `json:"attachment_url,omitempty"`
	ProofPath            string                  `json:"proof_path,omitempty"`
	BuktiTransfer        string                  `json:"bukti_transfer,omitempty"`
	BuktiTransferPreview string                  `json:"bukti_transfer_preview,omitempty"`
	Total                int64                   `json:"total"`
	TotalTagihan         int64                   `json:"total_tagihan,omitempty"`
	OrderDate            string                  `json:"order_date,omitempty"`
	Tanggal              string                  `json:"tanggal,omitempty"`
	CreatedAt            string                  `json:"created_at,omitempty"`
	ReceivedAt           *string                 `json:"received_at,omitempty"`
	PaidAt               *string                 `json:"paid_at,omitempty"`
	TanggalBayar         *string                 `json:"tanggal_bayar,omitempty"`
	PaymentDate          *string                 `json:"payment_date,omitempty"`
	BankTujuan           string                  `json:"bank_tujuan,omitempty"`
	Bank                 string                  `json:"bank,omitempty"`
	Items                []PurchaseOrderItem     `json:"items"`
	Payments             []PaymentRecordResponse `json:"payments,omitempty"`
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
	var lastPaymentBank string
	var lastPaymentTime string
	paymentsRes := make([]PaymentRecordResponse, 0, len(po.Payments))

	if len(po.Payments) > 0 {
		for _, p := range po.Payments {
			pProofURL, _, _ := helper.ResolveProofURLs(p.ProofPath)
			paidAtStr := p.PaidAt.Format("2006-01-02")
			paymentsRes = append(paymentsRes, PaymentRecordResponse{
				ID:        p.ID,
				Amount:    p.Amount,
				PaidAt:    paidAtStr,
				ProofPath: p.ProofPath,
				ProofURL:  pProofURL,
				Bank:      p.Bank,
			})
		}
		last := po.Payments[len(po.Payments)-1]
		if proof == "" {
			proof = strings.TrimSpace(last.ProofPath)
		}
		lastPaymentBank = last.Bank
		lastPaymentTime = last.PaidAt.Format("2006-01-02")
	}

	proofURL, attachmentURL, cloudURL := helper.ResolveProofURLs(proof)
	formattedProof := cloudURL
	if formattedProof == "" {
		formattedProof = attachmentURL
	}

	var previewURL string
	ext := strings.ToLower(filepath.Ext(proof))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		previewURL = formattedProof
	}

	orderDate := po.AuditFields.CreatedAt.Format("2006-01-02")
	if orderDate == "0001-01-01" {
		orderDate = "2026-11-10"
	}

	var receivedAtStr *string
	if po.ReceivedAt != nil {
		s := po.ReceivedAt.Format("2006-01-02")
		receivedAtStr = &s
	}

	var paidAtStr *string
	if lastPaymentTime != "" {
		paidAtStr = &lastPaymentTime
	} else if po.Payment == "paid" {
		paidAtStr = &orderDate
	}

	if lastPaymentBank == "" && po.Payment == "paid" {
		lastPaymentBank = "Bank Mandiri (1310-0982-4412 a.n. Vendor)"
	}

	// Check if all items are fully received
	complete := true
	for _, item := range po.Items {
		if item.Received < item.Quantity {
			complete = false
			break
		}
	}

	status := po.Status
	payment := po.Payment
	// D03: For partial reception that has recorded a payment, keep status as received partial
	// and payment as partially paid to ensure Catat GRN remains active for remaining items.
	if !complete && status == "received partial" && payment == "paid" {
		payment = "partially paid"
	}

	result := PurchaseOrder{
		ID:                   po.ID,
		NomorPO:              po.ID,
		Supplier:             po.Supplier,
		SupplierName:         po.Supplier,
		Status:               status,
		Payment:              payment,
		Proof:                formattedProof,
		ProofURL:             proofURL,
		AttachmentURL:        attachmentURL,
		ProofPath:            proof,
		BuktiTransfer:        formattedProof,
		BuktiTransferPreview: previewURL,
		OrderDate:            orderDate,
		Tanggal:              orderDate,
		CreatedAt:            orderDate,
		ReceivedAt:           receivedAtStr,
		PaidAt:               paidAtStr,
		TanggalBayar:         paidAtStr,
		PaymentDate:          paidAtStr,
		BankTujuan:           lastPaymentBank,
		Bank:                 lastPaymentBank,
		Items:                make([]PurchaseOrderItem, 0, len(po.Items)),
		Payments:             paymentsRes,
	}

	for _, item := range po.Items {
		result.Total += int64(item.Quantity) * item.UnitCost
		result.Items = append(result.Items, PurchaseOrderItem{
			ID:          item.ID,
			Name:        item.Name,
			Quantity:    item.Quantity,
			UnitCost:    item.UnitCost,
			QtyReceived: item.Received,
		})
	}
	result.TotalTagihan = result.Total

	return result
}

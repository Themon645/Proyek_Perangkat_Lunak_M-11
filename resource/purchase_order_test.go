package resource

import (
	"strings"
	"testing"

	"m11/model"
)

func TestNewPurchaseOrderMapsTotalsAndReceivedQuantity(t *testing.T) {
	result := NewPurchaseOrder(model.PurchaseOrder{
		ID:    "PO-1",
		Proof: "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf",
		Items: []model.PurchaseOrderItem{{
			ID: "ITEM-1", Name: "Laptop", Quantity: 2, UnitCost: 100, Received: 1,
		}},
	})

	if result.Total != 200 || len(result.Items) != 1 || result.Items[0].QtyReceived != 1 {
		t.Fatalf("unexpected purchase order resource: %+v", result)
	}
	if result.Proof != "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf" {
		t.Fatalf("unexpected proof: %s", result.Proof)
	}
	if result.ProofURL != "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf" {
		t.Fatalf("unexpected proof_url: %s", result.ProofURL)
	}
	if result.AttachmentURL != "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf" {
		t.Fatalf("unexpected attachment_url: %s", result.AttachmentURL)
	}
}

func TestNewPurchaseOrderResolvesLocalProofToFullDownloadableLink(t *testing.T) {
	result := NewPurchaseOrder(model.PurchaseOrder{
		ID:    "PO-2",
		Proof: "storage/payment-proofs/3f8b9c042cbeecb57f69983b0efb3829.png",
		Items: []model.PurchaseOrderItem{{
			ID: "ITEM-2", Name: "Mouse", Quantity: 1, UnitCost: 50, Received: 1,
		}},
	})

	if !strings.HasPrefix(result.Proof, "http") {
		t.Fatalf("expected full link for proof, got %s", result.Proof)
	}
	if !strings.HasPrefix(result.ProofURL, "http") {
		t.Fatalf("expected full link for proof_url, got %s", result.ProofURL)
	}
	if !strings.HasPrefix(result.AttachmentURL, "http") || !strings.HasSuffix(result.AttachmentURL, "3f8b9c042cbeecb57f69983b0efb3829.png") {
		t.Fatalf("expected full downloadable link for attachment_url, got %s", result.AttachmentURL)
	}
}

func TestNewGoodsReceiptResponse(t *testing.T) {
	res := NewGoodsReceiptResponse(model.GoodsReceipt{
		ReceiptNo:       "SJ-001",
		PurchaseOrderID: "PO-001",
		Receiver:        "Staff Gudang",
		Notes:           "Kondisi baik",
		Items: []model.GoodsReceiptItem{{
			QtyReceived: 10,
			Item: model.PurchaseOrderItem{
				ID:   "ITEM-1",
				Name: "Kertas A4",
			},
		}},
	})

	if res.ReceiptNo != "SJ-001" || res.PurchaseOrderID != "PO-001" {
		t.Fatalf("unexpected goods receipt response: %+v", res)
	}
	if len(res.Items) != 1 || res.Items[0].ItemID != "ITEM-1" || res.Items[0].ItemName != "Kertas A4" {
		t.Fatalf("unexpected goods receipt items: %+v", res.Items)
	}
}

package repository

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"m11/common"
	"m11/database"
	"m11/dto"
	"m11/model"
)

func TestMongoPurchaseOrderFlow(t *testing.T) {
	_ = godotenv.Load("../.env")
	if os.Getenv("MONGODB_URI") == "" {
		t.Skip("MONGODB_URI not set, skipping MongoDB repository tests")
	}

	mongoDB, err := database.GetMongoDatabase()
	if err != nil {
		t.Fatalf("Failed to get mongo database: %v", err)
	}

	// 1. Reset and Seed clean data
	if err := database.ResetAndSeedMongo(mongoDB); err != nil {
		t.Fatalf("Failed to reset and seed mongo: %v", err)
	}

	repo := NewMongoPurchaseOrderRepository(mongoDB)

	// 2. Test List
	orders, err := repo.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(orders) != 10 {
		t.Fatalf("Expected 10 orders, got %d", len(orders))
	}

	// 3. Test Find
	po, err := repo.Find("PO-2026-001")
	if err != nil {
		t.Fatalf("Find error: %v", err)
	}
	if po.Supplier != "PT Graha Sarana Mitra" {
		t.Fatalf("Unexpected supplier: %s", po.Supplier)
	}

	// 4. Test ReceiptHistory
	foundPO, receipts, err := repo.ReceiptHistory("PO-2026-001")
	if err != nil {
		t.Fatalf("ReceiptHistory error: %v", err)
	}
	if foundPO.ID != "PO-2026-001" {
		t.Fatalf("Expected PO-2026-001, got %s", foundPO.ID)
	}
	if len(receipts) == 0 {
		t.Fatal("Expected at least 1 receipt in history")
	}

	// 5. Test Create PO
	newPO := &model.PurchaseOrder{
		ID:       "PO-TEST-001",
		Supplier: "PT Test Supplier",
		Status:   common.StatusApproved,
		Payment:  common.PaymentUnpaid,
		Items: []model.PurchaseOrderItem{
			{
				ID:       "ITEM-T-01",
				Name:     "Test Item",
				Quantity: 20,
				UnitCost: 50000,
			},
		},
	}
	if err := repo.Create(newPO); err != nil {
		t.Fatalf("Create PO error: %v", err)
	}

	// 6. Test Receive partial
	updatedPO, err := repo.Receive("PO-TEST-001", map[string]int{"ITEM-T-01": 10})
	if err != nil {
		t.Fatalf("Receive error: %v", err)
	}
	if updatedPO.Status != common.StatusReceivedPartial {
		t.Fatalf("Expected status 'received partial', got %s", updatedPO.Status)
	}

	// 7. Test CreateGoodsReceipt completing the remaining 10 items (making total 20)
	receipt1, err := repo.CreateGoodsReceipt(&dto.CreateGoodsReceiptRequest{
		ReceiptNo:       "SJ-TEST-001",
		PurchaseOrderID: "PO-TEST-001",
		Receiver:        "Admin Test",
		Notes:           "Penerimaan 1",
		Items: []dto.CreateGoodsReceiptItem{
			{ItemID: "ITEM-T-01", QtyReceived: 10},
		},
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateGoodsReceipt 1 error: %v", err)
	}
	if receipt1.ReceiptNo != "SJ-TEST-001" {
		t.Fatalf("Expected receipt SJ-TEST-001, got %s", receipt1.ReceiptNo)
	}

	receipt2, err := repo.CreateGoodsReceipt(&dto.CreateGoodsReceiptRequest{
		ReceiptNo:       "SJ-TEST-002",
		PurchaseOrderID: "PO-TEST-001",
		Receiver:        "Admin Test",
		Notes:           "Penerimaan 2 (Pelunasan barang)",
		Items: []dto.CreateGoodsReceiptItem{
			{ItemID: "ITEM-T-01", QtyReceived: 10},
		},
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateGoodsReceipt 2 error: %v", err)
	}
	if receipt2.ReceiptNo != "SJ-TEST-002" {
		t.Fatalf("Expected receipt SJ-TEST-002, got %s", receipt2.ReceiptNo)
	}

	// Check that PO status is now fully received
	poCompleted, err := repo.Find("PO-TEST-001")
	if err != nil {
		t.Fatalf("Find error: %v", err)
	}
	if poCompleted.Status != common.StatusReceived {
		t.Fatalf("Expected status 'received', got %s", poCompleted.Status)
	}

	// 8. Test Pay
	paidPO, err := repo.Pay("PO-TEST-001", "proofs/transfer-test.pdf")
	if err != nil {
		t.Fatalf("Pay error: %v", err)
	}
	if paidPO.Payment != common.PaymentPaid {
		t.Fatalf("Expected payment 'paid', got %s", paidPO.Payment)
	}

	// 9. Test Unpaid
	unpaidRows, total, err := repo.Unpaid()
	if err != nil {
		t.Fatalf("Unpaid error: %v", err)
	}
	if len(unpaidRows) == 0 || total == 0 {
		t.Fatal("Expected unpaid rows and non-zero total")
	}

	// 10. Test Report
	report, err := repo.Report(time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Report error: %v", err)
	}
	if len(report) == 0 {
		t.Fatal("Expected report entries")
	}
}

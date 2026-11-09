package database

import (
	"context"
	"fmt"
	"time"

	"m11/common"
	"m11/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CleanMongo membersihkan seluruh collection data di MongoDB agar database bersih.
func CleanMongo(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections := []string{"purchase_orders", "goods_receipts", "payment_records"}
	for _, col := range collections {
		if err := db.Collection(col).Drop(ctx); err != nil {
			return fmt.Errorf("gagal membersihkan collection %s: %w", col, err)
		}
	}
	return nil
}

// SeedMongo mengisi MongoDB dengan data awal (10 PO & 10 Goods Receipt) jika collection masih kosong.
func SeedMongo(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	count, err := db.Collection("purchase_orders").CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return ResetAndSeedMongo(db)
}

// ResetAndSeedMongo membersihkan data lama dan mengisi data awal yang rapi.
func ResetAndSeedMongo(db *mongo.Database) error {
	if err := CleanMongo(db); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data := []struct {
		supplier, itemName, receiptNo string
		qty                           int
		unitCost                      int64
	}{
		{"PT Graha Sarana Mitra", "Kertas HVS A4 80gr Sinar Dunia", "SJ-2026/X/9812", 100, 65000},
		{"CV Mandiri Sukses Perkasa", "Tinta Printer Epson L3110 Black Original", "SJ-2026/X/9813", 50, 185000},
		{"PT Pasundan Mega Elektrik", "Kabel LAN Cat6 305 Meter", "SJ-2026/X/9814", 20, 1250000},
		{"CV Cipta Stationery Sentosa", "Stopmap Folio Karton Tebal", "SJ-2026/X/9815", 100, 12000},
		{"PT Pasundan Niaga Logistik", "Kursi Kantor Ergonomis", "SJ-2026/X/9816", 15, 1850000},
		{"PT Sinar IT Solusindo", "SSD NVMe 1TB Kingston", "SJ-2026/X/9817", 12, 1450000},
		{"CV Prima Abadi Furniture", "Meja Kerja Modular 120 cm", "SJ-2026/X/9818", 10, 2350000},
		{"PT Pasundan Cipta Grafika", "Toner Printer HP 136A", "SJ-2026/X/9819", 25, 980000},
		{"CV Citra Makmur Sarana", "Lampu LED Panel 36 Watt", "SJ-2026/X/9820", 40, 175000},
		{"PT Global Network Prima", "Access Point TP-Link EAP225", "SJ-2026/X/9821", 8, 1125000},
	}

	poDocs := make([]any, 0, len(data))
	receiptDocs := make([]any, 0, len(data))

	for i, item := range data {
		poID := fmt.Sprintf("PO-2026-%03d", i+1)
		itemID := fmt.Sprintf("ITEM-%d", 101+i)
		poUUID := uuid.New()
		itemUUID := uuid.New()
		receiptUUID := uuid.New()
		qty := item.qty
		now := time.Now().UTC()

		po := model.PurchaseOrder{
			ID:       poID,
			Supplier: item.supplier,
			Status:   common.StatusReceived,
			Payment:  common.PaymentUnpaid,
			Items: []model.PurchaseOrderItem{{
				ID:                itemID,
				PurchaseOrderUUID: poUUID,
				Name:              item.itemName,
				Quantity:          qty,
				UnitCost:          item.unitCost,
				Received:          qty,
			}},
			ReceivedAt: &now,
		}
		po.AuditFields.UUID = poUUID
		po.AuditFields.CreatedAt = now
		po.AuditFields.UpdatedAt = now
		po.Items[0].AuditFields.UUID = itemUUID
		po.Items[0].AuditFields.CreatedAt = now
		po.Items[0].AuditFields.UpdatedAt = now

		receipt := model.GoodsReceipt{
			ReceiptNo:         item.receiptNo,
			PurchaseOrderID:   poID,
			PurchaseOrderUUID: poUUID,
			ReceivedDate:      now,
			Receiver:          "Tyezar Hasan",
			Notes:             "Barang diterima dalam kondisi baik",
			Items: []model.GoodsReceiptItem{{
				ID:          fmt.Sprintf("GRN-ITEM-%03d", i+1),
				ReceiptUUID: receiptUUID,
				ItemUUID:    itemUUID,
				QtyReceived: qty,
				Item: model.PurchaseOrderItem{
					ID:   itemID,
					Name: item.itemName,
				},
			}},
		}
		receipt.AuditFields.UUID = receiptUUID
		receipt.AuditFields.CreatedAt = now
		receipt.AuditFields.UpdatedAt = now

		poDocs = append(poDocs, po)
		receiptDocs = append(receiptDocs, receipt)
	}

	if len(poDocs) > 0 {
		if _, err := db.Collection("purchase_orders").InsertMany(ctx, poDocs); err != nil {
			return err
		}
	}

	if len(receiptDocs) > 0 {
		if _, err := db.Collection("goods_receipts").InsertMany(ctx, receiptDocs); err != nil {
			return err
		}
	}

	return nil
}


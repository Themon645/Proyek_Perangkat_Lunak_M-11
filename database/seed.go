package database

import (
	"fmt"
	"time"

	"m11/common"
	"m11/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedDummy(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.PurchaseOrder{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
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
			po.Items[0].AuditFields.UUID = itemUUID
			if err := tx.Create(&po).Error; err != nil {
				return err
			}
			receipt := model.GoodsReceipt{
				ReceiptNo:         item.receiptNo,
				PurchaseOrderUUID: poUUID,
				ReceivedDate:      now,
				Receiver:          "Tyezar Hasan",
				Notes:             "Barang diterima dalam kondisi baik",
				Items: []model.GoodsReceiptItem{{
					ReceiptUUID: receiptUUID,
					ItemUUID:    itemUUID,
					QtyReceived: qty,
				}},
			}
			receipt.AuditFields.UUID = receiptUUID
			if err := tx.Create(&receipt).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

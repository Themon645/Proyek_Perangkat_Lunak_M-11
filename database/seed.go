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
			qty, receivedQty              int
			unitCost                      int64
			status                        string
			payment                       string
			hasReceipt                    bool
		}{
			// 1. PO Disetujui (Approved) - Siap dicatat GRN
			{"PT Graha Sarana Mitra", "Kertas HVS A4 80gr Sinar Dunia", "", 100, 0, 65000, common.StatusApproved, common.PaymentUnpaid, false},
			{"CV Mandiri Sukses Perkasa", "Tinta Printer Epson L3110 Black Original", "", 50, 0, 185000, common.StatusApproved, common.PaymentUnpaid, false},

			// 2. PO Diterima Sebagian (Received Partial)
			{"PT Pasundan Mega Elektrik", "Kabel LAN Cat6 305 Meter", "SJ-2026/X/9814", 20, 10, 1250000, common.StatusReceivedPartial, common.PaymentUnpaid, true},

			// 3. PO Disetujui (Approved) - Siap dicatat GRN
			{"CV Cipta Stationery Sentosa", "Stopmap Folio Karton Tebal", "", 100, 0, 12000, common.StatusApproved, common.PaymentUnpaid, false},

			// 4. PO Diterima Penuh (Received) - Siap dibayar
			{"PT Pasundan Niaga Logistik", "Kursi Kantor Ergonomis", "SJ-2026/X/9816", 15, 15, 1850000, common.StatusReceived, common.PaymentUnpaid, true},

			// 5. PO Disetujui (Approved) - Siap dicatat GRN
			{"PT Sinar IT Solusindo", "SSD NVMe 1TB Kingston", "", 12, 0, 1450000, common.StatusApproved, common.PaymentUnpaid, false},

			// 6. PO Diterima Sebagian (Received Partial)
			{"CV Prima Abadi Furniture", "Meja Kerja Modular 120 cm", "SJ-2026/X/9818", 10, 5, 2350000, common.StatusReceivedPartial, common.PaymentUnpaid, true},

			// 7. PO Selesai & Lunas (Received & Paid)
			{"PT Pasundan Cipta Grafika", "Toner Printer HP 136A", "SJ-2026/X/9819", 25, 25, 980000, common.StatusReceived, common.PaymentPaid, true},

			// 8. PO Baru Diajukan (Submitted)
			{"CV Citra Makmur Sarana", "Lampu LED Panel 36 Watt", "", 40, 0, 175000, common.StatusPending, common.PaymentUnpaid, false},

			// 9. PO Disetujui (Approved) - Siap dicatat GRN
			{"PT Global Network Prima", "Access Point TP-Link EAP225", "", 8, 0, 1125000, common.StatusApproved, common.PaymentUnpaid, false},
		}

		for i, item := range data {
			poID := fmt.Sprintf("PO-2026-%03d", i+1)
			itemID := fmt.Sprintf("ITEM-%d", 101+i)
			poUUID := uuid.New()
			itemUUID := uuid.New()
			now := time.Now().UTC()

			po := model.PurchaseOrder{
				ID:       poID,
				Supplier: item.supplier,
				Status:   item.status,
				Payment:  item.payment,
				Items: []model.PurchaseOrderItem{{
					ID:                itemID,
					PurchaseOrderUUID: poUUID,
					Name:              item.itemName,
					Quantity:          item.qty,
					UnitCost:          item.unitCost,
					Received:          item.receivedQty,
				}},
			}
			if item.status == common.StatusReceived || item.status == common.StatusReceivedPartial {
				po.ReceivedAt = &now
			}
			po.AuditFields.UUID = poUUID
			po.Items[0].AuditFields.UUID = itemUUID

			if err := tx.Create(&po).Error; err != nil {
				return err
			}

			// Buat Goods Receipt jika ada penerimaan
			if item.hasReceipt {
				receiptUUID := uuid.New()
				receipt := model.GoodsReceipt{
					ReceiptNo:         item.receiptNo,
					PurchaseOrderUUID: poUUID,
					ReceivedDate:      now,
					Receiver:          "Tyezar Hasan",
					Notes:             "Barang diterima dalam kondisi baik",
					Items: []model.GoodsReceiptItem{{
						ReceiptUUID: receiptUUID,
						ItemUUID:    itemUUID,
						QtyReceived: item.receivedQty,
					}},
				}
				receipt.AuditFields.UUID = receiptUUID
				if err := tx.Create(&receipt).Error; err != nil {
					return err
				}
			}

			// Buat Payment Record jika status sudah lunas
			if item.payment == common.PaymentPaid {
				paymentRec := model.PaymentRecord{
					PurchaseOrderUUID: poUUID,
					Amount:            int64(item.receivedQty) * item.unitCost,
					PaidAt:            now,
					ProofPath:         "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf",
				}
				if err := tx.Create(&paymentRec).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// CleanPostgres mengosongkan seluruh tabel data pengadaan pada PostgreSQL / Neon.
func CleanPostgres(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM goods_receipt_items").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM goods_receipts").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM payment_records").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM purchase_order_items").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM purchase_orders").Error; err != nil {
			return err
		}
		return nil
	})
}

// ResetAndSeedPostgres membersihkan data lama dan mengisi ulang 10 dummy PO & Goods Receipt dengan variasi status realistis.
func ResetAndSeedPostgres(db *gorm.DB) error {
	if err := CleanPostgres(db); err != nil {
		return err
	}
	return SeedDummy(db)
}

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
			orderDateStr                  string
			receivedDateStr               string
			paidDateStr                   string
			bank                          string
		}{
			// 1. PO-2026-001: Disetujui (Approved) - Siap dicatat GRN
			{
				supplier:        "PT Graha Sarana Mitra",
				itemName:        "Kertas HVS A4 80gr Sinar Dunia",
				receiptNo:       "",
				qty:             100,
				receivedQty:     0,
				unitCost:        65000,
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 08:30:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 2. PO-2026-002: Selesai & Lunas (Received & Paid)
			{
				supplier:        "CV Mandiri Sukses Perkasa",
				itemName:        "Tinta Printer Epson L3110 Black Original",
				receiptNo:       "SJ-2026/X/9815",
				qty:             50,
				receivedQty:     50,
				unitCost:        185000,
				status:          common.StatusReceived,
				payment:         common.PaymentPaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 09:00:00",
				receivedDateStr: "2026-11-12 10:15:00",
				paidDateStr:     "2026-11-13 14:00:00",
				bank:            "Bank Mandiri (1310-0982-4412 a.n. Vendor)",
			},
			// 3. PO-2026-003: Diterima Penuh tapi Belum Bayar (Received & Unpaid) - Tagihan Belum Lunas
			{
				supplier:        "PT Pasundan Mega Elektrik",
				itemName:        "Kabel LAN Cat6 305 Meter",
				receiptNo:       "SJ-2026/X/9814",
				qty:             20,
				receivedQty:     20,
				unitCost:        1250000,
				status:          common.StatusReceived,
				payment:         common.PaymentUnpaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 09:30:00",
				receivedDateStr: "2026-11-12 11:00:00",
				paidDateStr:     "",
				bank:            "",
			},
			// 4. PO-2026-004: Disetujui (Approved) - Siap dicatat GRN
			{
				supplier:        "CV Cipta Stationery Sentosa",
				itemName:        "Stopmap Folio Karton Tebal",
				receiptNo:       "",
				qty:             100,
				receivedQty:     0,
				unitCost:        12000,
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 10:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 5. PO-2026-005: Selesai & Lunas (Received & Paid)
			{
				supplier:        "PT Pasundan Niaga Logistik",
				itemName:        "Kursi Kantor Ergonomis",
				receiptNo:       "SJ-2026/X/9816",
				qty:             15,
				receivedQty:     15,
				unitCost:        1850000,
				status:          common.StatusReceived,
				payment:         common.PaymentPaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 10:30:00",
				receivedDateStr: "2026-11-12 13:00:00",
				paidDateStr:     "2026-11-13 15:30:00",
				bank:            "Bank Mandiri (1310-0982-4412 a.n. Vendor)",
			},
			// 6. PO-2026-006: Disetujui (Approved) - Siap dicatat GRN
			{
				supplier:        "PT Sinar IT Solusindo",
				itemName:        "SSD NVMe 1TB Kingston",
				receiptNo:       "",
				qty:             12,
				receivedQty:     0,
				unitCost:        1450000,
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 11:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 7. PO-2026-007: Diterima Sebagian (Received Partial & Unpaid) - Siap dicatat GRN sisa & Bayar
			{
				supplier:        "CV Prima Abadi Furniture",
				itemName:        "Meja Kerja Modular 120 cm",
				receiptNo:       "SJ-2026/X/9818",
				qty:             10,
				receivedQty:     5,
				unitCost:        2350000,
				status:          common.StatusReceivedPartial,
				payment:         common.PaymentUnpaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 11:30:00",
				receivedDateStr: "2026-11-11 14:00:00",
				paidDateStr:     "",
				bank:            "",
			},
			// 8. PO-2026-008: Selesai & Lunas (Received & Paid)
			{
				supplier:        "PT Pasundan Cipta Grafika",
				itemName:        "Toner Printer HP 136A",
				receiptNo:       "SJ-2026/X/9819",
				qty:             25,
				receivedQty:     25,
				unitCost:        980000,
				status:          common.StatusReceived,
				payment:         common.PaymentPaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 13:00:00",
				receivedDateStr: "2026-11-12 15:00:00",
				paidDateStr:     "2026-11-13 16:00:00",
				bank:            "Bank BNI (0882-9912-331 a.n. Vendor)",
			},
			// 9. PO-2026-009: Baru Diajukan (Submitted / Pending) - Tidak punya tombol aksi
			{
				supplier:        "CV Citra Makmur Sarana",
				itemName:        "Lampu LED Panel 36 Watt",
				receiptNo:       "",
				qty:             40,
				receivedQty:     0,
				unitCost:        175000,
				status:          common.StatusPending,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 14:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 10. PO-2026-010: Disetujui (Approved) - Siap dicatat GRN
			{
				supplier:        "PT Global Network Prima",
				itemName:        "Access Point TP-Link EAP225",
				receiptNo:       "",
				qty:             8,
				receivedQty:     0,
				unitCost:        1125000,
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 15:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
		}

		for i, item := range data {
			poID := fmt.Sprintf("PO-2026-%03d", i+1)
			itemID := fmt.Sprintf("ITEM-%d", 101+i)
			poUUID := uuid.New()
			itemUUID := uuid.New()

			orderTime, _ := time.Parse("2006-01-02 15:04:05", item.orderDateStr)

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
			po.AuditFields.CreatedAt = orderTime
			po.AuditFields.UpdatedAt = orderTime
			po.Items[0].AuditFields.CreatedAt = orderTime
			po.Items[0].AuditFields.UpdatedAt = orderTime

			if item.payment == common.PaymentPaid {
				po.Proof = "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf"
			}
			if item.receivedDateStr != "" {
				recTime, _ := time.Parse("2006-01-02 15:04:05", item.receivedDateStr)
				po.ReceivedAt = &recTime
			}
			po.AuditFields.UUID = poUUID
			po.Items[0].AuditFields.UUID = itemUUID

			if err := tx.Create(&po).Error; err != nil {
				return err
			}

			// Buat Goods Receipt jika ada penerimaan
			if item.hasReceipt {
				recTime, _ := time.Parse("2006-01-02 15:04:05", item.receivedDateStr)
				receiptUUID := uuid.New()
				receipt := model.GoodsReceipt{
					ReceiptNo:         item.receiptNo,
					PurchaseOrderUUID: poUUID,
					ReceivedDate:      recTime,
					Receiver:          "Tyezar Hasan",
					Notes:             "Barang diterima dalam kondisi baik",
					Items: []model.GoodsReceiptItem{{
						ReceiptUUID: receiptUUID,
						ItemUUID:    itemUUID,
						QtyReceived: item.receivedQty,
					}},
				}
				receipt.AuditFields.UUID = receiptUUID
				receipt.AuditFields.CreatedAt = recTime
				receipt.AuditFields.UpdatedAt = recTime
				if err := tx.Create(&receipt).Error; err != nil {
					return err
				}
			}

			// Buat Payment Record jika status sudah lunas
			if item.payment == common.PaymentPaid {
				paidTime, _ := time.Parse("2006-01-02 15:04:05", item.paidDateStr)
				paymentRec := model.PaymentRecord{
					PurchaseOrderUUID: poUUID,
					Amount:            int64(item.receivedQty) * item.unitCost,
					PaidAt:            paidTime,
					ProofPath:         "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf",
					Bank:              item.bank,
				}
				paymentRec.AuditFields.CreatedAt = paidTime
				paymentRec.AuditFields.UpdatedAt = paidTime
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

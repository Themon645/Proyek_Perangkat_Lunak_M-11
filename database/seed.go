package database

import (
	"fmt"
	"time"

	"m11/common"
	"m11/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type itemSeed struct {
	id          string
	name        string
	qty         int
	receivedQty int
	unitCost    int64
}

type poSeed struct {
	supplier        string
	items           []itemSeed
	receiptNo       string
	status          string
	payment         string
	hasReceipt      bool
	orderDateStr    string
	receivedDateStr string
	paidDateStr     string
	bank            string
}

func SeedDummy(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.PurchaseOrder{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		data := []poSeed{
			// 1. PO-2026-001: Disetujui (Approved) - Siap dicatat GRN (Multi-Item: 3 barang)
			{
				supplier: "PT Graha Sarana Mitra",
				items: []itemSeed{
					{id: "ITEM-101", name: "Kertas HVS A4 80gr Sinar Dunia", qty: 100, receivedQty: 0, unitCost: 65000},
					{id: "ITEM-101-B", name: "Map Odner Bantex Folio 7cm", qty: 50, receivedQty: 0, unitCost: 35000},
					{id: "ITEM-101-C", name: "Ballpoint Standard AE7 Hitam (Box 12)", qty: 20, receivedQty: 0, unitCost: 32000},
				},
				receiptNo:       "",
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 08:30:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 2. PO-2026-002: Selesai & Lunas (Received & Paid) (Multi-Item: 2 barang)
			{
				supplier: "CV Mandiri Sukses Perkasa",
				items: []itemSeed{
					{id: "ITEM-102", name: "Tinta Printer Epson L3110 Black Original", qty: 50, receivedQty: 50, unitCost: 185000},
					{id: "ITEM-102-B", name: "Tinta Printer Epson L3110 Cyan/Magenta/Yellow", qty: 30, receivedQty: 30, unitCost: 185000},
				},
				receiptNo:       "SJ-2026/X/9815",
				status:          common.StatusReceived,
				payment:         common.PaymentPaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 09:00:00",
				receivedDateStr: "2026-11-12 10:15:00",
				paidDateStr:     "2026-11-13 14:00:00",
				bank:            "Bank Mandiri (1310-0982-4412 a.n. Vendor)",
			},
			// 3. PO-2026-003: Diterima Penuh tapi Belum Bayar (Received & Unpaid) - Tagihan Belum Lunas (Multi-Item: 2 barang)
			{
				supplier: "PT Pasundan Mega Elektrik",
				items: []itemSeed{
					{id: "ITEM-103", name: "Kabel LAN Cat6 305 Meter Belden", qty: 20, receivedQty: 20, unitCost: 1250000},
					{id: "ITEM-103-B", name: "Konektor RJ45 Cat6 Passthrough (Box 100)", qty: 10, receivedQty: 10, unitCost: 175000},
				},
				receiptNo:       "SJ-2026/X/9814",
				status:          common.StatusReceived,
				payment:         common.PaymentUnpaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 09:30:00",
				receivedDateStr: "2026-11-12 11:00:00",
				paidDateStr:     "",
				bank:            "",
			},
			// 4. PO-2026-004: Disetujui (Approved) - Siap dicatat GRN (Multi-Item: 2 barang)
			{
				supplier: "CV Cipta Stationery Sentosa",
				items: []itemSeed{
					{id: "ITEM-104", name: "Stopmap Folio Karton Tebal", qty: 100, receivedQty: 0, unitCost: 12000},
					{id: "ITEM-104-B", name: "Amplop Coklat Folio Tali (Pak 100)", qty: 50, receivedQty: 0, unitCost: 45000},
				},
				receiptNo:       "",
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 10:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 5. PO-2026-005: Selesai & Lunas (Received & Paid) (Multi-Item: 2 barang)
			{
				supplier: "PT Pasundan Niaga Logistik",
				items: []itemSeed{
					{id: "ITEM-105", name: "Kursi Kantor Ergonomis Staff", qty: 15, receivedQty: 15, unitCost: 1850000},
					{id: "ITEM-105-B", name: "Footrest Ergonomis Kaki Kantor", qty: 15, receivedQty: 15, unitCost: 250000},
				},
				receiptNo:       "SJ-2026/X/9816",
				status:          common.StatusReceived,
				payment:         common.PaymentPaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 10:30:00",
				receivedDateStr: "2026-11-12 13:00:00",
				paidDateStr:     "2026-11-13 15:30:00",
				bank:            "Bank Mandiri (1310-0982-4412 a.n. Vendor)",
			},
			// 6. PO-2026-006: Disetujui (Approved) - Siap dicatat GRN (Multi-Item: 2 barang)
			{
				supplier: "PT Sinar IT Solusindo",
				items: []itemSeed{
					{id: "ITEM-106", name: "SSD NVMe 1TB Kingston KC3000", qty: 12, receivedQty: 0, unitCost: 1450000},
					{id: "ITEM-106-B", name: "RAM DDR4 16GB PC3200 SODIMM", qty: 12, receivedQty: 0, unitCost: 650000},
				},
				receiptNo:       "",
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 11:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 7. PO-2026-007: Diterima Sebagian (Received Partial & Unpaid) - (Multi-Item: 2 barang)
			{
				supplier: "CV Prima Abadi Furniture",
				items: []itemSeed{
					{id: "ITEM-107", name: "Meja Kerja Modular 120 cm", qty: 10, receivedQty: 5, unitCost: 2350000},
					{id: "ITEM-107-B", name: "Mobile Drawer 3 Laci Kayu", qty: 10, receivedQty: 4, unitCost: 850000},
				},
				receiptNo:       "SJ-2026/X/9818",
				status:          common.StatusReceivedPartial,
				payment:         common.PaymentUnpaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 11:30:00",
				receivedDateStr: "2026-11-11 14:00:00",
				paidDateStr:     "",
				bank:            "",
			},
			// 8. PO-2026-008: Selesai & Lunas (Received & Paid) (Multi-Item: 2 barang)
			{
				supplier: "PT Pasundan Cipta Grafika",
				items: []itemSeed{
					{id: "ITEM-108", name: "Toner Printer HP 136A Original", qty: 25, receivedQty: 25, unitCost: 980000},
					{id: "ITEM-108-B", name: "Drum Unit HP Laserjet Imaging", qty: 10, receivedQty: 10, unitCost: 620000},
				},
				receiptNo:       "SJ-2026/X/9819",
				status:          common.StatusReceived,
				payment:         common.PaymentPaid,
				hasReceipt:      true,
				orderDateStr:    "2026-11-10 13:00:00",
				receivedDateStr: "2026-11-12 15:00:00",
				paidDateStr:     "2026-11-13 16:00:00",
				bank:            "Bank BNI (0882-9912-331 a.n. Vendor)",
			},
			// 9. PO-2026-009: Baru Diajukan (Submitted / Pending) - (Multi-Item: 2 barang)
			{
				supplier: "CV Citra Makmur Sarana",
				items: []itemSeed{
					{id: "ITEM-109", name: "Lampu LED Panel 36 Watt Philips", qty: 40, receivedQty: 0, unitCost: 175000},
					{id: "ITEM-109-B", name: "Saklar Ganda Panasonic + Inbow Doos", qty: 30, receivedQty: 0, unitCost: 45000},
				},
				receiptNo:       "",
				status:          common.StatusPending,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 14:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
			// 10. PO-2026-010: Disetujui (Approved) - Siap dicatat GRN (Multi-Item: 2 barang)
			{
				supplier: "PT Global Network Prima",
				items: []itemSeed{
					{id: "ITEM-110", name: "Access Point TP-Link EAP225 Gigabit", qty: 8, receivedQty: 0, unitCost: 1125000},
					{id: "ITEM-110-B", name: "Switch Gigabit 16-Port TP-Link TL-SG1016D", qty: 4, receivedQty: 0, unitCost: 950000},
				},
				receiptNo:       "",
				status:          common.StatusApproved,
				payment:         common.PaymentUnpaid,
				hasReceipt:      false,
				orderDateStr:    "2026-11-10 15:00:00",
				receivedDateStr: "",
				paidDateStr:     "",
				bank:            "",
			},
		}

		for i, itemData := range data {
			poID := fmt.Sprintf("PO-2026-%03d", i+1)
			poUUID := uuid.New()
			orderTime, _ := time.Parse("2006-01-02 15:04:05", itemData.orderDateStr)

			po := model.PurchaseOrder{
				ID:       poID,
				Supplier: itemData.supplier,
				Status:   itemData.status,
				Payment:  itemData.payment,
				Items:    make([]model.PurchaseOrderItem, 0, len(itemData.items)),
			}
			po.AuditFields.UUID = poUUID
			po.AuditFields.CreatedAt = orderTime
			po.AuditFields.UpdatedAt = orderTime

			if itemData.payment == common.PaymentPaid {
				po.Proof = "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf"
			}
			if itemData.receivedDateStr != "" {
				recTime, _ := time.Parse("2006-01-02 15:04:05", itemData.receivedDateStr)
				po.ReceivedAt = &recTime
			}

			for _, it := range itemData.items {
				itemUUID := uuid.New()
				poItem := model.PurchaseOrderItem{
					ID:                it.id,
					PurchaseOrderUUID: poUUID,
					Name:              it.name,
					Quantity:          it.qty,
					UnitCost:          it.unitCost,
					Received:          it.receivedQty,
				}
				poItem.AuditFields.UUID = itemUUID
				poItem.AuditFields.CreatedAt = orderTime
				poItem.AuditFields.UpdatedAt = orderTime
				po.Items = append(po.Items, poItem)
			}

			if err := tx.Create(&po).Error; err != nil {
				return err
			}

			// Buat Goods Receipt jika ada penerimaan
			if itemData.hasReceipt {
				recTime, _ := time.Parse("2006-01-02 15:04:05", itemData.receivedDateStr)
				receiptUUID := uuid.New()
				receipt := model.GoodsReceipt{
					ReceiptNo:         itemData.receiptNo,
					PurchaseOrderUUID: poUUID,
					ReceivedDate:      recTime,
					Receiver:          "Tyezar Hasan",
					Notes:             "Barang diterima dalam kondisi baik",
					Items:             make([]model.GoodsReceiptItem, 0, len(po.Items)),
				}
				receipt.AuditFields.UUID = receiptUUID
				receipt.AuditFields.CreatedAt = recTime
				receipt.AuditFields.UpdatedAt = recTime

				for _, poItem := range po.Items {
					if poItem.Received > 0 {
						grItemUUID := uuid.New()
						grItem := model.GoodsReceiptItem{
							ID:          uuid.NewString(),
							ReceiptUUID: receiptUUID,
							ItemUUID:    poItem.AuditFields.UUID,
							QtyReceived: poItem.Received,
						}
						grItem.AuditFields.UUID = grItemUUID
						grItem.AuditFields.CreatedAt = recTime
						grItem.AuditFields.UpdatedAt = recTime
						receipt.Items = append(receipt.Items, grItem)
					}
				}

				if err := tx.Create(&receipt).Error; err != nil {
					return err
				}
			}

			// Buat Payment Record jika status sudah lunas
			if itemData.payment == common.PaymentPaid {
				paidTime, _ := time.Parse("2006-01-02 15:04:05", itemData.paidDateStr)
				var totalAmount int64
				for _, poItem := range po.Items {
					totalAmount += int64(poItem.Received) * poItem.UnitCost
				}
				paymentRec := model.PaymentRecord{
					PurchaseOrderUUID: poUUID,
					Amount:            totalAmount,
					PaidAt:            paidTime,
					ProofPath:         "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf",
					Bank:              itemData.bank,
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

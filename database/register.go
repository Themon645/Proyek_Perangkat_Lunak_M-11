package database

import (
	"m11/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Register(dsn string, autoMigrate bool) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if !autoMigrate {
		return database, nil
	}
	if err := database.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(
		&model.PurchaseOrder{},
		&model.PurchaseOrderItem{},
		&model.GoodsReceipt{},
		&model.GoodsReceiptItem{},
	); err != nil {
		return nil, err
	}
	if err := database.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_goods_receipts_purchase_order_uuid'
			) THEN
				ALTER TABLE goods_receipts
					ADD CONSTRAINT fk_goods_receipts_purchase_order_uuid
					FOREIGN KEY (purchase_order_uuid) REFERENCES purchase_orders(uuid);
			END IF;
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_goods_receipt_items_item_uuid'
			) THEN
				ALTER TABLE goods_receipt_items
					ADD CONSTRAINT fk_goods_receipt_items_item_uuid
					FOREIGN KEY (item_uuid) REFERENCES purchase_order_items(uuid);
			END IF;
		END $$;
	`).Error; err != nil {
		return nil, err
	}
	return database, nil
}

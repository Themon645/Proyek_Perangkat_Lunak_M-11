package database

import (
	"os"
	"strings"
	"sync"
	"time"

	"m11/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	postgresInstance *gorm.DB
	postgresOnce     sync.Once
	postgresErr      error
)

// BuildPostgresDSN menyusun DSN PostgreSQL dari environment variables.
func BuildPostgresDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	return "host=" + os.Getenv("DB_HOST") +
		" port=" + dbPort +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" sslmode=" + sslMode
}

// GetPostgresDB mengembalikan instance singleton GORM DB dengan connection pool
// yang dioptimalkan untuk arsitektur serverless.
func GetPostgresDB(autoMigrate bool) (*gorm.DB, error) {
	postgresOnce.Do(func() {
		dsn := BuildPostgresDSN()
		db, err := Register(dsn, autoMigrate)
		if err != nil {
			postgresErr = err
			return
		}

		// Batasi connection pool sql.DB agar ramah serverless
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.SetMaxOpenConns(10)
			sqlDB.SetMaxIdleConns(2)
			sqlDB.SetConnMaxLifetime(5 * time.Minute)
		}

		postgresInstance = db
	})

	if postgresErr != nil {
		postgresOnce = sync.Once{}
		return nil, postgresErr
	}

	return postgresInstance, nil
}

func Register(dsn string, autoMigrate bool) (*gorm.DB, error) {
	logLevel := logger.Warn
	switch strings.ToLower(os.Getenv("DB_LOG_LEVEL")) {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "info":
		logLevel = logger.Info
	case "warn":
		logLevel = logger.Warn
	}

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
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
		&model.PaymentRecord{},
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
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_payment_records_purchase_order_uuid'
			) THEN
				ALTER TABLE payment_records
					ADD CONSTRAINT fk_payment_records_purchase_order_uuid
					FOREIGN KEY (purchase_order_uuid) REFERENCES purchase_orders(uuid);
			END IF;
		END $$;
	`).Error; err != nil {
		return nil, err
	}
	return database, nil
}

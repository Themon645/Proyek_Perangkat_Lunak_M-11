package database_test

import (
	"testing"
	"time"

	"m11/database"
	"m11/model"

	"github.com/joho/godotenv"
)

func TestNeonLiveConnection(t *testing.T) {
	_ = godotenv.Load("../.env")

	db, err := database.GetPostgresDB(true)
	if err != nil {
		t.Skipf("Skipping live Neon test (network unreachable in sandbox): %v", err)
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Gagal mendapatkan sql.DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping ke Neon PostgreSQL gagal: %v", err)
	}

	t.Log("Koneksi ke Neon PostgreSQL BERHASIL!")

	// Jalankan SeedDummy jika tabel masih kosong
	if err := database.SeedDummy(db); err != nil {
		t.Fatalf("Seeding data dummy gagal: %v", err)
	}

	var poCount int64
	if err := db.Model(&model.PurchaseOrder{}).Count(&poCount).Error; err != nil {
		t.Fatalf("Gagal menghitung purchase orders: %v", err)
	}
	t.Logf("Total Purchase Orders di Neon: %d", poCount)

	var receiptCount int64
	if err := db.Model(&model.GoodsReceipt{}).Count(&receiptCount).Error; err != nil {
		t.Fatalf("Gagal menghitung goods receipts: %v", err)
	}
	t.Logf("Total Goods Receipts di Neon: %d", receiptCount)

	time.Sleep(100 * time.Millisecond)
}

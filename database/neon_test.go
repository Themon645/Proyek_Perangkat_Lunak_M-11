package database_test

import (
	"testing"
	"time"

	"github.com/joho/godotenv"
	"m11/common"
	"m11/database"
	"m11/model"
)

func TestNeonResetAndSeed(t *testing.T) {
	_ = godotenv.Load("../.env")

	db, err := database.GetPostgresDB(true)
	if err != nil {
		t.Skipf("Skipping live Neon test (network unreachable in sandbox): %v", err)
		return
	}

	// 1. Reset and re-seed with realistic statuses
	if err := database.ResetAndSeedPostgres(db); err != nil {
		t.Fatalf("Gagal reset and seed Neon: %v", err)
	}

	// 2. Count Approved POs
	var approvedCount int64
	if err := db.Model(&model.PurchaseOrder{}).Where("status = ?", common.StatusApproved).Count(&approvedCount).Error; err != nil {
		t.Fatalf("Gagal menghitung approved PO: %v", err)
	}
	t.Logf("Total PO berstatus 'approved' di Neon: %d", approvedCount)

	// 3. Count Total POs
	var totalCount int64
	if err := db.Model(&model.PurchaseOrder{}).Count(&totalCount).Error; err != nil {
		t.Fatalf("Gagal menghitung total PO: %v", err)
	}
	t.Logf("Total seluruh PO di Neon: %d", totalCount)

	time.Sleep(100 * time.Millisecond)
}

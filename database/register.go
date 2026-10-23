package database

import (
	"m11/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Register(path string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(&model.PurchaseOrder{}, &model.PurchaseOrderItem{}); err != nil {
		return nil, err
	}
	return database, nil
}

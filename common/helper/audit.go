package helper

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditFields struct {
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-"`
	UUID      uuid.UUID      `json:"uuid" gorm:"type:uuid;default:gen_random_uuid();uniqueIndex;not null"`
}

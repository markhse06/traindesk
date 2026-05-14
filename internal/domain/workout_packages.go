package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkoutPackage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	TrainerID uuid.UUID `gorm:"index"`
	ClientID  uuid.UUID `gorm:"index"`

	TotalCount int `gorm:"not null"`
	UsedCount  int `gorm:"default:0"` // Использовано

	IsActive bool    `gorm:"default:true"`
	Price    float64 `gorm:"type:decimal(10,2)"`

	CreatedAt time.Time
}

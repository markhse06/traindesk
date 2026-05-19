package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkoutPackage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TrainerID uuid.UUID `gorm:"index" json:"trainer_id"`
	ClientID  uuid.UUID `gorm:"index" json:"client_id"`

	TotalCount int `gorm:"not null" json:"total_count"`
	UsedCount  int `gorm:"default:0" json:"used_count"`

	IsActive bool    `gorm:"default:true" json:"is_active"`
	Price    float64 `gorm:"type:decimal(10,2)" json:"price"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

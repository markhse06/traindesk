package domain

import (
	"time"

	"github.com/google/uuid"
)

// Client — сущность клиента тренера.
type Client struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	FirstName string    `gorm:"not null"`
	LastName  string    `gorm:"not null"`

	Workouts []Workout `gorm:"many2many:workout_clients"`

	Height float64 `json:"height"`
	Weight float64 `json:"weight"`
	Goal   string  `gorm:"type:text" json:"goal"`

	TotalSessions int `gorm:"default:0" json:"total_sessions"`
	LeftSessions  int `gorm:"default:0" json:"left_sessions"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
}

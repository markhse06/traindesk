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

	Workouts        []Workout        `gorm:"many2many:workout_clients"`
	WorkoutPackages []WorkoutPackage `gorm:"foreignKey:ClientID;references:ID"`

	Height float64 `gorm:"type:decimal(6,2);default:0" json:"height"`
	Weight float64 `gorm:"type:decimal(6,2);default:0" json:"weight"`
	Goal   string  `gorm:"type:text;default:''" json:"goal"`

	TotalSessions int `gorm:"default:0" json:"total_sessions"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
}

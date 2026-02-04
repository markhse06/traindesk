package user

import (
	"time"

	"github.com/google/uuid"
)

// User — сущность тренера в БД.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	TrainerName  string    `gorm:"not null"`

	EmailVerified bool `gorm:"not null;default:false"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type EmailVerification struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	UserID    uuid.UUID `gorm:"index"`
	Code      string    `gorm:"size:6;index"`
	ExpiresAt time.Time
}

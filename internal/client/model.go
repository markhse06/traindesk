package client

import (
	"time"

	"github.com/google/uuid"
)

// Client — сущность клиента тренера.
type Client struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	FirstName string `gorm:"not null"`
	LastName  string `gorm:"not null"`
	// TODO: phone/email/notes при необходимости.

	CreatedAt time.Time
	UpdatedAt time.Time
}

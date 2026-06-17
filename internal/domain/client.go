package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
)

func ValidateAndFormatPhone(inputPhone string) (string, error) {
	// "RU" — регион по умолчанию, если пользователь ввел номер без "+" (например, "8999...")
	parsedNumber, err := phonenumbers.Parse(inputPhone, "RU")
	if err != nil {
		return "", err
	}

	// Проверяем, является ли номер теоретически возможным и валидным
	if !phonenumbers.IsValidNumber(parsedNumber) {
		return "", fmt.Errorf("invalid phone number structure")
	}

	// Форматируем в международный стандарт E.164 (всегда начинается с +)
	formatted := phonenumbers.Format(parsedNumber, phonenumbers.E164)
	return formatted, nil
}

// Client — сущность клиента тренера.
type Client struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	FirstName string    `gorm:"not null"`
	LastName  string    `gorm:"not null"`

	Workouts        []Workout        `gorm:"many2many:workout_clients"`
	WorkoutPackages []WorkoutPackage `gorm:"foreignKey:ClientID;references:ID"`

	Phone  string  `gorm:"type:varchar(20);not null"`
	Height float64 `gorm:"type:decimal(6,2);default:0" json:"height"`
	Weight float64 `gorm:"type:decimal(6,2);default:0" json:"weight"`
	Goal   string  `gorm:"type:text;default:''" json:"goal"`

	TotalSessions int `gorm:"default:0" json:"total_sessions"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
}

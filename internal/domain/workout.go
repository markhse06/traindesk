package domain

import (
	"time"

	"github.com/google/uuid"
)

// WorkoutType — доменный тип тренировки.
type WorkoutType string

const (
	WorkoutTypeCardio     WorkoutType = "cardio"
	WorkoutTypeStrength   WorkoutType = "strength"
	WorkoutTypeStretch    WorkoutType = "stretch"
	WorkoutTypeFunctional WorkoutType = "functional"
	// TODO: в будущем вынести виды тренировок в отдельную сущность / таблицу.
)

// ValidWorkoutTypes — список допустимых типов тренировок.
var ValidWorkoutTypes = []WorkoutType{
	WorkoutTypeCardio,
	WorkoutTypeStrength,
	WorkoutTypeStretch,
	WorkoutTypeFunctional,
}

// IsValidType проверяет, что строка — один из известных типов.
func IsValidType(t string) bool {
	wt := WorkoutType(t)
	for _, v := range ValidWorkoutTypes {
		if wt == v {
			return true
		}
	}
	return false
}

// Workout — сущность тренировки в БД.
type Workout struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	Clients   []Client  `gorm:"many2many:workout_clients;constraint:OnDelete:CASCADE"`
	PackageID uuid.UUID `gorm:"foreignKey:ID;reference:ID;constraint:OnDelete:CASCADE"`

	DateTime    time.Time   `gorm:"type:timestamptz;not null"`
	DurationMin int         `gorm:"not null"`
	Type        WorkoutType `gorm:"type:varchar(32);not null"`
	Notes       string      `gorm:"type:text"`
	Price       float64     `gorm:"type:decimal(10,2);not null;default:0" json:"price"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
}

// WorkoutClient — связь многие-ко-многим между тренировками и клиентами.
type WorkoutClient struct {
	WorkoutID uuid.UUID `gorm:"type:uuid;primaryKey"`
	ClientID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}

package workout

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

	Date        time.Time   `gorm:"not null"`
	DurationMin int         `gorm:"not null"`
	Type        WorkoutType `gorm:"type:varchar(32);not null"`
	Notes       string      `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// WorkoutClient — связь многие-ко-многим между тренировками и клиентами.
type WorkoutClient struct {
	WorkoutID uuid.UUID `gorm:"type:uuid;primaryKey"`
	ClientID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}

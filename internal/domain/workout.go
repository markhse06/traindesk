package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkoutType string

const (
	WorkoutTypeCardio     WorkoutType = "cardio"
	WorkoutTypeStrength   WorkoutType = "strength"
	WorkoutTypeStretch    WorkoutType = "stretch"
	WorkoutTypeFunctional WorkoutType = "functional"
)

var ValidWorkoutTypes = []WorkoutType{
	WorkoutTypeCardio,
	WorkoutTypeStrength,
	WorkoutTypeStretch,
	WorkoutTypeFunctional,
}

func IsValidType(t string) bool {
	wt := WorkoutType(t)
	for _, v := range ValidWorkoutTypes {
		if wt == v {
			return true
		}
	}
	return false
}

type WorkoutStatus string

const (
	WorkoutStatusPlanned   WorkoutStatus = "planned"
	WorkoutStatusCompleted WorkoutStatus = "completed"
	WorkoutStatusCancelled WorkoutStatus = "cancelled"
)

var ValidWorkoutStatuses = []WorkoutStatus{
	WorkoutStatusPlanned,
	WorkoutStatusCompleted,
	WorkoutStatusCancelled,
}

func IsValidStatus(status string) bool {
	ws := WorkoutStatus(status)
	for _, v := range ValidWorkoutStatuses {
		if ws == v {
			return true
		}
	}
	return false
}

type Workout struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	Clients   []Client  `gorm:"many2many:workout_clients;constraint:OnDelete:CASCADE"`
	PackageID uuid.UUID `gorm:"foreignKey:ID;reference:ID;constraint:OnDelete:CASCADE"`

	DateTime    time.Time     `gorm:"type:timestamptz;not null"`
	DurationMin int           `gorm:"not null"`
	Type        WorkoutType   `gorm:"type:varchar(32);not null"`
	Status      WorkoutStatus `gorm:"type:varchar(32);not null;default:'planned'"`
	Notes       string        `gorm:"type:text"`
	Price       float64       `gorm:"type:decimal(10,2);not null;default:0" json:"price"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now();not null"`
}

type WorkoutClient struct {
	WorkoutID uuid.UUID `gorm:"type:uuid;primaryKey"`
	ClientID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}

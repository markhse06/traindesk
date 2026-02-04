package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"traindesk/internal/client"
	"traindesk/internal/user"
	"traindesk/internal/workout"
)

type DB struct {
	*gorm.DB
}

func NewDB() (*DB, error) {
	host := "localhost"
	port := 5432
	userDB := "postgres"
	password := "postgres"
	dbname := "traindesk"

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, userDB, password, dbname,
	)

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("connected to postgres")

	if err := autoMigrate(gormDB); err != nil {
		return nil, err
	}

	return &DB{gormDB}, nil
}

func autoMigrate(gormDB *gorm.DB) error {
	return gormDB.AutoMigrate(
		&user.User{},
		&client.Client{},
		&workout.Workout{},
		&workout.WorkoutClient{},
	)
}

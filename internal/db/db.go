package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"traindesk/internal/client"
	"traindesk/internal/config"
	"traindesk/internal/user"
	"traindesk/internal/workout"
)

var cfg = config.Load()

type DB struct {
	*gorm.DB
}

func NewDB() (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	log.Println("DB CONFIG:",
		"host=", cfg.DBHost,
		"port=", cfg.DBPort,
		"user=", cfg.DBUser,
		"db=", cfg.DBName,
	)

	log.Println("DSN:", dsn)

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
		&user.EmailVerification{},
	)
}

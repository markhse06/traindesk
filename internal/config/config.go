package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string
	HTTPPort  string
}

type SMTPConfig struct {
	API    string
	Sender string
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		HTTPPort:   os.Getenv("HTTP_PORT"),
	}

	if cfg.JWTSecret == "dev-secret-key" {
		log.Println("WARN: JWT_SECRET не задан, используется dev-secret-key")
	}

	return cfg
}

func LoadSMTP() SMTPConfig {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	SMTPcfg := SMTPConfig{
		API:    os.Getenv("SMTP_API"),
		Sender: os.Getenv("EMAIL_SENDER"),
	}

	return SMTPcfg
}

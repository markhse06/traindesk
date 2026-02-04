package config

import (
	"log"
	"os"
)

type Config struct {
	JWTSecret string
	// TODO: добавить сюда параметры БД, SMTP и т.п.
}

func Load() Config {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("WARN: JWT_SECRET is empty, using dev default")
		secret = "dev-secret-key"
	}

	return Config{
		JWTSecret: secret,
	}
}

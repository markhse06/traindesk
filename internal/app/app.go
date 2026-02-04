package app

import (
	"traindesk/internal/config"

	"github.com/gin-gonic/gin"

	"traindesk/internal/db"
	"traindesk/internal/email"
)

type App struct {
	router *gin.Engine
	db     *db.DB
	mailer *email.Sender
}

func NewApp() (*App, error) {
	r := gin.Default()

	// TODO: заменить все глобальные переменные и заглушки на зщначения из .env
	_ = config.Load()

	database, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	mailer := email.NewSender(email.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "your_smtp_user",
		Password: "your_smtp_password",
		From:     "no-reply@traindesk.app",
	})

	a := &App{
		router: r,
		db:     database,
		mailer: mailer,
	}

	a.registerRoutes()

	return a, nil
}

func (a *App) Run() error {
	// TODO: порт читать из конфигурации/переменных окружения.
	return a.router.Run(":8080")
}

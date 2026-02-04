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

	database, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	mailer := email.NewSender()

	a := &App{
		router: r,
		db:     database,
		mailer: mailer,
	}

	a.registerRoutes()

	return a, nil
}

func (a *App) Run() error {
	cfg := config.Load()
	return a.router.Run(":" + cfg.HTTPPort)
}

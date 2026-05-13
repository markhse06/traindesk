package main

import (
	"log"

	"traindesk/internal/app"

	_ "traindesk/docs"
)

// @title TrainDesk API
// @version 1.0
// @description REST API for trainer account, client, and workout management.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}

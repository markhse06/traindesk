package main

import (
	"log"

	"traindesk/internal/app"
)

func main() {
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}

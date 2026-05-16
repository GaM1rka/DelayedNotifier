package main

import (
	"os"

	"sender-service/config"
	"sender-service/internal/app"
	"sender-service/internal/logger"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	application := app.New(cfg, log)
	if err := application.Run(); err != nil {
		log.Error("failed to run server", "error", err)
		os.Exit(1)
	}
}

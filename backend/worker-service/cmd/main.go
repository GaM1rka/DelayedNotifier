package main

import (
	"os"

	"worker-service/config"
	"worker-service/internal/app"
	"worker-service/internal/logger"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	application, err := app.New(cfg, log)
	if err != nil {
		log.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	if err := application.Run(); err != nil {
		log.Error("failed to run app", "error", err)
		os.Exit(1)
	}
}

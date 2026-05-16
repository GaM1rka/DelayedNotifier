package main

import (
	"log/slog"
	"os"

	"api-service/config"
	"api-service/internal/app"
	"api-service/internal/logger"
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

	slog.Info("api-service stopped")
}

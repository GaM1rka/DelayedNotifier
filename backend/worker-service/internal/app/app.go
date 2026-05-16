package app

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"worker-service/config"
	"worker-service/internal/queue"
	"worker-service/internal/repository"
	"worker-service/internal/sender"
)

type App struct {
	repo   *repository.Repository
	worker *queue.Worker
}

func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()

	repo, err := retry(ctx, logger, "postgres", func() (*repository.Repository, error) {
		return repository.New(ctx, cfg.Postgres.DSN)
	})
	if err != nil {
		return nil, err
	}

	senderClient := sender.New(cfg.Sender.URL)
	worker, err := retry(ctx, logger, "rabbitmq", func() (*queue.Worker, error) {
		return queue.NewWorker(cfg.RabbitMQ.URL, cfg.RabbitMQ.Queue, repo, senderClient, cfg.Retry, logger)
	})
	if err != nil {
		repo.Close()
		return nil, err
	}

	return &App{repo: repo, worker: worker}, nil
}

func retry[T any](ctx context.Context, logger *slog.Logger, name string, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	for attempt := 1; attempt <= 12; attempt++ {
		value, err := fn()
		if err == nil {
			return value, nil
		}
		lastErr = err
		logger.Warn("dependency is not ready", "dependency", name, "attempt", attempt, "error", err)

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return zero, lastErr
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer a.repo.Close()
	defer a.worker.Close()

	return a.worker.Run(ctx)
}

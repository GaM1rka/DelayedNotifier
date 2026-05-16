package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"api-service/config"
	"api-service/internal/cache"
	"api-service/internal/handler"
	"api-service/internal/queue"
	"api-service/internal/repository"
)

type App struct {
	cfg    *config.Config
	logger *slog.Logger
	server *http.Server
	repo   *repository.Repository
	cache  *cache.Cache
	pub    *queue.Publisher
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

	redisCache := cache.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.TTL)
	if err := retryErr(ctx, logger, "redis", func() error {
		return redisCache.Ping(ctx)
	}); err != nil {
		repo.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	publisher, err := retry(ctx, logger, "rabbitmq", func() (*queue.Publisher, error) {
		return queue.NewPublisher(cfg.RabbitMQ.URL, cfg.RabbitMQ.Queue)
	})
	if err != nil {
		repo.Close()
		_ = redisCache.Close()
		return nil, err
	}

	h := handler.New(repo, redisCache, publisher, logger)
	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{cfg: cfg, logger: logger, server: server, repo: repo, cache: redisCache, pub: publisher}, nil
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

func retryErr(ctx context.Context, logger *slog.Logger, name string, fn func() error) error {
	_, err := retry(ctx, logger, name, func() (struct{}, error) {
		return struct{}{}, fn()
	})

	return err
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("api-service started", "addr", a.cfg.HTTP.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	_ = a.pub.Close()
	_ = a.cache.Close()
	a.repo.Close()

	return nil
}

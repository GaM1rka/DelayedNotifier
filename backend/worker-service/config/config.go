package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultEnvPath = ".env"

type Config struct {
	Postgres PostgresConfig
	RabbitMQ RabbitMQConfig
	Sender   SenderConfig
	Retry    RetryConfig
}

type PostgresConfig struct {
	DSN string
}

type RabbitMQConfig struct {
	URL   string
	Queue string
}

type SenderConfig struct {
	URL string
}

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func Load() (*Config, error) {
	if err := loadDotEnv(defaultEnvPath); err != nil {
		return nil, err
	}

	cfg := &Config{
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", "postgres://delayed:delayed@postgres:5432/delayed_notifier?sslmode=disable"),
		},
		RabbitMQ: RabbitMQConfig{
			URL:   getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
			Queue: getEnv("RABBITMQ_QUEUE", "notifications.delayed"),
		},
		Sender: SenderConfig{
			URL: getEnv("SENDER_URL", "http://sender-service:8004/send/email"),
		},
		Retry: RetryConfig{
			MaxAttempts: getIntEnv("MAX_ATTEMPTS", 5),
			BaseDelay:   getDurationEnv("RETRY_BASE_DELAY", 10*time.Second),
			MaxDelay:    getDurationEnv("RETRY_MAX_DELAY", 15*time.Minute),
		},
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.Postgres.DSN == "" {
		return errors.New("POSTGRES_DSN is required")
	}
	if c.RabbitMQ.URL == "" {
		return errors.New("RABBITMQ_URL is required")
	}
	if c.RabbitMQ.Queue == "" {
		return errors.New("RABBITMQ_QUEUE is required")
	}
	if c.Sender.URL == "" {
		return errors.New("SENDER_URL is required")
	}
	if c.Retry.MaxAttempts < 1 {
		return errors.New("MAX_ATTEMPTS must be greater than zero")
	}

	return nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		value = cleanEnvValue(value)
		if key == "" {
			return fmt.Errorf("%s:%d: empty env key", path, lineNumber)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s:%d: set %s: %w", path, lineNumber, key, err)
		}
	}

	return scanner.Err()
}

func cleanEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '\'' || quote == '"') && value[len(value)-1] == quote {
			return value[1 : len(value)-1]
		}
	}
	if before, _, ok := strings.Cut(value, " #"); ok {
		return strings.TrimSpace(before)
	}

	return value
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

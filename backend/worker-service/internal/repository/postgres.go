package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"worker-service/internal/model"
)

var ErrNotFound = errors.New("notification not found")

type Repository struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) Get(ctx context.Context, id string) (model.Notification, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, email, subject, message, send_at, status, attempts, last_error
FROM notifications
WHERE id = $1
`, id)

	var n model.Notification
	if err := row.Scan(&n.ID, &n.Email, &n.Subject, &n.Message, &n.SendAt, &n.Status, &n.Attempts, &n.LastError); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Notification{}, ErrNotFound
		}
		return model.Notification{}, fmt.Errorf("scan notification: %w", err)
	}

	return n, nil
}

func (r *Repository) MarkSending(ctx context.Context, id string, attempt int) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
UPDATE notifications
SET status = $2, attempts = $3, updated_at = $4, last_error = ''
WHERE id = $1 AND status IN ($5, $6)
`, id, model.StatusSending, attempt, time.Now().UTC(), model.StatusScheduled, model.StatusFailed)
	if err != nil {
		return false, fmt.Errorf("mark sending: %w", err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *Repository) MarkSent(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
UPDATE notifications
SET status = $2, updated_at = $3, completed_at = $3
WHERE id = $1
`, id, model.StatusSent, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}

	return nil
}

func (r *Repository) MarkFailed(ctx context.Context, id string, attempts int, message string) error {
	_, err := r.pool.Exec(ctx, `
UPDATE notifications
SET status = $2, attempts = $3, last_error = $4, updated_at = $5
WHERE id = $1
`, id, model.StatusFailed, attempts, message, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	return nil
}

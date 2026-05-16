package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"api-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	repo := &Repository{pool: pool}
	if err := repo.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return repo, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) migrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS notifications (
	id UUID PRIMARY KEY,
	email TEXT NOT NULL,
	subject TEXT NOT NULL,
	message TEXT NOT NULL,
	send_at TIMESTAMPTZ NOT NULL,
	status TEXT NOT NULL,
	attempts INT NOT NULL DEFAULT 0,
	last_error TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	completed_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS notifications_status_send_at_idx ON notifications (status, send_at);
`)
	if err != nil {
		return fmt.Errorf("migrate postgres: %w", err)
	}

	return nil
}

func (r *Repository) Create(ctx context.Context, n model.Notification) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO notifications (id, email, subject, message, send_at, status, attempts, last_error, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`, n.ID, n.Email, n.Subject, n.Message, n.SendAt, n.Status, n.Attempts, n.LastError, n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, id string) (model.Notification, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, email, subject, message, send_at, status, attempts, last_error, created_at, updated_at, COALESCE(completed_at, '0001-01-01'::timestamptz)
FROM notifications
WHERE id = $1
`, id)

	return scanNotification(row)
}

func (r *Repository) List(ctx context.Context) ([]model.Notification, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, email, subject, message, send_at, status, attempts, last_error, created_at, updated_at, COALESCE(completed_at, '0001-01-01'::timestamptz)
FROM notifications
ORDER BY created_at DESC
LIMIT 100
`)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	items := make([]model.Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return items, nil
}

func (r *Repository) Cancel(ctx context.Context, id string) (model.Notification, error) {
	row := r.pool.QueryRow(ctx, `
UPDATE notifications
SET status = $2, updated_at = $3
WHERE id = $1 AND status IN ($4, $5, $6)
RETURNING id::text, email, subject, message, send_at, status, attempts, last_error, created_at, updated_at, COALESCE(completed_at, '0001-01-01'::timestamptz)
`, id, model.StatusCanceled, time.Now().UTC(), model.StatusScheduled, model.StatusFailed, model.StatusSending)

	return scanNotification(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanNotification(row scanner) (model.Notification, error) {
	var n model.Notification
	if err := row.Scan(
		&n.ID,
		&n.Email,
		&n.Subject,
		&n.Message,
		&n.SendAt,
		&n.Status,
		&n.Attempts,
		&n.LastError,
		&n.CreatedAt,
		&n.UpdatedAt,
		&n.CompletedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Notification{}, ErrNotFound
		}
		return model.Notification{}, fmt.Errorf("scan notification: %w", err)
	}

	return n, nil
}

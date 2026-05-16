package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"worker-service/config"
	"worker-service/internal/model"
	"worker-service/internal/repository"
	"worker-service/internal/sender"
)

type Worker struct {
	conn       *amqp.Connection
	ch         *amqp.Channel
	queue      string
	repo       *repository.Repository
	sender     *sender.Client
	retry      config.RetryConfig
	logger     *slog.Logger
	deliveries <-chan amqp.Delivery
}

func NewWorker(url, queueName string, repo *repository.Repository, senderClient *sender.Client, retry config.RetryConfig, logger *slog.Logger) (*Worker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("consume queue: %w", err)
	}

	return &Worker{conn: conn, ch: ch, queue: queueName, repo: repo, sender: senderClient, retry: retry, logger: logger, deliveries: deliveries}, nil
}

func (w *Worker) Close() error {
	if err := w.ch.Close(); err != nil {
		_ = w.conn.Close()
		return err
	}

	return w.conn.Close()
}

func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker started", "queue", w.queue)
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-w.deliveries:
			if !ok {
				return nil
			}
			w.handleDelivery(ctx, delivery)
		}
	}
}

func (w *Worker) handleDelivery(ctx context.Context, delivery amqp.Delivery) {
	var msg model.QueueMessage
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		w.logger.Error("invalid queue message", "error", err)
		_ = delivery.Ack(false)
		return
	}

	if err := waitUntil(ctx, msg.SendAt); err != nil {
		_ = delivery.Nack(false, true)
		return
	}

	notification, err := w.repo.Get(ctx, msg.ID)
	if err != nil {
		w.logger.Error("get notification", "id", msg.ID, "error", err)
		_ = delivery.Ack(false)
		return
	}
	if notification.Status == model.StatusCanceled || notification.Status == model.StatusSent {
		_ = delivery.Ack(false)
		return
	}

	attempt := max(msg.Attempt+1, notification.Attempts+1)
	if ok, err := w.repo.MarkSending(ctx, notification.ID, attempt); err != nil || !ok {
		if err != nil {
			w.logger.Error("mark sending", "id", notification.ID, "error", err)
		}
		_ = delivery.Ack(false)
		return
	}

	err = w.sender.SendEmail(ctx, model.SendEmailRequest{
		Email:   notification.Email,
		Subject: notification.Subject,
		Message: notification.Message,
	})
	if err == nil {
		if markErr := w.repo.MarkSent(ctx, notification.ID); markErr != nil {
			w.logger.Error("mark sent", "id", notification.ID, "error", markErr)
			_ = delivery.Nack(false, true)
			return
		}
		w.logger.Info("notification sent", "id", notification.ID, "attempt", attempt)
		_ = delivery.Ack(false)
		return
	}

	w.logger.Error("send notification failed", "id", notification.ID, "attempt", attempt, "error", err)
	if markErr := w.repo.MarkFailed(ctx, notification.ID, attempt, err.Error()); markErr != nil {
		w.logger.Error("mark failed", "id", notification.ID, "error", markErr)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)
	if attempt >= w.retry.MaxAttempts {
		w.logger.Error("notification attempts exhausted", "id", notification.ID, "attempts", attempt)
		return
	}

	next := model.QueueMessage{
		ID:      notification.ID,
		SendAt:  time.Now().UTC().Add(w.backoff(attempt)),
		Attempt: attempt,
	}
	if err := w.publishRetry(ctx, next); err != nil {
		w.logger.Error("publish retry", "id", notification.ID, "error", err)
	}
}

func (w *Worker) publishRetry(ctx context.Context, msg model.QueueMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode retry message: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = w.ch.PublishWithContext(ctx, "", w.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish retry message: %w", err)
	}

	return nil
}

func (w *Worker) backoff(attempt int) time.Duration {
	power := math.Pow(2, float64(max(attempt-1, 0)))
	delay := time.Duration(power) * w.retry.BaseDelay
	if delay > w.retry.MaxDelay {
		return w.retry.MaxDelay
	}

	return delay
}

func waitUntil(ctx context.Context, deadline time.Time) error {
	delay := time.Until(deadline)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

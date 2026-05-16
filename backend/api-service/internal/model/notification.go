package model

import "time"

const (
	StatusScheduled = "scheduled"
	StatusCanceled  = "canceled"
	StatusSending   = "sending"
	StatusSent      = "sent"
	StatusFailed    = "failed"
)

type Notification struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Subject     string    `json:"subject"`
	Message     string    `json:"message"`
	SendAt      time.Time `json:"send_at"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type CreateNotificationRequest struct {
	Email   string    `json:"email"`
	Subject string    `json:"subject"`
	Message string    `json:"message"`
	SendAt  time.Time `json:"send_at"`
}

type QueueMessage struct {
	ID      string    `json:"id"`
	SendAt  time.Time `json:"send_at"`
	Attempt int       `json:"attempt"`
}

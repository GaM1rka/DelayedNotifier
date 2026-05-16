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
	ID        string
	Email     string
	Subject   string
	Message   string
	SendAt    time.Time
	Status    string
	Attempts  int
	LastError string
}

type QueueMessage struct {
	ID      string    `json:"id"`
	SendAt  time.Time `json:"send_at"`
	Attempt int       `json:"attempt"`
}

type SendEmailRequest struct {
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

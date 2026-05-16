package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"api-service/internal/cache"
	"api-service/internal/model"
	"api-service/internal/queue"
	"api-service/internal/repository"
	"github.com/google/uuid"
)

type Handler struct {
	repo      *repository.Repository
	cache     *cache.Cache
	publisher *queue.Publisher
	logger    *slog.Logger
}

func New(repo *repository.Repository, cache *cache.Cache, publisher *queue.Publisher, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, cache: cache, publisher: publisher, logger: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /notify", h.createNotification)
	mux.HandleFunc("GET /notify", h.listNotifications)
	mux.HandleFunc("GET /notify/{id}", h.getNotification)
	mux.HandleFunc("DELETE /notify/{id}", h.cancelNotification)
	return cors(mux)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createNotification(w http.ResponseWriter, r *http.Request) {
	var req model.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)
	if err := validateCreateRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	notification := model.Notification{
		ID:        uuid.NewString(),
		Email:     req.Email,
		Subject:   req.Subject,
		Message:   req.Message,
		SendAt:    req.SendAt.UTC(),
		Status:    model.StatusScheduled,
		Attempts:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.Create(r.Context(), notification); err != nil {
		h.logger.Error("create notification", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create notification")
		return
	}
	if err := h.publisher.Publish(r.Context(), model.QueueMessage{ID: notification.ID, SendAt: notification.SendAt, Attempt: 0}); err != nil {
		h.logger.Error("publish notification", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to enqueue notification")
		return
	}
	_ = h.cache.Set(r.Context(), notification)

	writeJSON(w, http.StatusCreated, notification)
}

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		h.logger.Error("list notifications", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) getNotification(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	if cached, ok, err := h.cache.Get(r.Context(), id); err == nil && ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	notification, err := h.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error("get notification", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get notification")
		return
	}
	_ = h.cache.Set(r.Context(), notification)

	writeJSON(w, http.StatusOK, notification)
}

func (h *Handler) cancelNotification(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	notification, err := h.repo.Cancel(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "notification not found or already completed")
			return
		}
		h.logger.Error("cancel notification", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to cancel notification")
		return
	}
	_ = h.cache.Set(r.Context(), notification)

	writeJSON(w, http.StatusOK, notification)
}

func validateCreateRequest(req model.CreateNotificationRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	addr, err := mail.ParseAddress(req.Email)
	if err != nil || addr.Address != req.Email {
		return errors.New("email is invalid")
	}
	if req.Subject == "" {
		return errors.New("subject is required")
	}
	if strings.ContainsAny(req.Subject, "\r\n") {
		return errors.New("subject is invalid")
	}
	if req.Message == "" {
		return errors.New("message is required")
	}
	if req.SendAt.IsZero() {
		return errors.New("send_at is required")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

package router

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/yorukot/gitgram/internal/formatter"
	"github.com/yorukot/gitgram/internal/githubwebhook"
	"github.com/yorukot/gitgram/internal/store"
)

const defaultMaxBodyBytes = int64(10 << 20)

type TelegramSender interface {
	SendMessage(ctx context.Context, chatID string, text string) error
}

type GitHubWebhookHandler struct {
	Secret       string
	ChatID       string
	MaxBodyBytes int64
	AllowedRepo  func(repo string) bool
	Store        store.DeliveryStore
	Sender       TelegramSender
	Logger       *slog.Logger
}

func Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (h GitHubWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxBodyBytes := h.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultMaxBodyBytes
	}

	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		http.Error(w, "request body too large or unreadable", http.StatusRequestEntityTooLarge)
		return
	}

	if !githubwebhook.VerifySignature(h.Secret, body, r.Header.Get("X-Hub-Signature-256")) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		http.Error(w, "missing X-GitHub-Event", http.StatusBadRequest)
		return
	}

	deliveryID := r.Header.Get("X-GitHub-Delivery")
	if deliveryID == "" {
		http.Error(w, "missing X-GitHub-Delivery", http.StatusBadRequest)
		return
	}

	if h.Store != nil && !h.Store.Claim(deliveryID) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("duplicate delivery ignored\n"))
		return
	}

	releaseDelivery := true
	defer func() {
		if releaseDelivery && h.Store != nil {
			h.Store.Release(deliveryID)
		}
	}()

	activity, err := githubwebhook.ParseEvent(eventType, deliveryID, body)
	if errors.Is(err, githubwebhook.ErrIgnored) {
		releaseDelivery = false
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event ignored\n"))
		return
	}
	if err != nil {
		http.Error(w, "invalid github payload", http.StatusBadRequest)
		return
	}

	if h.AllowedRepo != nil && !h.AllowedRepo(activity.Repo) {
		releaseDelivery = false
		http.Error(w, "repo not allowed", http.StatusForbidden)
		return
	}

	if h.Sender == nil {
		http.Error(w, "telegram sender is not configured", http.StatusInternalServerError)
		return
	}

	message := formatter.TelegramHTML(activity)
	if err := h.Sender.SendMessage(r.Context(), h.ChatID, message); err != nil {
		if h.Logger != nil {
			h.Logger.Error("telegram send failed", "delivery_id", deliveryID, "repo", activity.Repo, "event", eventType, "error", err)
		}
		http.Error(w, "telegram send failed", http.StatusBadGateway)
		return
	}

	releaseDelivery = false
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

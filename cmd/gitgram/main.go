package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yorukot/gitgram/internal/config"
	"github.com/yorukot/gitgram/internal/router"
	"github.com/yorukot/gitgram/internal/store"
	"github.com/yorukot/gitgram/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	telegramClient := telegram.NewClient(cfg.TelegramBotToken, &http.Client{
		Timeout: 10 * time.Second,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", router.Healthz)
	mux.Handle("/webhooks/github", router.GitHubWebhookHandler{
		Secret:       cfg.GitHubWebhookSecret,
		ChatID:       cfg.TelegramChatID,
		MaxBodyBytes: cfg.MaxBodyBytes,
		AllowedRepo:  cfg.RepoAllowed,
		Store:        store.NewMemoryDeliveryStore(cfg.DeliveryCacheSize),
		Sender:       telegramClient,
		Logger:       logger,
	})

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("gitgram listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

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

	"notifications/internal/api"
	v1 "notifications/internal/api/v1"
	"notifications/internal/config"
	"notifications/internal/pkg/logger"
	"notifications/internal/service"
	"notifications/internal/sms/messagebird"
	"notifications/internal/store"
)

// @title Notifications API
// @version 1.0
// @description SMS notifications for overdue loans.
// @BasePath /v1
func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("failed to run service", "error", err)
		os.Exit(1)
	}
}

func run(log logger.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.New(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	sender := messagebird.New(cfg.MessageBird, log)

	customerRepo := store.NewCustomers(db.DB())
	templateRepo := store.NewTemplates(db.DB())
	messageRepo := store.NewMessages(db.DB())

	customers := service.NewCustomers(customerRepo)
	templates := service.NewTemplates(templateRepo)
	messages := service.NewMessages(customerRepo, templateRepo, messageRepo, sender, log)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: api.NewServer(v1.NewServer(customers, templates, messages, log), log).Handler(),
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr, "messagebird_base_url", cfg.MessageBird.BaseURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}

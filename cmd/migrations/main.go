package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"notifications/internal/config"
	"notifications/internal/pkg/logger"
	"notifications/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
}

func run(log logger.Logger) error {
	db, err := config.LoadDatabase()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, db)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		return err
	}

	log.Info("migrations applied")
	return nil
}

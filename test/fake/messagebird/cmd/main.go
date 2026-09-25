package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	fake "notifications/test/fake/messagebird"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := os.Getenv("FAKE_MESSAGEBIRD_ADDR")
	if addr == "" {
		addr = ":8099"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           fake.New(os.Getenv("MESSAGEBIRD_ACCESS_KEY")).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("fake messagebird listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to listen and serve", "error", err)
		os.Exit(1)
	}
}

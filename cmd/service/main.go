package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/a-pelikh/tripgo/internal/config"
	"github.com/a-pelikh/tripgo/internal/container"
)

func main() {
	cfg := config.MustLoad()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app, err := container.New(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to initialize service", "error", err)
		return
	}
	log := app.Logger
	httpServer := app.Server

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()
	log.InfoContext(ctx, "service started")

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed while serving http", "error", err)
		}
	}

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCtxCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to shutdown http server", "error", err)
		if closeErr := httpServer.Close(); closeErr != nil {
			log.Error("failed to close http server", "error", closeErr)
		}
	}
	if err := app.Close(shutdownCtx); err != nil {
		log.Error("failed to shutdown dependencies", "error", err)
	}
	log.Info("service stopped")
}

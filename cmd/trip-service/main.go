package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/a-pelikh/tripgo/internal/config"
	httpHandler "github.com/a-pelikh/tripgo/internal/handler/http"
	"github.com/a-pelikh/tripgo/internal/logger"
	postgresTripRepository "github.com/a-pelikh/tripgo/internal/repository/postgres/trip"
	"github.com/a-pelikh/tripgo/internal/service/trip"
	postgresTxManager "github.com/a-pelikh/tripgo/internal/tx-manager/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustInit(cfg.LogLevel)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	dbCfg, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		log.Error("failed to parse database config", "error", err)
		return
	}

	dbCfg.MaxConns = cfg.DB.MaxConns
	dbCfg.MinConns = cfg.DB.MinConns
	dbCfg.MaxConnLifetime = cfg.DB.MaxConnLifetime
	dbCfg.ConnConfig.ConnectTimeout = cfg.DB.ConnectTimeout
	pool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		log.ErrorContext(ctx, "failed to create pool connection to database", "error", err)
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.ErrorContext(ctx, "failed to ping database", "error", err)
		return
	}

	txManager := postgresTxManager.New(log, pool, cfg.DB.QueryTimeout)

	tripRepository := postgresTripRepository.New(pool, cfg.DB.QueryTimeout)
	tripService := trip.New(txManager, tripRepository)

	router, err := httpHandler.NewRouter(log, tripService)
	if err != nil {
		log.Error("failed to initialize http router", "error", err)
		return
	}
	httpServer := http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed while serving http", "error", err)
		}
		return
	}

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCtxCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to shutdown http server", "error", err)
		if closeErr := httpServer.Close(); closeErr != nil {
			log.Error("failed to close http server", "error", closeErr)
		}
	}
}

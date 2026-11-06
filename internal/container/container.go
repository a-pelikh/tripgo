package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/a-pelikh/tripgo/api"
	"github.com/a-pelikh/tripgo/internal/config"
	httpTransport "github.com/a-pelikh/tripgo/internal/http"
	"github.com/a-pelikh/tripgo/internal/http/health"
	"github.com/a-pelikh/tripgo/internal/http/middleware"
	"github.com/a-pelikh/tripgo/internal/http/ready"
	createTripHandler "github.com/a-pelikh/tripgo/internal/http/trip/create"
	finishTripHandler "github.com/a-pelikh/tripgo/internal/http/trip/finish"
	getTripHandler "github.com/a-pelikh/tripgo/internal/http/trip/get"
	"github.com/a-pelikh/tripgo/internal/infra/postgres"
	"github.com/a-pelikh/tripgo/internal/infra/postgres/transaction"
	"github.com/a-pelikh/tripgo/internal/infra/telemetry"
	tripRepository "github.com/a-pelikh/tripgo/internal/repository/trip"
	createTrip "github.com/a-pelikh/tripgo/internal/usecase/trip/create"
	finishTrip "github.com/a-pelikh/tripgo/internal/usecase/trip/finish"
	getTrip "github.com/a-pelikh/tripgo/internal/usecase/trip/get"
)

type Container struct {
	Logger *slog.Logger
	Server *http.Server

	pool      *pgxpool.Pool
	telemetry *telemetry.Telemetry
}

func New(ctx context.Context, cfg config.Config) (_ *Container, err error) {
	telemetrySDK, err := telemetry.New(ctx, cfg.Telemetry, cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("initialize telemetry: %w", err)
	}
	log := telemetrySDK.Logger()

	var pool *pgxpool.Pool
	defer func() {
		if err == nil {
			return
		}
		if pool != nil {
			pool.Close()
		}
		_ = telemetrySDK.Shutdown(ctx)
	}()

	pool, err = postgres.New(ctx, cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres: %w", err)
	}

	txManager := transaction.New(log, pool, cfg.DB.QueryTimeout)
	trips := tripRepository.New(pool, cfg.DB.QueryTimeout)
	tripObserver, err := telemetry.NewTripObserver(telemetrySDK.Meter(), log)
	if err != nil {
		return nil, fmt.Errorf("initialize trip telemetry: %w", err)
	}

	createTripUseCase := createTrip.New(txManager, trips, tripObserver)
	getTripUseCase := getTrip.New(trips)
	finishTripUseCase := finishTrip.New(txManager, trips, tripObserver)

	createHandler := createTripHandler.New(log, createTripUseCase)
	getHandler := getTripHandler.New(log, getTripUseCase)
	finishHandler := finishTripHandler.New(log, finishTripUseCase)
	healthHandler := health.New()
	readyHandler := ready.New(trips)

	router := httpTransport.NewRouter(createHandler, getHandler, finishHandler, healthHandler, readyHandler)
	spec, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}
	httpMetrics, err := middleware.NewHTTPMetrics(telemetrySDK.Meter())
	if err != nil {
		return nil, fmt.Errorf("initialize HTTP metrics: %w", err)
	}

	handler := router
	handler = middleware.Validate(spec, handler)
	handler = httpMetrics.Wrap(handler)
	handler = middleware.Logging(log, handler)
	handler = middleware.Tracing(handler)
	handler = middleware.Recover(log, handler)

	return &Container{
		Logger:    log,
		pool:      pool,
		telemetry: telemetrySDK,
		Server: &http.Server{
			Addr:              cfg.HTTP.Addr,
			Handler:           handler,
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
		},
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	err := c.telemetry.Shutdown(ctx)
	c.pool.Close()
	return err
}

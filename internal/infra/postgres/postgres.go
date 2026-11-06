package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
)

type Config struct {
	URL             string        `env:"DATABASE_URL,required,notEmpty"`
	MaxConns        int32         `env:"DATABASE_MAX_CONNS,required"`
	MinConns        int32         `env:"DATABASE_MIN_CONNS,required"`
	MaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME,required"`
	ConnectTimeout  time.Duration `env:"DATABASE_CONNECT_TIMEOUT,required"`
	QueryTimeout    time.Duration `env:"DATABASE_QUERY_TIMEOUT,required"`
}

func (cfg Config) Validate() error {
	if cfg.MaxConns <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}
	if cfg.MinConns < 0 {
		return fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if cfg.MinConns > cfg.MaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}
	if cfg.MaxConnLifetime <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONN_LIFETIME must be positive")
	}
	if cfg.ConnectTimeout <= 0 {
		return fmt.Errorf("DATABASE_CONNECT_TIMEOUT must be positive")
	}
	if cfg.QueryTimeout <= 0 {
		return fmt.Errorf("DATABASE_QUERY_TIMEOUT must be positive")
	}
	return nil
}

func New(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	collection := attribute.String("db.collection.name", "trips")
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerAttributes(collection),
		otelpgx.WithMeterAttributes(collection),
		otelpgx.WithMetricOperationName(otelpgx.SQLOperationName),
	)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("record postgres pool stats: %w", err)
	}

	return pool, nil
}

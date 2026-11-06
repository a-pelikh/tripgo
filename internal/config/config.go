package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type app struct {
	LogLevel        string        `env:"LOG_LEVEL,required,notEmpty"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
	HTTP            http
	DB              db
}

type http struct {
	Addr              string        `env:"HTTP_ADDR,required,notEmpty"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT,required"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT,required"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT,required"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT,required"`
}

type db struct {
	URL             string        `env:"DATABASE_URL,required,notEmpty"`
	MaxConns        int32         `env:"DATABASE_MAX_CONNS,required"`
	MinConns        int32         `env:"DATABASE_MIN_CONNS,required"`
	MaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME,required"`
	ConnectTimeout  time.Duration `env:"DATABASE_CONNECT_TIMEOUT,required"`
	QueryTimeout    time.Duration `env:"DATABASE_QUERY_TIMEOUT,required"`
}

func Load() (app, error) {
	cfg, err := env.ParseAs[app]()
	if err != nil {
		return app{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return app{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (cfg app) validate() error {
	if cfg.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	if cfg.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_TIMEOUT must be positive")
	}
	if cfg.HTTP.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_HEADER_TIMEOUT must be positive")
	}
	if cfg.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("HTTP_WRITE_TIMEOUT must be positive")
	}
	if cfg.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("HTTP_IDLE_TIMEOUT must be positive")
	}
	if cfg.DB.MaxConns <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}
	if cfg.DB.MinConns < 0 {
		return fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if cfg.DB.MinConns > cfg.DB.MaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}
	if cfg.DB.MaxConnLifetime <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONN_LIFETIME must be positive")
	}
	if cfg.DB.ConnectTimeout <= 0 {
		return fmt.Errorf("DATABASE_CONNECT_TIMEOUT must be positive")
	}
	if cfg.DB.QueryTimeout <= 0 {
		return fmt.Errorf("DATABASE_QUERY_TIMEOUT must be positive")
	}

	return nil
}

func MustLoad() app {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}

	return cfg
}

package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/a-pelikh/tripgo/internal/infra/postgres"
	"github.com/a-pelikh/tripgo/internal/infra/telemetry"
)

type Config struct {
	LogLevel        string        `env:"LOG_LEVEL,required,notEmpty"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
	HTTP            HTTPConfig
	DB              postgres.Config
	Telemetry       telemetry.Config
}

type HTTPConfig struct {
	Addr              string        `env:"HTTP_ADDR,required,notEmpty"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT,required"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT,required"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT,required"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT,required"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (cfg Config) validate() error {
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
	if err := cfg.DB.Validate(); err != nil {
		return fmt.Errorf("postgres config: %w", err)
	}
	if err := cfg.Telemetry.Validate(); err != nil {
		return fmt.Errorf("telemetry config: %w", err)
	}

	return nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}

	return cfg
}

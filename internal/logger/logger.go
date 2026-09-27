package logger

import (
	"fmt"
	"log/slog"
	"os"
)

func MustInit(level string) *slog.Logger {
	logger, err := Init(level)
	if err != nil {
		panic(err)
	}

	return logger
}

func Init(level string) (*slog.Logger, error) {
	var slogLevel slog.Level

	if err := slogLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", level, err)
	}

	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slogLevel,
		},
	)

	log := slog.New(handler).With(
		slog.String("service", "trip-service"),
	)

	return log, nil
}

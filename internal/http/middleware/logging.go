package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		writer := &responseWriter{ResponseWriter: w}

		handler.ServeHTTP(writer, r)

		attrs := []any{
			"method", r.Method,
			"route", routePattern(r),
			"status", writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
		}
		if writer.Status() >= http.StatusInternalServerError {
			logger.ErrorContext(r.Context(), "request handled", attrs...)
			return
		}
		logger.InfoContext(r.Context(), "request handled", attrs...)
	})
}

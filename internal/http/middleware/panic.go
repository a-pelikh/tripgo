package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/a-pelikh/tripgo/internal/http/domainerror"
)

func Recover(logger *slog.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.ErrorContext(r.Context(), "panic recovered", "panic", err, "stack", string(debug.Stack()))
				domainerror.Internal(w, r)
			}
		}()

		handler.ServeHTTP(w, r)
	})
}

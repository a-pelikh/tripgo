package middleware

import (
	"log/slog"
	"net/http"
)

func Recover(logger *slog.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		handler.ServeHTTP(w, r)
	})
}

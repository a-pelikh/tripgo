package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func routePattern(r *http.Request) string {
	if route := chi.RouteContext(r.Context()).RoutePattern(); route != "" {
		return route
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case path == "/api/v1/trips":
		return "/api/v1/trips"
	case strings.HasPrefix(path, "/api/v1/trips/") && strings.HasSuffix(path, "/finish"):
		return "/api/v1/trips/{tripId}/finish"
	case strings.HasPrefix(path, "/api/v1/trips/"):
		return "/api/v1/trips/{tripId}"
	case path == "/health":
		return "/health"
	case path == "/ready":
		return "/ready"
	default:
		return "unmatched"
	}
}

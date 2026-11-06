package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HTTPMetrics struct {
	duration metric.Float64Histogram
	active   metric.Int64UpDownCounter
}

func NewHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of HTTP server requests."),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP duration metric: %w", err)
	}
	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Number of active HTTP server requests."),
	)
	if err != nil {
		return nil, fmt.Errorf("create active HTTP requests metric: %w", err)
	}
	return &HTTPMetrics{duration: duration, active: active}, nil
}

func (m *HTTPMetrics) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		route := routePattern(r)
		activeAttrs := metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("http.route", route),
		)
		m.active.Add(r.Context(), 1, activeAttrs)
		defer m.active.Add(r.Context(), -1, activeAttrs)

		writer := &responseWriter{ResponseWriter: w}
		handler.ServeHTTP(writer, r)

		m.duration.Record(
			r.Context(),
			time.Since(startedAt).Seconds(),
			metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", routePattern(r)),
				attribute.Int("http.response.status_code", writer.Status()),
			),
		)
	})
}

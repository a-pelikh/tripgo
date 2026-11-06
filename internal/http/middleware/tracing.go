package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(handler http.Handler) http.Handler {
	routeNamer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)

		route := routePattern(r)
		span := trace.SpanFromContext(r.Context())
		span.SetName(r.Method + " " + route)
		span.SetAttributes(attribute.String("http.route", route))
	})

	return otelhttp.NewHandler(
		routeNamer,
		"HTTP ",
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
	)
}

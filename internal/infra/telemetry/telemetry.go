package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const instrumentationName = "github.com/a-pelikh/tripgo"

type Config struct {
	ServiceName        string `env:"OTEL_SERVICE_NAME,required,notEmpty"`
	Endpoint           string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required,notEmpty"`
	Protocol           string `env:"OTEL_EXPORTER_OTLP_PROTOCOL,required,notEmpty"`
	ResourceAttributes string `env:"OTEL_RESOURCE_ATTRIBUTES,required,notEmpty"`
	TracesSampler      string `env:"OTEL_TRACES_SAMPLER,required,notEmpty"`
}

func (cfg Config) Validate() error {
	if cfg.Protocol != "http/protobuf" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be http/protobuf")
	}
	if _, err := sampler(cfg.TracesSampler); err != nil {
		return err
	}
	return nil
}

type Telemetry struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	logger         *slog.Logger
	meter          metric.Meter
}

func New(ctx context.Context, cfg Config, logLevel string) (*Telemetry, error) {
	logger, err := newLogger(logLevel)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", buildVersion()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	traceSampler, err := sampler(cfg.TracesSampler)
	if err != nil {
		return nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(traceSampler),
	)

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn("telemetry export failed", "error", err)
	}))

	return &Telemetry{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		logger:         logger,
		meter:          meterProvider.Meter(instrumentationName),
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(
		t.meterProvider.Shutdown(ctx),
		t.tracerProvider.Shutdown(ctx),
	)
}

func (t *Telemetry) Logger() *slog.Logger {
	return t.logger
}

func (t *Telemetry) Meter() metric.Meter {
	return t.meter
}

func sampler(value string) (sdktrace.Sampler, error) {
	switch strings.ToLower(value) {
	case "always_on":
		return sdktrace.AlwaysSample(), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	default:
		return nil, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER %q", value)
	}
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

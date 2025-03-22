package telemetry

import (
	"context"
	"fmt"

	"github.com/sakajunquality/buildx-telemetry/internal/buildx"
	"github.com/sakajunquality/buildx-telemetry/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Config holds the configuration for the tracer
type Config struct {
	OTLPEndpoint string
	ServiceName  string
}

// Tracer manages the OpenTelemetry tracing
type Tracer struct {
	provider *sdktrace.TracerProvider
	config   Config
	logger   logger.Logger
}

// NewTracer creates a new telemetry tracer with the given config
func NewTracer(ctx context.Context, config Config) (*Tracer, error) {
	// Create a noop logger if none is provided
	noop, _ := logger.New(logger.DefaultConfig())
	return NewTracerWithLogger(ctx, config, noop)
}

// NewTracerWithLogger creates a new telemetry tracer with a logger
func NewTracerWithLogger(ctx context.Context, config Config, log logger.Logger) (*Tracer, error) {
	log.Info("Initializing OpenTelemetry tracer",
		zap.String("endpoint", config.OTLPEndpoint),
		zap.String("service", config.ServiceName))

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)

	log.Info("OpenTelemetry tracer initialized")

	return &Tracer{
		provider: tracerProvider,
		config:   config,
		logger:   log,
	}, nil
}

// ExportBuildTraces exports the build steps as OpenTelemetry traces
func (t *Tracer) ExportBuildTraces(ctx context.Context, steps []buildx.BuildStep) (string, error) {
	t.logger.Info("Starting to export build traces", zap.Int("steps", len(steps)))

	tracer := otel.Tracer("buildx")
	ctx, span := tracer.Start(ctx, "build")
	defer span.End()
	traceID := span.SpanContext().TraceID()

	for i, step := range steps {
		spanName := step.Name
		if step.Cached {
			spanName += " (cached)"
		}
		_, span := tracer.Start(ctx, spanName, trace.WithTimestamp(step.Started))
		span.End(trace.WithTimestamp(step.Completed))

		if i%10 == 0 && i > 0 {
			t.logger.Debug("Exported step traces", zap.Int("count", i))
		}
	}

	t.logger.Info("Completed exporting build traces",
		zap.String("traceID", traceID.String()),
		zap.Int("steps", len(steps)))

	return traceID.String(), nil
}

// Shutdown gracefully shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	t.logger.Info("Shutting down OpenTelemetry tracer")
	return t.provider.Shutdown(ctx)
}

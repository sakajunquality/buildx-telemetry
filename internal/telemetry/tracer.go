package telemetry

import (
	"context"
	"fmt"

	"github.com/sakajunquality/buildx-telemetry/internal/buildx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
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
}

// NewTracer creates a new telemetry tracer with the given config
func NewTracer(ctx context.Context, config Config) (*Tracer, error) {
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

	return &Tracer{
		provider: tracerProvider,
		config:   config,
	}, nil
}

// ExportBuildTraces exports the build steps as OpenTelemetry traces
func (t *Tracer) ExportBuildTraces(ctx context.Context, steps []buildx.BuildStep) (string, error) {
	tracer := otel.Tracer("buildx")
	ctx, span := tracer.Start(ctx, "build")
	defer span.End()
	traceID := span.SpanContext().TraceID()

	for _, step := range steps {
		spanName := step.Name
		if step.Cached {
			spanName += " (cached)"
		}
		_, span := tracer.Start(ctx, spanName, trace.WithTimestamp(step.Started))
		span.End(trace.WithTimestamp(step.Completed))
	}

	return traceID.String(), nil
}

// Shutdown gracefully shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

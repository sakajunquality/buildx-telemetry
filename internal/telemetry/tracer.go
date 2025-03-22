package telemetry

import (
	"context"
	"fmt"

	"github.com/sakajunquality/buildx-telemetry/internal/buildx"
	"github.com/sakajunquality/buildx-telemetry/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	Version      string
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
		zap.String("service", config.ServiceName),
		zap.String("version", config.Version))

	// Check if we have a parent span context
	parentSpanContext := trace.SpanContextFromContext(ctx)
	if parentSpanContext.IsValid() {
		log.Info("Found valid parent trace context",
			zap.String("traceID", parentSpanContext.TraceID().String()),
			zap.String("spanID", parentSpanContext.SpanID().String()))
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	// Prepare resource attributes
	resourceOpts := []resource.Option{
		resource.WithAttributes(semconv.ServiceName(config.ServiceName)),
	}

	// Add version information if provided
	if config.Version != "" {
		resourceOpts = append(resourceOpts, resource.WithAttributes(
			semconv.ServiceVersion(config.Version),
		))
	}

	res, err := resource.New(ctx, resourceOpts...)
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

	// Create a new span for the build, potentially as a child of an existing trace
	tracer := otel.Tracer("buildx")

	// Check if we have a parent span in the context
	parentSpanContext := trace.SpanContextFromContext(ctx)

	var span trace.Span
	if parentSpanContext.IsValid() {
		t.logger.Info("Creating build span as child of parent span",
			zap.String("parentTraceID", parentSpanContext.TraceID().String()))
		ctx, span = tracer.Start(ctx, "docker-build")
	} else {
		t.logger.Info("Creating new root build span")
		ctx, span = tracer.Start(ctx, "docker-build")
	}

	// Add version attribute to the span if provided
	if t.config.Version != "" {
		span.SetAttributes(attribute.String("version", t.config.Version))
	}

	defer span.End()

	traceID := span.SpanContext().TraceID()

	for i, step := range steps {
		spanName := step.Name
		if step.Cached {
			spanName += " (cached)"
		}

		// Create child spans for each build step
		_, stepSpan := tracer.Start(ctx, spanName, trace.WithTimestamp(step.Started))

		// Add version attribute to step spans as well
		if t.config.Version != "" {
			stepSpan.SetAttributes(attribute.String("version", t.config.Version))
		}

		stepSpan.End(trace.WithTimestamp(step.Completed))

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

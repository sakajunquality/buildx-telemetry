package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/sakajunquality/buildx-telemetry/internal/buildx"
	"github.com/sakajunquality/buildx-telemetry/internal/logger"
	"github.com/sakajunquality/buildx-telemetry/internal/telemetry"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Build information. Populated at build-time by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	otlpEndpoint    = flag.String("otlp-endpoint", "localhost:4317", "OpenTelemetry endpoint")
	serviceName     = flag.String("service-name", "docker-build-telemetry", "Service name for telemetry")
	debug           = flag.Bool("debug", false, "Debug mode")
	inputFile       = flag.String("input", "", "Input file (defaults to stdin)")
	logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	exitCodeOnError = flag.Int("exit-code-on-error", 1, "Exit code when an error occurs")
	traceContext    = flag.String("trace-context", "", "W3C Trace Context header for distributed tracing (default: empty)")
	versionFlag     = flag.String("version", "", "Version information to add to the trace (default: empty)")
	showVersion     = flag.Bool("v", false, "Show version information and exit")
)

func main() {
	flag.Parse()

	// Show version information if requested
	if *showVersion {
		fmt.Printf("buildx-telemetry version %s, commit %s, built on %s\n", version, commit, date)
		fmt.Printf("Built with %s\n", runtime.Version())
		return
	}

	// Initialize logger
	logConfig := logger.DefaultConfig()
	logConfig.Development = *debug
	logConfig.Level = *logLevel

	log, err := logger.New(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(*exitCodeOnError)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("Starting buildx-telemetry",
		zap.String("app_version", version),
		zap.String("service", *serviceName),
		zap.String("otlp-endpoint", *otlpEndpoint),
		zap.Bool("debug", *debug),
		zap.Int("exit-code-on-error", *exitCodeOnError),
		zap.String("trace-context", *traceContext),
		zap.String("version", *versionFlag))

	// Set up the input reader
	var reader *os.File
	if *inputFile != "" {
		reader, err = os.Open(*inputFile)
		if err != nil {
			log.Error("Error opening file",
				zap.String("file", *inputFile),
				zap.Error(err))
			os.Exit(*exitCodeOnError)
		}
		defer reader.Close()
		log.Info("Reading from file", zap.String("file", *inputFile))
	} else {
		reader = os.Stdin
		log.Info("Reading from stdin")
	}

	// Parse buildx logs
	parser := buildx.NewParserWithLogger(reader, log)
	steps, err := parser.Parse()
	if err != nil {
		log.Error("Error parsing log", zap.Error(err))
		os.Exit(*exitCodeOnError)
	}

	log.Info("Parsed build log", zap.Int("step_count", len(steps)))

	// Set up trace context if provided
	ctx := context.Background()
	if *traceContext != "" {
		tc := propagation.TraceContext{}
		headerMap := propagation.MapCarrier{"traceparent": *traceContext}
		ctx = tc.Extract(ctx, headerMap)

		spanCtx := trace.SpanContextFromContext(ctx)
		if spanCtx.IsValid() {
			log.Info("Using parent trace context",
				zap.String("traceID", spanCtx.TraceID().String()),
				zap.String("spanID", spanCtx.SpanID().String()))
		} else {
			log.Warn("Invalid trace context provided", zap.String("trace-context", *traceContext))
		}
	}

	// Initialize telemetry tracer
	tracerConfig := telemetry.Config{
		OTLPEndpoint: *otlpEndpoint,
		ServiceName:  *serviceName,
	}

	// Add version if provided
	if *versionFlag != "" {
		tracerConfig.Version = *versionFlag
	} else if version != "dev" {
		// Use the build version if no explicit version was provided
		tracerConfig.Version = version
	}

	tracer, err := telemetry.NewTracerWithLogger(ctx, tracerConfig, log)
	if err != nil {
		log.Error("Error initializing tracer", zap.Error(err))
		os.Exit(*exitCodeOnError)
	}
	defer func() {
		if err := tracer.Shutdown(ctx); err != nil {
			log.Error("Error shutting down tracer", zap.Error(err))
		}
	}()

	// Export traces
	traceID, err := tracer.ExportBuildTraces(ctx, steps)
	if err != nil {
		log.Error("Error exporting traces", zap.Error(err))
		os.Exit(*exitCodeOnError)
	}

	log.Info("Exported traces", zap.String("traceID", traceID))
	fmt.Printf("TraceID: %s\n", traceID)

	// Print debug information if requested
	if *debug {
		log.Debug("Printing detailed build steps")
		buildx.PrintSteps(steps)
	}
}

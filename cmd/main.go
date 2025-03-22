package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sakajunquality/buildx-telemetry/internal/buildx"
	"github.com/sakajunquality/buildx-telemetry/internal/logger"
	"github.com/sakajunquality/buildx-telemetry/internal/telemetry"
	"go.uber.org/zap"
)

var (
	otlpEndpoint    = flag.String("otlp-endpoint", "localhost:4317", "OpenTelemetry endpoint")
	serviceName     = flag.String("service-name", "docker-build-telemetry", "Service name for telemetry")
	debug           = flag.Bool("debug", false, "Debug mode")
	inputFile       = flag.String("input", "", "Input file (defaults to stdin)")
	logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	exitCodeOnError = flag.Int("exit-code-on-error", 1, "Exit code when an error occurs")
)

func main() {
	flag.Parse()

	// Initialize logger
	logConfig := logger.DefaultConfig()
	logConfig.Development = *debug
	logConfig.Level = *logLevel

	log, err := logger.New(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(*exitCodeOnError)
	}
	defer log.Sync()

	log.Info("Starting buildx-telemetry",
		zap.String("service", *serviceName),
		zap.String("otlp-endpoint", *otlpEndpoint),
		zap.Bool("debug", *debug),
		zap.Int("exit-code-on-error", *exitCodeOnError))

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

	// Initialize telemetry tracer
	ctx := context.Background()
	tracer, err := telemetry.NewTracerWithLogger(ctx, telemetry.Config{
		OTLPEndpoint: *otlpEndpoint,
		ServiceName:  *serviceName,
	}, log)
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

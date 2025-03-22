package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sakajunquality/buildx-telemetry/internal/buildx"
	"github.com/sakajunquality/buildx-telemetry/internal/telemetry"
)

var (
	otlpEndpoint = flag.String("otlp-endpoint", "localhost:4317", "OpenTelemetry endpoint")
	serviceName  = flag.String("service-name", "docker-build-telemetry", "Service name for telemetry")
	debug        = flag.Bool("debug", false, "Debug mode")
	inputFile    = flag.String("input", "", "Input file (defaults to stdin)")
)

func main() {
	flag.Parse()

	// Set up the input reader
	var reader *os.File
	var err error
	if *inputFile != "" {
		reader, err = os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer reader.Close()
	} else {
		reader = os.Stdin
	}

	// Parse buildx logs
	parser := buildx.NewParser(reader)
	steps, err := parser.Parse()
	if err != nil {
		log.Fatalf("Error parsing log: %v", err)
	}

	// Initialize telemetry tracer
	ctx := context.Background()
	tracer, err := telemetry.NewTracer(ctx, telemetry.Config{
		OTLPEndpoint: *otlpEndpoint,
		ServiceName:  *serviceName,
	})
	if err != nil {
		log.Fatalf("Error initializing tracer: %v", err)
	}
	defer func() {
		if err := tracer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()

	// Export traces
	traceID, err := tracer.ExportBuildTraces(ctx, steps)
	if err != nil {
		log.Fatalf("Error exporting traces: %v", err)
	}
	fmt.Printf("TraceID: %s\n", traceID)

	// Print debug information if requested
	if *debug {
		buildx.PrintSteps(steps)
	}
}

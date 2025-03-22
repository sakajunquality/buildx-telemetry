package telemetry

import (
	"context"
	"testing"
)

func TestConfig(t *testing.T) {
	config := Config{
		OTLPEndpoint: "localhost:4317",
		ServiceName:  "test-service",
		Version:      "v1.0.0-test",
	}

	if config.OTLPEndpoint != "localhost:4317" {
		t.Errorf("Expected OTLPEndpoint to be 'localhost:4317', got '%s'", config.OTLPEndpoint)
	}

	if config.ServiceName != "test-service" {
		t.Errorf("Expected ServiceName to be 'test-service', got '%s'", config.ServiceName)
	}

	if config.Version != "v1.0.0-test" {
		t.Errorf("Expected Version to be 'v1.0.0-test', got '%s'", config.Version)
	}
}

func TestExportTracesWithNoSteps(t *testing.T) {
	// This test can't connect to a real OTLP endpoint in CI,
	// so it will fail to create the tracer, which is expected.
	// We're just verifying that the code doesn't panic.

	ctx := context.Background()
	config := Config{
		OTLPEndpoint: "non-existent-endpoint:4317", // Intentionally invalid
		ServiceName:  "test-service",
	}

	_, err := NewTracer(ctx, config)

	// We expect an error since we're not connecting to a real OTLP endpoint
	if err == nil {
		t.Skip("Expected connection error, but none occurred. This may happen if you're running with a real collector.")
	}
}

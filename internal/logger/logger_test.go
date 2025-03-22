package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Development {
		t.Errorf("Expected Development to be false by default")
	}

	if config.Level != "info" {
		t.Errorf("Expected default Level to be 'info', got '%s'", config.Level)
	}
}

func TestNewLogger(t *testing.T) {
	// Test with default config
	defaultConfig := DefaultConfig()
	logger, err := New(defaultConfig)

	if err != nil {
		t.Errorf("Expected no error creating logger with default config, got %v", err)
	}

	if logger == nil {
		t.Errorf("Expected logger to be created with default config")
	}

	// Test with development config
	devConfig := DefaultConfig()
	devConfig.Development = true
	devConfig.Level = "debug"

	devLogger, err := New(devConfig)
	if err != nil {
		t.Errorf("Expected no error creating development logger, got %v", err)
	}

	if devLogger == nil {
		t.Errorf("Expected development logger to be created")
	}

	// Test with invalid log level
	invalidConfig := DefaultConfig()
	invalidConfig.Level = "invalid-level"

	_, err = New(invalidConfig)
	if err == nil {
		t.Errorf("Expected error with invalid log level")
	}
}

func TestLoggerWith(t *testing.T) {
	logger, _ := New(DefaultConfig())

	childLogger := logger.With(zap.String("test", "value"))
	if childLogger == nil {
		t.Errorf("Expected child logger to be created")
	}
}

// Test that none of the logging methods panic
func TestLoggingMethods(t *testing.T) {
	logger, _ := New(DefaultConfig())

	// These should not panic
	logger.Info("info message", zap.String("test", "value"))
	logger.Debug("debug message", zap.String("test", "value"))
	logger.Warn("warn message", zap.String("test", "value"))
	logger.Error("error message", zap.String("test", "value"))

	// Fatal would exit the program, so we can't easily test it
}

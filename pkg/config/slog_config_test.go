package config

import (
	"errors"
	"testing"

	"log/slog"
)

func TestGetSlogLogger(t *testing.T) {
	t.Run("wrong type of log level", func(t *testing.T) {
		const logLevel = "infox"
		const logFormat = defaultLogFormat
		logger, err := getSlogLogger(logLevel, logFormat)
		if logger != nil {
			t.Errorf("expected logger to be nil, got %v", logger)
		}
		if err == nil || err.Error() != "slog: level string \"infox\": unknown name" {
			t.Errorf("expected error to be '%s', got %v", "slog: level string \"infox\": unknown name", err)
		}
	})

	t.Run("correct log level with default format", func(t *testing.T) {
		const logLevel = "warn"
		const logFormat = defaultLogFormat
		logger, err := getSlogLogger(logLevel, logFormat)
		if err != nil {
			t.Errorf("expected error to be nil, got %v", err)
		}
		if logger == nil {
			t.Errorf("expected logger to be not nil, got %v", logger)
		}

		verifyLoggerHandler(t, logger, logFormat)
	})

	t.Run("wrong type of log format", func(t *testing.T) {
		const logLevel = defaultLogLevel
		const logFormat = "test"
		logger, err := getSlogLogger(logLevel, logFormat)
		if logger != nil {
			t.Errorf("expected logger to be nil, got %v", logger)
		}
		if !errors.Is(err, ErrUnknownLogFormat) {
			t.Errorf("expected error to be '%v', got %v", ErrUnknownLogFormat, err)
		}
	})

	t.Run("correct log level and format", func(t *testing.T) {
		const logLevel = defaultLogLevel
		const logFormat = defaultLogFormat
		logger, err := getSlogLogger(logLevel, logFormat)
		if err != nil {
			t.Errorf("expected error to be nil, got %v", err)
		}
		if logger == nil {
			t.Errorf("expected logger to be not nil, got %v", logger)
		}

		verifyLoggerHandler(t, logger, logFormat)
	})

	t.Run("json log format", func(t *testing.T) {
		const logLevel = defaultLogLevel
		const logFormat = "json"
		logger, err := getSlogLogger(logLevel, logFormat)
		if err != nil {
			t.Errorf("expected error to be nil, got %v", err)
		}
		if logger == nil {
			t.Errorf("expected logger to be not nil, got %v", logger)
		}

		verifyLoggerHandler(t, logger, logFormat)
	})
}

func verifyLoggerHandler(t *testing.T, logger *slog.Logger, expectedFormat string) {
	handler := logger.Handler()
	if handler == nil {
		t.Errorf("expected handler to be not nil, got nil")
	}

	switch h := handler.(type) {
	case *slog.TextHandler:
		if expectedFormat != LogFormatText {
			t.Errorf("expected TextHandler for format %s, got %T", expectedFormat, h)
		}
	case *slog.JSONHandler:
		if expectedFormat != LogFormatJSON {
			t.Errorf("expected JSONHandler for format %s, got %T", expectedFormat, h)
		}
	default:
		t.Errorf("unexpected handler type: %T", h)
	}
}

func TestSetupSlog(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			Logging: LoggingConfig{
				Level:  "info",
				Format: "text",
			},
		}

		err := SetupSlog(cfg)
		if err != nil {
			t.Errorf("expected error to be nil, got %v", err)
		}

		// Verify that the default logger is set
		defaultLogger := slog.Default()
		if defaultLogger == nil {
			t.Errorf("expected default logger to be not nil, got nil")
		}
	})

	t.Run("invalid log level", func(t *testing.T) {
		cfg := &Config{
			Logging: LoggingConfig{
				Level:  "invalid-level",
				Format: "text",
			},
		}

		err := SetupSlog(cfg)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("invalid log format", func(t *testing.T) {
		cfg := &Config{
			Logging: LoggingConfig{
				Level:  "info",
				Format: "invalid-format",
			},
		}

		err := SetupSlog(cfg)
		if err == nil || !errors.Is(err, ErrUnknownLogFormat) {
			t.Errorf("expected error '%v', got %v", ErrUnknownLogFormat, err)
		}
	})
}

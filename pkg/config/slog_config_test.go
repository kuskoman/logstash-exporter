package config

import (
	"errors"
	"testing"
)

func TestGetSlogLogger(t *testing.T) {
	t.Run("Wrong type of log level", func(t *testing.T) {
		const LogLevel = "infox"
		const LogFormat = defaultLogFormat
		logger, err := getSlogLogger(LogLevel, LogFormat)
		if logger != nil {
			t.Errorf("expected logger to be nil, got %s\"", logger)
		}
		if err.Error() != "slog: level string \"infox\": unknown name" {
			t.Errorf("expected error to be '%s', got %s\"", "slog: level string \"infox\": unknown name", err)
		}
	})
	t.Run("Wrong type of log level", func(t *testing.T) {
		const LogLevel = "warn"
		const LogFormat = defaultLogFormat
		logger, err := getSlogLogger(LogLevel, LogFormat)
		if err != nil {
			t.Errorf("expected error to be nil, got %s\"", err)
		}
		if logger == nil {
			t.Errorf("expected logger to be not nil, got %s\"", logger)
		}
	})
	t.Run("Wrong type of log format", func(t *testing.T) {
		const LogLevel = defaultLogLevel
		const LogFormat = "test"
		logger, err := getSlogLogger(LogLevel, LogFormat)
		if logger != nil {
			t.Errorf("expected logger to be nil, got %s\"", logger)
		}
		if !errors.Is(err, ErrUnknownLogFormat) {
			t.Errorf("expected error to be '%s', got %s\"", ErrUnknownLogFormat, err)
		}
	})
	t.Run("Correct type of log format", func(t *testing.T) {
		const LogLevel = defaultLogLevel
		const LogFormat = defaultLogFormat
		logger, err := getSlogLogger(LogLevel, LogFormat)
		if err != nil {
			t.Errorf("expected error to be nil, got %s\"", err)
		}
		if logger == nil {
			t.Errorf("expected logger to be not nil, got %s\"", logger)
		}
	})
	t.Run("Json type of log format", func(t *testing.T) {
		const LogLevel = defaultLogLevel
		const LogFormat = "json"
		logger, err := getSlogLogger(LogLevel, LogFormat)
		if err != nil {
			t.Errorf("expected error to be nil, got %s\"", err)
		}
		if logger == nil {
			t.Errorf("expected logger to be not nil, got %s\"", logger)
		}
	})
}

package config

import (
	"errors"
	"testing"
)

func TestSetupSlog(t *testing.T) {
	t.Run("Wrong type of log level", func(t *testing.T) {
		const LogLevel = "infox"
		const LogFormat = defaultLogFormat
		logger, err := SetupSlog(LogLevel, LogFormat)
		if logger != nil {
			t.Errorf("Expected logger to be nil, got %s\"", logger)
		}
		if err.Error() != "slog: level string \"infox\": unknown name" {
			t.Errorf("Expected error to be '%s', got %s\"", "slog: level string \"infox\": unknown name", err)
		}
	})
	t.Run("Wrong type of log level", func(t *testing.T) {
		const LogLevel = "warn"
		const LogFormat = defaultLogFormat
		logger, err := SetupSlog(LogLevel, LogFormat)
		if err != nil {
			t.Errorf("Expected error to be nil, got %s\"", err)
		}
		if logger == nil {
			t.Errorf("Expected logger to be not nil, got %s\"", logger)
		}
	})
	t.Run("Wrong type of log format", func(t *testing.T) {
		const LogLevel = defaultLogLevel
		const LogFormat = "test"
		logger, err := SetupSlog(LogLevel, LogFormat)
		if logger != nil {
			t.Errorf("Expected logger to be nil, got %s\"", logger)
		}
		if !errors.Is(err, ErrUnknownLogFormat) {
			t.Errorf("Expected error to be '%s', got %s\"", ErrUnknownLogFormat, err)
		}
	})
	t.Run("Correct type of log format", func(t *testing.T) {
		const LogLevel = defaultLogLevel
		const LogFormat = defaultLogFormat
		logger, err := SetupSlog(LogLevel, LogFormat)
		if err != nil {
			t.Errorf("Expected error to be nil, got %s\"", err)
		}
		if logger == nil {
			t.Errorf("Expected logger to be not nil, got %s\"", logger)
		}
	})
	t.Run("Json type of log format", func(t *testing.T) {
		const LogLevel = defaultLogLevel
		const LogFormat = "json"
		logger, err := SetupSlog(LogLevel, LogFormat)
		if err != nil {
			t.Errorf("Expected error to be nil, got %s\"", err)
		}
		if logger == nil {
			t.Errorf("Expected logger to be not nil, got %s\"", logger)
		}
	})
}

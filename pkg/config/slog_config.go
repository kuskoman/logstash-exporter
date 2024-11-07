package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ErrUnknownLogFormat is returned when the log format is not recognized by the slog package
var ErrUnknownLogFormat = fmt.Errorf("unknown log format")

const (
	// LogFormatJSON is the JSON log format
	LogFormatJSON = "json"
	// LogFormatText is the text log format
	LogFormatText = "text"
)

func getSlogHandler(logFormat string, level slog.Level) slog.Handler {
	handlerOptions := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(logFormat) {
	case LogFormatText:
		return slog.NewTextHandler(os.Stdout, handlerOptions)
	case LogFormatJSON:
		return slog.NewJSONHandler(os.Stdout, handlerOptions)
	default:
		return nil
	}
}

func getSlogLogger(logLevel string, logFormat string) (*slog.Logger, error) {
	level := slog.LevelInfo
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return nil, err
	}

	handler := getSlogHandler(logFormat, level)
	if handler == nil {
		return nil, ErrUnknownLogFormat
	}

	return slog.New(handler), nil
}

func SetupSlog(cfg *Config) error {
	logLevel, logFormat := cfg.Logging.Level, cfg.Logging.Format
	logger, err := getSlogLogger(logLevel, logFormat)
	if err != nil {
		return err
	}

	slog.SetDefault(logger)
	return nil
}

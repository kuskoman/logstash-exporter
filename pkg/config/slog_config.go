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

func SetupSlog(logLevel string, logFormat string) (*slog.Logger, error) {
	level := slog.LevelInfo
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	switch strings.ToLower(logFormat) {
	case LogFormatText:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	case LogFormatJSON:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	default:
		return nil, ErrUnknownLogFormat
	}

	return slog.New(handler), nil
}

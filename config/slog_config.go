package config

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/exp/slog"
)

var (
	LogLevel  = getEnvWithDefault("LOG_LEVEL", "info")
	LogFormat = getEnvWithDefault("LOG_FORMAT", "text")

	ErrUnknownLogFormat = fmt.Errorf("unknown log format")
)

func SetupSlog() (*slog.Logger, error) {
	level := slog.LevelInfo
	err := level.UnmarshalText([]byte(LogLevel))
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	switch strings.ToLower(LogFormat) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	default:
		return nil, ErrUnknownLogFormat
	}

	return slog.New(handler), nil
}

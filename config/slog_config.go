package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var ErrUnknownLogFormat = fmt.Errorf("unknown log format")

func SetupSlog(logLevel string, logFormat string) (*slog.Logger, error) {
	level := slog.LevelInfo
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	switch strings.ToLower(logFormat) {
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

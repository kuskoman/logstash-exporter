package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/kuskoman/logstash-exporter/internal/startup_manager"
)

func main() {
	ctx := context.Background()

	flagsConfig, err := flags.ParseFlags(os.Args[1:])
	if err != nil {
		slog.Error("failed to parse flags", "err", err)
		return
	}

	shouldExit := flags.HandleFlags(flagsConfig)
	if shouldExit {
		return
	}

	startupManager := startup_manager.NewStartupManager(flagsConfig)
	err = startupManager.Initialize(ctx)
	if err != nil {
		slog.Error("failed to initialize logstash exporter", "err", err)
		return
	}
}

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/kuskoman/logstash-exporter/internal/startup_manager"
)

func main() {
	flagsConfig, err := flags.ParseFlags(os.Args[1:])
	if err != nil {
		slog.Error("failed to parse flags", "err", err)
		return
	}

	if shouldExit := flags.HandleFlags(flagsConfig); shouldExit {
		os.Exit(0)
	}

	startupManager, err := startup_manager.NewStartupManager(flagsConfig.ConfigLocation, flagsConfig)
	if err != nil {
		slog.Error("failed to create startup manager", "err", err)
		os.Exit(1)
	}

	ctx := context.TODO()
	if err := startupManager.Initialize(ctx); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

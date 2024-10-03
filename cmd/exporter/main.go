package main

import (
	"context"
	"log/slog"

	"github.com/kuskoman/logstash-exporter/internal/startup_manager"
)

func main() {
	ctx := context.Background()

	startupManager := startup_manager.NewStartupManager()
	err := startupManager.Initialize(ctx)
	if err != nil {
		slog.Error("failed to initialize logstash exporter", "err", err)
		return
	}
}

package main

import (
	"context"

	"github.com/kuskoman/logstash-exporter/internal/startup_manager"
)

func main() {
	ctx := context.Background()

	startupManager := startup_manager.NewStartupManager()
	startupManager.Initialize(ctx)
}

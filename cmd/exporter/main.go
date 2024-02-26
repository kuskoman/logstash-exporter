package main

import "github.com/kuskoman/logstash-exporter/internal/startup_manager"

func main() {
	startupManager := startup_manager.NewStartupManager()
	startupManager.Initialize()
}

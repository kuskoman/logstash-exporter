package main

import (
	"log"
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/kuskoman/logstash-exporter/collectors"
	"github.com/kuskoman/logstash-exporter/config"
	"github.com/kuskoman/logstash-exporter/server"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	warn := godotenv.Load()
	if warn != nil {
		log.Println(warn)
	}

	logger, err := config.SetupSlog()
	if err != nil {
		log.Fatalf("failed to setup slog: %s", err)
	}
	slog.SetDefault(logger)

	port := config.Port
	host := config.Host
	logstashUrl := config.LogstashUrl

	slog.Debug("Application starting... ")
	versionInfo := config.GetVersionInfo()
	slog.Info(versionInfo.String())

	collectorManager := collectors.NewCollectorManager(logstashUrl)
	server := server.NewAppServer(host, port)
	prometheus.MustRegister(collectorManager)

	slog.Info("Starting server on port", "port", port)
	err = server.ListenAndServe()
	if err != nil {
		slog.Error("failed to listen and serve", "err", err)
	}
}

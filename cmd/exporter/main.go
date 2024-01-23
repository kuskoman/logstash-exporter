package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/kuskoman/logstash-exporter/collectors"
	"github.com/kuskoman/logstash-exporter/config"
	"github.com/kuskoman/logstash-exporter/server"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	version := flag.Bool("version", false, "prints the version and exits")

	flag.Parse()
	if *version {
		fmt.Printf("%s\n", config.SemanticVersion)
		return
	}

	warn := godotenv.Load()
	if warn != nil {
		log.Printf("failed to load .env file: %s", warn)
	}

	exporterConfig, err := config.GetConfig(config.ExporterConfigLocation)
	if err != nil {
		log.Fatalf("failed to get exporter config: %s", err)
		os.Exit(1)
	}

	logger, err := config.SetupSlog(exporterConfig.Logging.Level, exporterConfig.Logging.Format)
	if err != nil {
		log.Fatalf("failed to setup slog: %s", err)
	} else {
		slog.SetDefault(logger)
	}

	host := exporterConfig.Server.Host
	port := strconv.Itoa(exporterConfig.Server.Port)

	slog.Debug("application starting... ")
	versionInfo := config.GetVersionInfo()
	slog.Info(versionInfo.String())

	slog.Debug("http timeout", "timeout", exporterConfig.Logstash.HttpTimeout)

	collectorManager := collectors.NewCollectorManager(exporterConfig.Logstash.Servers,
        exporterConfig.Logstash.HttpTimeout)
	prometheus.MustRegister(collectorManager)

	appServer := server.NewAppServer(host, port, exporterConfig, exporterConfig.Logstash.HttpTimeout)

	slog.Info("starting server on", "host", host, "port", port)
	if err := appServer.ListenAndServe(); err != nil {
		slog.Error("failed to listen and serve", "err", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/kuskoman/logstash-exporter/pkg/manager"
)

func main() {
	versionFlag := flag.Bool("version", false, "prints the version and exits")
	helpFlag := flag.Bool("help", false, "prints the help message and exits")
	configLocationFlag := flag.String("config", config.ExporterConfigLocation, "location of the exporter config file")

	flag.Parse()

	if *helpFlag {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		return
	}

	if *versionFlag {
		fmt.Printf("%s\n", config.SemanticVersion)
		return
	}

	warn := godotenv.Load()

	exporterConfig, err := config.GetConfig(*configLocationFlag)
	if err != nil {
		log.Fatalf("failed to get exporter config: %s", err)
		os.Exit(1)
	}

	logger, err := config.SetupSlog(exporterConfig.Logging.Level, exporterConfig.Logging.Format)
	if err != nil {
		log.Printf("failed to load .env file: %s", err)
		log.Fatalf("failed to setup slog: %s", err)
	}

	slog.SetDefault(logger)

	if warn != nil {
		slog.Warn("failed to load .env file", "error", warn)
	}

	host := exporterConfig.Server.Host
	port := strconv.Itoa(exporterConfig.Server.Port)

	slog.Debug("application starting... ")
	versionInfo := config.GetVersionInfo()
	slog.Info(versionInfo.String())

	slog.Debug("http timeout", "timeout", exporterConfig.Logstash.HttpTimeout)

	collectorManager := manager.NewCollectorManager(
		exporterConfig.Logstash.Servers,
		exporterConfig.Logstash.HttpTimeout,
		exporterConfig.Logstash.HttpInsecure,
	)
	prometheus.MustRegister(collectorManager)

	appServer := server.NewAppServer(host, port, exporterConfig, exporterConfig.Logstash.HttpTimeout)

	slog.Info("starting server on", "host", host, "port", port)
	if err := appServer.ListenAndServe(); err != nil {
		slog.Error("failed to listen and serve", "err", err)
		os.Exit(1)
	}
}

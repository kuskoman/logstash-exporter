package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

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
		log.Println(warn)
	}

	logger, err := config.SetupSlog()
	if err != nil {
		log.Fatalf("failed to setup slog: %s", err)
	}
	slog.SetDefault(logger)

	port, host := config.Port, config.Host
	logstashUrl := config.LogstashUrl

	slog.Debug("application starting... ")
	versionInfo := config.GetVersionInfo()
	slog.Info(versionInfo.String())

	httpInsecure := config.GetHttpInsecure()
	slog.Debug("http insecure", "insecure", httpInsecure)

	httpTimeout, err := config.GetHttpTimeout()
	if err != nil {
		slog.Error("failed to get http timeout", "err", err)
		os.Exit(1)
	}
	slog.Debug("http timeout", "timeout", httpTimeout)

	collectorManager := collectors.NewCollectorManager(logstashUrl, httpInsecure, httpTimeout)
	appServer := server.NewAppServer(host, port, httpTimeout)
	prometheus.MustRegister(collectorManager)

	slog.Info("starting server on", "host", host, "port", port)
	if err := appServer.ListenAndServe(); err != nil {
		slog.Error("failed to listen and serve", "err", err)
		os.Exit(1)
	}
}

package main

import (
	"github.com/kuskoman/logstash-exporter/collectors"
	"github.com/kuskoman/logstash-exporter/config"
	"github.com/kuskoman/logstash-exporter/helpers"
	"github.com/kuskoman/logstash-exporter/server"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	err := helpers.InitializeLogger()
	if err != nil {
		panic(err)
	}

	warn := helpers.LoadEnv()
	if warn != nil {
		helpers.Logger.Warn(warn)
	}

	port := config.Port
	host := config.Host
	logstashUrl := config.LogstashUrl

	helpers.Logger.Info("Application starting...")

	collectorManager := collectors.NewCollectorManager(logstashUrl)
	server := server.NewAppServer(host, port)
	prometheus.MustRegister(collectorManager)

	helpers.Logger.Infof("Starting server on port %s", port)
	err = server.ListenAndServe()
	if err != nil {
		helpers.Logger.Panic(err)
	}
}

package main

import (
	"log"

	"github.com/kuskoman/logstash-exporter/collectors"
	"github.com/kuskoman/logstash-exporter/config"
	"github.com/kuskoman/logstash-exporter/server"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	port := config.Port
	host := config.Host
	logstashUrl := config.LogstashUrl

	log.Println("Application starting...")

	collectorManager := collectors.NewCollectorManager(logstashUrl)
	server := server.NewAppServer(host, port)
	prometheus.MustRegister(collectorManager)

	log.Printf("Starting server on port %s", port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"fmt"
	"log"

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

	port := config.Port
	host := config.Host
	var logstashApiUrls []string

	log.Println("Application starting...")
	if config.UseKubernetesEndpoints() {
		log.Println("Using Kubernetes Service API to locate Logstash Replicas...")
		logstashEndpoints, err := config.GetKubernetesLogstashApiEndpoints()
		if err != nil {
			log.Fatal(err)
		}
		if len(logstashEndpoints) < 1 {
			log.Fatal("No Logstash Kubernetes services with API endpoints were found. Exiting.")
		}
		for _, logstash := range logstashEndpoints {
			logstashUrl := fmt.Sprintf("%s:%d", logstash.Ip, logstash.Port)
			logstashApiUrls = append(logstashApiUrls, logstashUrl)
		}
	} else {
		log.Println("Using single-instance Logstash URL.")
		logstashApiUrls = append(logstashApiUrls, config.LogstashUrl)
	}
	collectorManager := collectors.NewCollectorManager(logstashApiUrls)

	server := server.NewAppServer(host, port)
	prometheus.MustRegister(collectorManager)

	log.Printf("Starting server on port %s", port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

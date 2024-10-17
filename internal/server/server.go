package server

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// NewAppServer creates a new http server with the given host and port
// and registers the prometheus handler and the healthcheck handler
// to the server's mux. The prometheus handler is managed under the
// hood by the prometheus client library.
func NewAppServer(cfg *config.Config) *http.Server {
	logstashUrls := convertServersToUrls(cfg.Logstash.Servers)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})

	mux.HandleFunc("/healthcheck", getHealthCheck(logstashUrls, cfg.Logstash.HttpTimeout))
	mux.HandleFunc("/version", getVersionInfoHandler(config.GetVersionInfo()))

	host, port := cfg.Server.Host, cfg.Server.Port
	listenUrl := fmt.Sprintf("%s:%d", host, port)

	server := &http.Server{
		Addr:    listenUrl,
		Handler: mux,
	}

	return server
}

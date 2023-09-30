package server

import (
	"fmt"
	"net/http"

	"github.com/kuskoman/logstash-exporter/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewAppServer creates a new http server with the given host and port
// and registers the prometheus handler and the healthcheck handler
// to the server's mux. The prometheus handler is managed under the
// hood by the prometheus client library.
func NewAppServer(host, port string, cfg *config.Config) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/healthcheck", getHealthCheck(cfg.GetLogstashUrls()))
	mux.HandleFunc("/version", getVersionInfoHandler(config.GetVersionInfo()))

	listenUrl := fmt.Sprintf("%s:%s", host, port)

	server := &http.Server{
		Addr:    listenUrl,
		Handler: mux,
	}

	return server
}

package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kuskoman/logstash-exporter/pkg/config"
	customtls "github.com/kuskoman/logstash-exporter/pkg/tls"
)

// NewAppServer creates a new http server with the given host and port
// and registers the prometheus handler and the healthcheck handler
// to the server's mux. The prometheus handler is managed under the
// hood by the prometheus client library.
func NewAppServer(cfg *config.Config) *http.Server {
	logstashUrls := convertInstancesToUrls(cfg.Logstash.Instances)

	mux := http.NewServeMux()
	handler := promhttp.Handler()

	// Configure basic authentication if enabled
	if cfg.Server.BasicAuth != nil {
		users, err := cfg.Server.BasicAuth.GetUsers()
		if err != nil {
			panic(fmt.Errorf("failed to get authentication users: %w", err))
		}

		handler = customtls.MultiUserAuthMiddleware(handler, users)
	}

	mux.Handle("/metrics", handler)
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

	// Configure read and write timeouts if specified
	if cfg.Server.ReadTimeout > 0 {
		server.ReadTimeout = time.Duration(cfg.Server.ReadTimeout) * time.Second
	}
	if cfg.Server.WriteTimeout > 0 {
		server.WriteTimeout = time.Duration(cfg.Server.WriteTimeout) * time.Second
	}

	return server
}

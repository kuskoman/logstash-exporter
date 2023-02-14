package server

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewAppServer(host, port string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/healthcheck", healthCheck)

	listenUrl := fmt.Sprintf("%s:%s", host, port)

	server := &http.Server{
		Addr:    listenUrl,
		Handler: mux,
	}

	return server
}

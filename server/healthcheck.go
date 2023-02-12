package server

import (
	"net/http"
	"time"

	"github.com/kuskoman/logstash-exporter/config"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: time.Second * 2,
	}

	resp, err := client.Get(config.LogstashUrl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

package server

import (
	"net/http"
	"time"
)

func getHealthCheck(logstashURL string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{
			Timeout: time.Second * 2,
		}

		resp, err := client.Get(logstashURL)
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
}

package server

import (
	"encoding/json"
	"net/http"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// getVersionInfoHandler returns a handler function that returns the current
// build information based on information provided during compilation and
// fetched from runtime.
func getVersionInfoHandler(versionInfo *config.VersionInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(versionInfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

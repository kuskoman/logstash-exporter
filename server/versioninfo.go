package server

import (
	"encoding/json"
	"net/http"

	"github.com/kuskoman/logstash-exporter/config"
)

// handleVersionInfo returns the current build information based on information
// provided during compilation and fetched from runtime.
func handleVersionInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config.GetBuildInfo())
}

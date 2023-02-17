package logstashclient

import (
	"fmt"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

func (client *DefaultClient) GetNodeInfo() (*responses.NodeInfoResponse, error) {
	fullPath := client.endpoint
	return getMetrics[responses.NodeInfoResponse](client.httpClient, fullPath)
}

func (client *DefaultClient) GetNodeStats() (*responses.NodeStatsResponse, error) {
	fullPath := fmt.Sprintf("%s/_node/stats", client.endpoint)
	return getMetrics[responses.NodeStatsResponse](client.httpClient, fullPath)
}

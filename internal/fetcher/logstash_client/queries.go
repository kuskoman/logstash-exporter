package logstash_client

import (
	"context"
	"fmt"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
)

// GetNodeInfo fetches the node info from the "/" endpoint of the Logstash API
func (client *DefaultClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	fullPath := client.endpoint
	return getMetrics[responses.NodeInfoResponse](ctx, client.httpClient, fullPath)
}

// GetNodeStats fetches the node stats from the "/_node/stats" endpoint of the Logstash API
func (client *DefaultClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	fullPath := fmt.Sprintf("%s/_node/stats", client.endpoint)
	return getMetrics[responses.NodeStatsResponse](ctx, client.httpClient, fullPath)
}

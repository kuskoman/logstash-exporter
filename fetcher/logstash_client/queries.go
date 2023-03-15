package logstash_client

import (
	"context"
	"fmt"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

func (client *DefaultClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	fullPath := client.endpoint
	return getMetrics[responses.NodeInfoResponse](ctx, client.httpClient, fullPath)
}

func (client *DefaultClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	fullPath := fmt.Sprintf("%s/_node/stats", client.endpoint)
	return getMetrics[responses.NodeStatsResponse](ctx, client.httpClient, fullPath)
}

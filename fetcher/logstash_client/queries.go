package logstashclient

import (
	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/httphandler"
)

func (c *DefaultClient) GetNodeInfo() (*responses.NodeInfoResponse, error) {
	var nodeInfoResponse responses.NodeInfoResponse
	err := httphandler.GetMetrics(c.httpClient, "", &nodeInfoResponse)
	if err != nil {
		return nil, err
	}

	return &nodeInfoResponse, nil
}

func (c *DefaultClient) GetNodeStats() (*responses.NodestatsResponse, error) {
	var nodeStatsResponse responses.NodestatsResponse
	err := httphandler.GetMetrics(c.httpClient, "_node/stats", &nodeStatsResponse)
	if err != nil {
		return nil, err
	}

	return &nodeStatsResponse, nil
}

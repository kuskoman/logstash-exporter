package logstashclient

import (
	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/httphandler"
)

func (c *DefaultClient) GetNodeInfo() (*responses.NodeInfoResponse, error) {
	var nodeInfoResponse responses.NodeInfoResponse
	err := httphandler.GetMetrics(c.httpClient, &nodeInfoResponse)
	if err != nil {
		return nil, err
	}

	return &nodeInfoResponse, nil
}

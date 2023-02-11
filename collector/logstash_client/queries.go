package logstashclient

import (
	"github.com/kuskoman/logstash-exporter/collector/responses"
	"github.com/kuskoman/logstash-exporter/httpclient"
)

func (c *Client) GetNodeInfo() (*responses.NodeInfoResponse, error) {
	var nodeInfoResponse responses.NodeInfoResponse
	err := httpclient.GetMetrics(c.httpClient, &nodeInfoResponse)
	if err != nil {
		return nil, err
	}

	return &nodeInfoResponse, nil
}

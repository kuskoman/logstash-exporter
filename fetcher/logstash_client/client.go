package logstashclient

import (
	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/httphandler"
)

type DefaultClient struct {
	httpClient httphandler.HTTPHandler
}

type Client interface {
	GetNodeInfo() (*responses.NodeInfoResponse, error)
	GetNodeStats() (*responses.NodestatsResponse, error)
}

const defaultLogstashEndpoint = "http://localhost:9600"

func NewClient(httpClient httphandler.HTTPHandler) *DefaultClient {
	var clientHandler httphandler.HTTPHandler

	if httpClient == nil {
		defaultHandler := httphandler.GetDefaultHTTPHandler(defaultLogstashEndpoint)
		clientHandler = defaultHandler
	} else {
		clientHandler = httpClient
	}

	return &DefaultClient{httpClient: clientHandler}
}

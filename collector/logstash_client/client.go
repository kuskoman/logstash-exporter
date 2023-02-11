package logstashclient

import "github.com/kuskoman/logstash-exporter/httphandler"

type Client struct {
	httpClient httphandler.HTTPHandler
}

const defaultLogstashEndpoint = "http://localhost:9600"

func NewClient(httpClient httphandler.HTTPHandler) *Client {
	var clientHandler httphandler.HTTPHandler

	if httpClient == nil {
		defaultHandler := httphandler.GetDefaultHTTPHandler(defaultLogstashEndpoint)
		clientHandler = defaultHandler
	} else {
		clientHandler = httpClient
	}

	return &Client{httpClient: clientHandler}
}

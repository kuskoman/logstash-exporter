package logstashclient

import "github.com/kuskoman/logstash-exporter/httpclient"

type Client struct {
	httpClient httpclient.HTTPHandler
}

const defaultLogstashEndpoint = "http://localhost:9600"

func NewClient(httpClient *httpclient.HTTPHandler) *Client {
	var clientHandler httpclient.HTTPHandler

	if httpClient == nil {
		defaultHandler := httpclient.GetDefaultHTTPHandler(defaultLogstashEndpoint)
		clientHandler = defaultHandler
	} else {
		clientHandler = *httpClient
	}

	return &Client{httpClient: clientHandler}
}

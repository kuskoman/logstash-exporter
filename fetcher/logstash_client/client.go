package logstashclient

import (
	"encoding/json"
	"net/http"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

type Client interface {
	GetNodeInfo() (*responses.NodeInfoResponse, error)
	GetNodeStats() (*responses.NodeStatsResponse, error)
}

type httpClient interface {
	Get(url string) (*http.Response, error)
}

type DefaultClient struct {
	httpClient httpClient
	endpoint   string
}

const defaultLogstashEndpoint = "http://localhost:9600"

func NewClient(endpoint string) Client {
	client := &DefaultClient{endpoint: endpoint}
	client.httpClient = &http.Client{}

	if endpoint == "" {
		client.endpoint = defaultLogstashEndpoint
	}

	return client
}

func getMetrics[T any](client httpClient, endpoint string) (*T, error) {
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return deserializeHttpResponse[T](resp)
}

func deserializeHttpResponse[T any](response *http.Response) (*T, error) {
	var result T

	err := json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

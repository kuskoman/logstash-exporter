package logstash_client

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

// Client is an interface for the Logstash client able to fetch data from the Logstash API
type Client interface {
	GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error)
	GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error)

	GetEndpoint() string
}

// DefaultClient is the default implementation of the Client interface
type DefaultClient struct {
	httpClient *http.Client
	endpoint   string
}

func (client *DefaultClient) GetEndpoint() string {
	return client.endpoint
}

const defaultLogstashEndpoint = "http://localhost:9600"

// NewClient returns a new instance of the DefaultClient configured with the given endpoint
func NewClient(endpoint string) Client {
	if endpoint == "" {
		endpoint = defaultLogstashEndpoint
	}

	return &DefaultClient{
		httpClient: &http.Client{},
		endpoint:   endpoint,
	}
}

func getMetrics[T any](ctx context.Context, client *http.Client, endpoint string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
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

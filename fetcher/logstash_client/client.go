package logstash_client

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

type Client interface {
	GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error)
	GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error)
}

type DefaultClient struct {
	httpClient *http.Client
	endpoint   string
}

const defaultLogstashEndpoint = "http://localhost:9600"

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

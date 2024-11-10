package logstash_client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
)

// Client is an interface for the Logstash client able to fetch data from the Logstash API
type Client interface {
	Name() string
	GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error)
	GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error)

	GetEndpoint() string
}

// DefaultClient is the default implementation of the Client interface
type DefaultClient struct {
	httpClient *http.Client
	endpoint   string
	name       string
}

func (client *DefaultClient) GetEndpoint() string {
	return client.endpoint
}

func (client *DefaultClient) Name() string {
	if client.name == "" {
		return client.convertHostnameToName()
	}

	return client.name
}

// convertHostnameToName converts a hostname to a name, that contains only alphanumeric characters and underscores
// Example: http://localhost:9600 -> localhost_9600
func (client *DefaultClient) convertHostnameToName() string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(client.endpoint, "_"), "_")
}

const defaultLogstashEndpoint = "http://localhost:9600"

// NewClient returns a new instance of the DefaultClient configured with the given endpoint
func NewClient(endpoint string, httpInsecure bool, name string) Client {
	if endpoint == "" {
		endpoint = defaultLogstashEndpoint
	}

	return &DefaultClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: httpInsecure},
			},
		},
		endpoint: endpoint,
		name:     name,
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

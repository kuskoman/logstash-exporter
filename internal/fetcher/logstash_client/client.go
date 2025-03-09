package logstash_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
	customtls "github.com/kuskoman/logstash-exporter/pkg/tls"
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
func NewClient(endpoint string, name string) Client {
	if endpoint == "" {
		endpoint = defaultLogstashEndpoint
	}

	// Create a basic HTTP client with default transport
	httpClient := &http.Client{
		Transport: http.DefaultTransport,
	}

	return &DefaultClient{
		httpClient: httpClient,
		endpoint:   endpoint,
		name:       name,
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

// NewClientWithTLS creates a new client with advanced TLS configuration
func NewClientWithTLS(baseUrl string, timeout time.Duration, caFile, serverName string, insecureSkipVerify bool) (*DefaultClient, error) {
	// Use the TLS package to configure the HTTP client
	httpClient, err := customtls.ConfigureHTTPClientWithTLS(timeout, caFile, serverName, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	return &DefaultClient{
		httpClient: httpClient,
		endpoint:   baseUrl,
	}, nil
}

// NewClientWithBasicAuth creates a new client with basic authentication
func NewClientWithBasicAuth(baseUrl string, timeout time.Duration, username, password string,
	caFile, serverName string, insecureSkipVerify bool) (*DefaultClient, error) {

	// Create a client with TLS configuration
	client, err := NewClientWithTLS(baseUrl, timeout, caFile, serverName, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	// Add basic auth using the TLS package
	client.httpClient = customtls.ConfigureBasicAuth(client.httpClient, username, password)

	return client, nil
}

// Get performs an HTTP GET request to the given path and returns the response body
func (c *DefaultClient) Get(path string) ([]byte, error) {
	url := c.endpoint + path
	slog.Debug("fetching data from logstash", "url", url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// NewClientWithHTTPClient returns a new instance of the DefaultClient configured with a provided HTTP client
func NewClientWithHTTPClient(endpoint string, httpClient *http.Client, name string) Client {
	return &DefaultClient{
		httpClient: httpClient,
		endpoint:   endpoint,
		name:       name,
	}
}

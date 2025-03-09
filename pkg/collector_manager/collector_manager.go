package collector_manager

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"

	"github.com/kuskoman/logstash-exporter/internal/collectors/nodeinfo"
	"github.com/kuskoman/logstash-exporter/internal/collectors/nodestats"
	"github.com/kuskoman/logstash-exporter/internal/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/kuskoman/logstash-exporter/pkg/tls"
)

// Collector is an interface that defines methods for collecting metrics
type Collector interface {
	// Collect is called by the Prometheus registry when collecting metrics
	Collect(context.Context, chan<- prometheus.Metric) (err error)
}

// CollectorManager is a collector that executes all other collectors
type CollectorManager struct {
	collectors      map[string]Collector
	scrapeDurations *prometheus.SummaryVec
	httpTimeout     time.Duration
	mu              sync.RWMutex
	instancesMap    map[string]*config.LogstashInstance // Used for dynamic instance management
}

func getClientsForEndpoints(instances []*config.LogstashInstance, timeout time.Duration) []logstash_client.Client {
	clients := make([]logstash_client.Client, len(instances))

	for i, instance := range instances {
		var client logstash_client.Client
		var err error

		// Create an HTTP client based on the instance configuration
		httpClient, err := tls.ConfigureHTTPClientFromLogstashInstance(instance, timeout)
		if err != nil {
			slog.Error("Failed to configure TLS client", "error", err)
			// Fall back to standard client
			client = logstash_client.NewClient(instance.Host, instance.HttpInsecure, instance.Name)
			clients[i] = client
			continue
		}

		// If there's basic auth configuration, add it
		if instance.BasicAuth != nil {
			password, err := instance.BasicAuth.GetPassword()
			if err != nil {
				slog.Error("Failed to get authentication password", "error", err)
				// Continue with a standard client as fallback
				client = logstash_client.NewClient(instance.Host, instance.HttpInsecure, instance.Name)
				clients[i] = client
				continue
			}

			// Add basic auth to the HTTP client
			httpClient = tls.ConfigureBasicAuth(httpClient, instance.BasicAuth.Username, password)
		}

		// Create a client with the configured HTTP client
		client = logstash_client.NewClientWithHTTPClient(instance.Host, httpClient, instance.Name)
		clients[i] = client
	}

	return clients
}

// NewCollectorManager creates a new CollectorManager with the provided logstash instances and http timeout
func NewCollectorManager(instances []*config.LogstashInstance, timeout time.Duration) *CollectorManager {
	// Build the instance map
	instancesMap := make(map[string]*config.LogstashInstance)
	for _, instance := range instances {
		instanceID := instance.Name
		if instanceID == "" {
			instanceID = instance.Host
		}
		instancesMap[instanceID] = instance
	}

	clients := getClientsForEndpoints(instances, timeout)

	collectors := getCollectors(clients)

	scrapeDurations := getScrapeDurationsCollector()
	prometheus.Unregister(version.NewCollector("logstash_exporter"))
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))

	return &CollectorManager{
		collectors:      collectors,
		scrapeDurations: scrapeDurations,
		httpTimeout:     timeout,
		instancesMap:    instancesMap,
	}
}

func getCollectors(clients []logstash_client.Client) map[string]Collector {
	collectors := make(map[string]Collector)
	collectors["nodeinfo"] = nodeinfo.NewNodeinfoCollector(clients)
	collectors["nodestats"] = nodestats.NewNodestatsCollector(clients)
	return collectors
}

// Collect executes all collectors and sends the collected metrics to the provided channel.
// It also sends the duration of the collection to the scrapeDurations collector.
func (manager *CollectorManager) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), manager.httpTimeout)
	defer cancel()

	// Create a safe copy of collectors to avoid concurrent map access
	manager.mu.RLock()
	collectors := make(map[string]Collector, len(manager.collectors))
	for name, collector := range manager.collectors {
		collectors[name] = collector
	}
	manager.mu.RUnlock()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(collectors))
	for name, collector := range collectors {
		go func(name string, collector Collector) {
			slog.Debug("executing collector", "name", name)
			manager.executeCollector(name, ctx, collector, ch)
			slog.Debug("collector finished", "name", name)
			waitGroup.Done()
		}(name, collector)
	}
	waitGroup.Wait()
}

// Describe runs the describe process for the scrapeDurations collector
func (manager *CollectorManager) Describe(ch chan<- *prometheus.Desc) {
	manager.scrapeDurations.Describe(ch)
}

func (manager *CollectorManager) executeCollector(name string, ctx context.Context, collector Collector, ch chan<- prometheus.Metric) {
	executionStart := time.Now()
	err := collector.Collect(ctx, ch)
	executionDuration := time.Since(executionStart)
	var executionStatus string

	if err != nil {
		slog.Error("executor failed", "name", name, "duration", executionDuration, "err", err)
		executionStatus = "error"
	} else {
		slog.Debug("executor succeeded", "name", name, "duration", executionDuration)
		executionStatus = "success"
	}

	manager.scrapeDurations.WithLabelValues(name, executionStatus).Observe(executionDuration.Seconds())
}

func getScrapeDurationsCollector() *prometheus.SummaryVec {
	scrapeDurations := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: config.PrometheusNamespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "logstash_exporter: Duration of a scrape job.",
		},
		[]string{"collector", "result"},
	)

	return scrapeDurations
}

// AddInstance adds a new Logstash instance to be monitored
func (manager *CollectorManager) AddInstance(id string, instance *config.LogstashInstance) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Check if already exists
	if _, exists := manager.instancesMap[id]; exists {
		slog.Debug("instance already exists, updating", "id", id)
	}

	// Add to instance map
	manager.instancesMap[id] = instance

	// Regenerate collectors with updated instances
	var instances []*config.LogstashInstance
	for _, inst := range manager.instancesMap {
		instances = append(instances, inst)
	}

	clients := getClientsForEndpoints(instances, manager.httpTimeout)
	manager.collectors = getCollectors(clients)
}

// RemoveInstance removes a Logstash instance from monitoring
func (manager *CollectorManager) RemoveInstance(id string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Check if exists
	if _, exists := manager.instancesMap[id]; !exists {
		slog.Debug("instance does not exist, nothing to remove", "id", id)
		return
	}

	// Remove from instance map
	delete(manager.instancesMap, id)

	// Regenerate collectors with updated instances
	var instances []*config.LogstashInstance
	for _, inst := range manager.instancesMap {
		instances = append(instances, inst)
	}

	clients := getClientsForEndpoints(instances, manager.httpTimeout)
	manager.collectors = getCollectors(clients)
}

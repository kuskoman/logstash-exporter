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
}

func getClientsForEndpoints(instances []*config.LogstashInstance) []logstash_client.Client {
	clients := make([]logstash_client.Client, len(instances))

	for i, instance := range instances {
		clients[i] = logstash_client.NewClient(instance.Host, instance.HttpInsecure, instance.Name)
	}

	return clients
}

// NewCollectorManager creates a new CollectorManager with the provided logstash instances and http timeout
func NewCollectorManager(instances []*config.LogstashInstance, httpTimeout time.Duration) *CollectorManager {
	clients := getClientsForEndpoints(instances)

	collectors := getCollectors(clients)

	scrapeDurations := getScrapeDurationsCollector()
	prometheus.Unregister(version.NewCollector("logstash_exporter"))
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))

	return &CollectorManager{collectors: collectors, scrapeDurations: scrapeDurations, httpTimeout: httpTimeout}
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

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(manager.collectors))
	for name, collector := range manager.collectors {
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

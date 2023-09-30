package collectors

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kuskoman/logstash-exporter/collectors/nodeinfo"
	"github.com/kuskoman/logstash-exporter/collectors/nodestats"
	"github.com/kuskoman/logstash-exporter/config"
	logstashclient "github.com/kuskoman/logstash-exporter/fetcher/logstash_client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

type Collector interface {
	Collect(context.Context, chan<- prometheus.Metric) (err error)
}

// CollectorManager is a collector that executes all other collectors
type CollectorManager struct {
	collectors      map[string]Collector
	scrapeDurations *prometheus.SummaryVec
}

func getClientsForEndpoints(endpoints []string) []logstashclient.Client {
	clients := make([]logstashclient.Client, len(endpoints))

	for i, endpoint := range endpoints {
		clients[i] = logstashclient.NewClient(endpoint)
	}

	return clients
}

func NewCollectorManager(endpoints []string) *CollectorManager {
	clients := getClientsForEndpoints(endpoints)

	collectors := getCollectors(clients)

	scrapeDurations := getScrapeDurationsCollector()
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))

	return &CollectorManager{collectors: collectors, scrapeDurations: scrapeDurations}
}

func getCollectors(clients []logstashclient.Client) map[string]Collector {
	collectors := make(map[string]Collector)
	collectors["nodeinfo"] = nodeinfo.NewNodeinfoCollector(clients)
	collectors["nodestats"] = nodestats.NewNodestatsCollector(clients) // TODO: support multiple clients
	return collectors
}

// Collect executes all collectors and sends the collected metrics to the provided channel.
// It also sends the duration of the collection to the scrapeDurations collector.
func (manager *CollectorManager) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), config.HttpTimeout)

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

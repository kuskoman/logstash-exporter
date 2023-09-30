package collectors

import (
	"context"
	"sync"
	"time"

	"log/slog"

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
	collectors      map[string]map[string]Collector // map[endpoint]map[collectorName]Collector
	scrapeDurations *prometheus.SummaryVec
}

func NewCollectorManager(endpoints []string) *CollectorManager {
	collectors := make(map[string]map[string]Collector)

	for _, endpoint := range endpoints {
		client := logstashclient.NewClient(endpoint)
		collectors[endpoint] = getCollectors(client)
	}

	scrapeDurations := getScrapeDurationsCollector()
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))

	return &CollectorManager{collectors: collectors, scrapeDurations: scrapeDurations}
}

func getCollectors(client logstashclient.Client) map[string]Collector {
	collectors := make(map[string]Collector)
	collectors["nodeinfo"] = nodeinfo.NewNodeinfoCollector(client)
	collectors["nodestats"] = nodestats.NewNodestatsCollector(client)
	return collectors
}

func (manager *CollectorManager) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), config.HttpTimeout)
	defer cancel()

	waitGroup := sync.WaitGroup{}
	for endpoint, endpointCollectors := range manager.collectors {
		waitGroup.Add(len(endpointCollectors))
		for name, collector := range endpointCollectors {
			go func(name string, endpoint string, collector Collector) {
				manager.executeCollector(name, endpoint, ctx, collector, ch)
				waitGroup.Done()
			}(name, endpoint, collector)
		}
	}
	waitGroup.Wait()
}

func (manager *CollectorManager) Describe(ch chan<- *prometheus.Desc) {
	manager.scrapeDurations.Describe(ch)
}

func (manager *CollectorManager) executeCollector(name string, endpoint string, ctx context.Context, collector Collector, ch chan<- prometheus.Metric) {
	executionStart := time.Now()
	err := collector.Collect(ctx, ch)
	executionDuration := time.Since(executionStart)
	var executionStatus string

	if err != nil {
		slog.Error("executor failed", "name", name, "endpoint", endpoint, "duration", executionDuration, "err", err)
		executionStatus = "error"
	} else {
		slog.Debug("executor succeeded", "name", name, "endpoint", endpoint, "duration", executionDuration)
		executionStatus = "success"
	}

	manager.scrapeDurations.WithLabelValues(name, endpoint, executionStatus).Observe(executionDuration.Seconds())
}

func getScrapeDurationsCollector() *prometheus.SummaryVec {
	scrapeDurations := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: config.PrometheusNamespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "logstash_exporter: Duration of a scrape job.",
		},
		[]string{"collector", "endpoint", "result"},
	)

	return scrapeDurations
}

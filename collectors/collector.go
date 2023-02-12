package collectors

import (
	"log"
	"sync"
	"time"

	"github.com/kuskoman/logstash-exporter/collectors/nodestats"
	"github.com/kuskoman/logstash-exporter/config"
	logstashclient "github.com/kuskoman/logstash-exporter/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/httphandler"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

type Collector interface {
	Collect(ch chan<- prometheus.Metric) (err error)
}

type CollectorManager struct {
	collectors      map[string]Collector
	scrapeDurations *prometheus.SummaryVec
}

func NewCollectorManager(endpoint string) *CollectorManager {
	httpHandler := httphandler.GetDefaultHTTPHandler(endpoint)
	client := logstashclient.NewClient(httpHandler)

	collectors := make(map[string]Collector)
	collectors["nodeinfo"] = nodestats.NewNodestatsCollector(client)

	scrapeDurations := getScrapeDurationsCollector()
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))

	return &CollectorManager{collectors: collectors, scrapeDurations: scrapeDurations}
}

func (manager *CollectorManager) Collect(ch chan<- prometheus.Metric) {
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(manager.collectors))
	for name, collector := range manager.collectors {
		go func(name string, collector Collector) {
			manager.executeCollector(name, collector, ch)
			waitGroup.Done()
		}(name, collector)
	}
	waitGroup.Wait()
}

func (manager *CollectorManager) executeCollector(name string, collector Collector, ch chan<- prometheus.Metric) {
	executionStart := time.Now()
	err := collector.Collect(ch)
	executionDuration := time.Since(executionStart)
	var executionStatus string

	if err != nil {
		log.Printf("executor %s failed after %s: %s", name, executionDuration, err.Error())
		executionStatus = "error"
	} else {
		log.Printf("executor %s succeeded after %s", name, executionDuration)
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

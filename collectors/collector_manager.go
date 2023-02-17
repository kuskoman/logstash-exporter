package collectors

import (
	"log"
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
	Collect(ch chan<- prometheus.Metric) (err error)
}

type CollectorManager struct {
	collectors      map[string]Collector
	scrapeDurations *prometheus.SummaryVec
}

func NewCollectorManager(endpoint string) *CollectorManager {
	client := logstashclient.NewClient(endpoint)

	collectors := getCollectors(client)

	scrapeDurations := getScrapeDurationsCollector()
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))

	return &CollectorManager{collectors: collectors, scrapeDurations: scrapeDurations}
}

func getCollectors(client logstashclient.Client) map[string]Collector {
	collectors := make(map[string]Collector)
	collectors["nodeinfo"] = nodeinfo.NewNodestatsCollector(client)
	collectors["nodestats"] = nodestats.NewNodestatsCollector(client)
	return collectors
}

func (manager *CollectorManager) Collect(ch chan<- prometheus.Metric) {
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(manager.collectors))
	for name, collector := range manager.collectors {
		go func(name string, collector Collector) {
			log.Printf("executing collector %s", name)
			manager.executeCollector(name, collector, ch)
			log.Printf("collector %s finished", name)
			waitGroup.Done()
		}(name, collector)
	}
	waitGroup.Wait()
}

func (manager *CollectorManager) Describe(ch chan<- *prometheus.Desc) {
	manager.scrapeDurations.Describe(ch)
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

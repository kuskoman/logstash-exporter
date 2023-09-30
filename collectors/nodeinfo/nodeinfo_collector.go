package nodeinfo

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/config"
	logstashclient "github.com/kuskoman/logstash-exporter/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
)

// NodeinfoCollector is a custom collector for the /_node/stats endpoint
type NodeinfoCollector struct {
	client logstashclient.Client

	NodeInfos  *prometheus.Desc
	BuildInfos *prometheus.Desc

	Up *prometheus.Desc

	PipelineWorkers    *prometheus.Desc
	PipelineBatchSize  *prometheus.Desc
	PipelineBatchDelay *prometheus.Desc

	Status *prometheus.Desc
}

func NewNodeinfoCollector(client logstashclient.Client) *NodeinfoCollector {
	const subsystem = "info"
	namespace := config.PrometheusNamespace
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodeinfoCollector{
		client: client,
		NodeInfos: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "node"),
			"A metric with a constant '1' value labeled by node name, version, host, http_address, and id of the logstash instance.",
			[]string{"name", "version", "http_address", "host", "id"},
			nil,
		),
		BuildInfos: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "build"),
			"A metric with a constant '1' value labeled by build date, sha, and snapshot of the logstash instance.",
			[]string{"date", "sha", "snapshot"},
			nil,
		),

		Up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"A metric that returns 1 if the node is up, 0 otherwise.",
			nil,
			nil,
		),

		PipelineWorkers: descHelper.NewDesc("pipeline_workers",
			"Number of worker threads that will process pipeline events.",
		),
		PipelineBatchSize: descHelper.NewDesc("pipeline_batch_size",
			"Number of events to retrieve from the input queue before sending to the filter and output stages.",
		),
		PipelineBatchDelay: descHelper.NewDesc("pipeline_batch_delay",
			"Amount of time to wait for events to fill the batch before sending to the filter and output stages.",
		),

		Status: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "status"),
			"A metric with a constant '1' value labeled by status.",
			[]string{"status"},
			nil,
		),
	}
}

func (c *NodeinfoCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	nodeInfo, err := c.client.GetNodeInfo(ctx)
	if err != nil {
		ch <- c.getUpStatus(nodeInfo, err)

		return err
	}

	ch <- prometheus.MustNewConstMetric(
		c.NodeInfos,
		prometheus.CounterValue,
		float64(1),
		nodeInfo.Name,
		nodeInfo.Version,
		nodeInfo.Host,
		nodeInfo.HTTPAddress,
		nodeInfo.ID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BuildInfos,
		prometheus.CounterValue,
		float64(1),
		nodeInfo.BuildDate,
		nodeInfo.BuildSHA,
		strconv.FormatBool(nodeInfo.BuildSnapshot),
	)

	ch <- prometheus.MustNewConstMetric(
		c.Up,
		prometheus.GaugeValue,
		float64(1),
	)

	ch <- prometheus.MustNewConstMetric(
		c.PipelineWorkers,
		prometheus.CounterValue,
		float64(nodeInfo.Pipeline.Workers),
	)

	ch <- prometheus.MustNewConstMetric(
		c.PipelineBatchSize,
		prometheus.CounterValue,
		float64(nodeInfo.Pipeline.BatchSize),
	)

	ch <- prometheus.MustNewConstMetric(
		c.PipelineBatchDelay,
		prometheus.CounterValue,
		float64(nodeInfo.Pipeline.BatchDelay),
	)

	ch <- prometheus.MustNewConstMetric(
		c.Status,
		prometheus.CounterValue,
		float64(1),
		nodeInfo.Status,
	)

	return nil
}

func (c *NodeinfoCollector) getUpStatus(nodeinfo *responses.NodeInfoResponse, err error) prometheus.Metric {
	status := 1
	if err != nil {
		status = 0
	} else if nodeinfo.Status != "green" && nodeinfo.Status != "yellow" {
		status = 0
	}

	return prometheus.MustNewConstMetric(
		c.Up,
		prometheus.GaugeValue,
		float64(status),
	)
}

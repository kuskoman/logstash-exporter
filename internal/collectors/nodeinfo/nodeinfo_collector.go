package nodeinfo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/internal/prometheus_helper"
	"github.com/kuskoman/logstash-exporter/pkg/config"
)

const subsystem = "info"

var (
	namespace = config.PrometheusNamespace
)

// NodeinfoCollector is a custom collector for the /_node/stats endpoint
type NodeinfoCollector struct {
	clients []logstash_client.Client

	NodeInfos  *prometheus.Desc
	BuildInfos *prometheus.Desc

	Up *prometheus.Desc

	PipelineWorkers    *prometheus.Desc
	PipelineBatchSize  *prometheus.Desc
	PipelineBatchDelay *prometheus.Desc

	Status *prometheus.Desc
}

func NewNodeinfoCollector(clients []logstash_client.Client) *NodeinfoCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodeinfoCollector{
		clients: clients,
		NodeInfos: descHelper.NewDesc("node",
			"A metric with a constant '1' value labeled by node name, version, host, http_address, and id of the logstash instance.",
			"name", "version", "http_address", "host", "id",
		),
		BuildInfos: descHelper.NewDesc("build",
			"A metric with a constant '1' value labeled by build date, sha, and snapshot of the logstash instance.",
			"date", "sha", "snapshot"),

		Up: descHelper.NewDesc("up",
			"A metric that returns 1 if the node is up, 0 otherwise."),
		PipelineWorkers: descHelper.NewDesc("pipeline_workers",
			"Number of worker threads that will process pipeline events.",
		),
		PipelineBatchSize: descHelper.NewDesc("pipeline_batch_size",
			"Number of events to retrieve from the input queue before sending to the filter and output stages."),
		PipelineBatchDelay: descHelper.NewDesc("pipeline_batch_delay",
			"Amount of time to wait for events to fill the batch before sending to the filter and output stages."),

		Status: descHelper.NewDesc("status",
			"A metric with a constant '1' value labeled by status.",
			"status"),
	}
}

func (c *NodeinfoCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	wg := sync.WaitGroup{}
	wg.Add(len(c.clients))

	errorChannel := make(chan error, len(c.clients))

	for _, client := range c.clients {
		go func(client logstash_client.Client) {
			err := c.collectSingleInstance(client, ctx, ch)
			if err != nil {
				errorChannel <- err
			}
			wg.Done()
		}(client)
	}

	wg.Wait()
	close(errorChannel)

	if len(errorChannel) == 0 {
		return nil
	}

	if len(errorChannel) == 1 {
		return <-errorChannel
	}

	errorString := fmt.Sprintf("encountered %d errors while collecting nodeinfo metrics", len(errorChannel))
	for err := range errorChannel {
		errorString += fmt.Sprintf("\n\t%s", err.Error())
	}

	return errors.New(errorString)
}

func (collector *NodeinfoCollector) collectSingleInstance(client logstash_client.Client, ctx context.Context, ch chan<- prometheus.Metric) error {
	endpoint := client.GetEndpoint()
	metricsHelper := prometheus_helper.SimpleMetricsHelper{Channel: ch, Labels: []string{endpoint}}

	nodeInfo, err := client.GetNodeInfo(ctx)
	if err != nil {
		status := collector.getUpStatus(nodeInfo, err)

		// ***** UP *****
		metricsHelper.NewIntMetric(collector.Up, prometheus.GaugeValue, status)
		// **************

		return err
	}

	// ***** NODE *****
	metricsHelper.Labels = []string{nodeInfo.Name, nodeInfo.Version, nodeInfo.Host, nodeInfo.HTTPAddress, nodeInfo.ID, endpoint}
	metricsHelper.NewIntMetric(collector.NodeInfos, prometheus.CounterValue, 1)
	// ****************

	// ***** BUILD *****
	metricsHelper.Labels = []string{nodeInfo.BuildDate, nodeInfo.BuildSHA, strconv.FormatBool(nodeInfo.BuildSnapshot), endpoint}
	metricsHelper.NewIntMetric(collector.BuildInfos, prometheus.CounterValue, 1)
	// *****************

	metricsHelper.Labels = []string{endpoint}

	// ***** UP *****
	metricsHelper.NewIntMetric(collector.Up, prometheus.GaugeValue, 1)
	// **************

	// ***** PIPELINE *****
	metricsHelper.NewIntMetric(collector.PipelineWorkers, prometheus.GaugeValue, nodeInfo.Pipeline.Workers)
	metricsHelper.NewIntMetric(collector.PipelineBatchSize, prometheus.GaugeValue, nodeInfo.Pipeline.BatchSize)
	metricsHelper.NewIntMetric(collector.PipelineBatchDelay, prometheus.GaugeValue, nodeInfo.Pipeline.BatchDelay)
	// ********************

	// ***** STATUS *****
	metricsHelper.Labels = []string{nodeInfo.Status, endpoint}
	metricsHelper.NewIntMetric(collector.Status, prometheus.CounterValue, 1)
	// ******************

	return nil
}

func (c *NodeinfoCollector) getUpStatus(nodeinfo *responses.NodeInfoResponse, err error) int {
	status := 1
	if err != nil {
		status = 0
	} else if nodeinfo.Status != "green" && nodeinfo.Status != "yellow" {
		status = 0
	}

	return status
}

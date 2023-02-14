package nodestats

import (
	"log"

	"github.com/kuskoman/logstash-exporter/config"
	logstashclient "github.com/kuskoman/logstash-exporter/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/helpers"
	"github.com/prometheus/client_golang/prometheus"
)

const subsystem = "stats"

var (
	namespace  = config.PrometheusNamespace
	descHelper = helpers.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}
)

type NodestatsCollector struct {
	client               logstashclient.Client
	pipelineSubcollector *PipelineSubcollector

	JvmThreadsCount     *prometheus.Desc
	JvmThreadsPeakCount *prometheus.Desc

	JvmMemHeapUsedPercent       *prometheus.Desc
	JvmMemHeapCommittedBytes    *prometheus.Desc
	JvmMemHeapMaxBytes          *prometheus.Desc
	JvmMemHeapUsedBytes         *prometheus.Desc
	JvmMemNonHeapCommittedBytes *prometheus.Desc

	JvmUptimeMillis *prometheus.Desc

	ProcessOpenFileDescriptors *prometheus.Desc
	ProcessMaxFileDescriptors  *prometheus.Desc
	ProcessCpuPercent          *prometheus.Desc
	ProcessCpuTotalMillis      *prometheus.Desc
	ProcessMemTotalVirtual     *prometheus.Desc

	ReloadSuccesses *prometheus.Desc
	ReloadFailures  *prometheus.Desc

	QueueEventsCount *prometheus.Desc
}

func NewNodestatsCollector(client logstashclient.Client) *NodestatsCollector {

	return &NodestatsCollector{
		client: client,

		pipelineSubcollector: NewPipelineSubcollector(),

		JvmThreadsCount:     descHelper.NewDesc("jvm_threads_count"),
		JvmThreadsPeakCount: descHelper.NewDesc("jvm_threads_peak_count"),

		JvmMemHeapUsedPercent:       descHelper.NewDesc("jvm_mem_heap_used_percent"),
		JvmMemHeapCommittedBytes:    descHelper.NewDesc("jvm_mem_heap_committed_bytes"),
		JvmMemHeapMaxBytes:          descHelper.NewDesc("jvm_mem_heap_max_bytes"),
		JvmMemHeapUsedBytes:         descHelper.NewDesc("jvm_mem_heap_used_bytes"),
		JvmMemNonHeapCommittedBytes: descHelper.NewDesc("jvm_mem_non_heap_committed_bytes"),

		JvmUptimeMillis: descHelper.NewDesc("jvm_uptime_millis"),

		ProcessOpenFileDescriptors: descHelper.NewDesc("process_open_file_descriptors"),
		ProcessMaxFileDescriptors:  descHelper.NewDesc("process_max_file_descriptors"),
		ProcessCpuPercent:          descHelper.NewDesc("process_cpu_percent"),
		ProcessCpuTotalMillis:      descHelper.NewDesc("process_cpu_total_millis"),
		ProcessMemTotalVirtual:     descHelper.NewDesc("process_mem_total_virtual"),

		ReloadSuccesses: descHelper.NewDesc("reload_successes"),
		ReloadFailures:  descHelper.NewDesc("reload_failures"),

		QueueEventsCount: descHelper.NewDesc("queue_events_count"),
	}
}

func (c *NodestatsCollector) Collect(ch chan<- prometheus.Metric) error {
	nodeStats, err := c.client.GetNodeStats()
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(c.JvmThreadsCount, prometheus.GaugeValue, float64(nodeStats.Jvm.Threads.Count))
	ch <- prometheus.MustNewConstMetric(c.JvmThreadsPeakCount, prometheus.GaugeValue, float64(nodeStats.Jvm.Threads.PeakCount))

	ch <- prometheus.MustNewConstMetric(c.JvmMemHeapUsedPercent, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.HeapUsedPercent))
	ch <- prometheus.MustNewConstMetric(c.JvmMemHeapCommittedBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.HeapCommittedInBytes))
	ch <- prometheus.MustNewConstMetric(c.JvmMemHeapMaxBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.HeapMaxInBytes))
	ch <- prometheus.MustNewConstMetric(c.JvmMemHeapUsedBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.HeapUsedInBytes))
	ch <- prometheus.MustNewConstMetric(c.JvmMemNonHeapCommittedBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.NonHeapCommittedInBytes))

	ch <- prometheus.MustNewConstMetric(c.JvmUptimeMillis, prometheus.GaugeValue, float64(nodeStats.Jvm.UptimeInMillis))

	ch <- prometheus.MustNewConstMetric(c.ProcessOpenFileDescriptors, prometheus.GaugeValue, float64(nodeStats.Process.OpenFileDescriptors))
	ch <- prometheus.MustNewConstMetric(c.ProcessMaxFileDescriptors, prometheus.GaugeValue, float64(nodeStats.Process.MaxFileDescriptors))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuPercent, prometheus.GaugeValue, float64(nodeStats.Process.CPU.Percent))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuTotalMillis, prometheus.GaugeValue, float64(nodeStats.Process.CPU.TotalInMillis))
	ch <- prometheus.MustNewConstMetric(c.ProcessMemTotalVirtual, prometheus.GaugeValue, float64(nodeStats.Process.Mem.TotalVirtualInBytes))

	ch <- prometheus.MustNewConstMetric(c.ReloadSuccesses, prometheus.GaugeValue, float64(nodeStats.Reloads.Successes))
	ch <- prometheus.MustNewConstMetric(c.ReloadFailures, prometheus.GaugeValue, float64(nodeStats.Reloads.Failures))

	ch <- prometheus.MustNewConstMetric(c.QueueEventsCount, prometheus.GaugeValue, float64(nodeStats.Queue.EventsCount))

	for pipelineId, pipelineStats := range nodeStats.Pipelines {
		err = c.pipelineSubcollector.Collect(&pipelineStats, pipelineId, ch)
		if err != nil {
			log.Printf("Error collecting pipeline %s, stats: %s", pipelineId, err.Error())
		}
		// we don't want to stop collecting other pipelines if one of them fails
	}

	// last error is returned
	return err
}

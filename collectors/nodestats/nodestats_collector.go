package nodestats

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/config"
	logstashclient "github.com/kuskoman/logstash-exporter/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
)

const subsystem = "stats"

var (
	namespace = config.PrometheusNamespace
)

// NodestatsCollector is a custom collector for the /_node/stats endpoint
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

	JvmMemPoolPeakUsedInBytes  *prometheus.Desc
	JvmMemPoolUsedInBytes      *prometheus.Desc
	JvmMemPoolPeakMaxInBytes   *prometheus.Desc
	JvmMemPoolMaxInBytes       *prometheus.Desc
	JvmMemPoolCommittedInBytes *prometheus.Desc

	JvmUptimeMillis *prometheus.Desc

	ProcessOpenFileDescriptors    *prometheus.Desc
	ProcessMaxFileDescriptors     *prometheus.Desc
	ProcessCpuPercent             *prometheus.Desc
	ProcessCpuTotalMillis         *prometheus.Desc
	ProcessCpuLoadAverageOneM     *prometheus.Desc
	ProcessCpuLoadAverageFiveM    *prometheus.Desc
	ProcessCpuLoadAverageFifteenM *prometheus.Desc
	ProcessMemTotalVirtual        *prometheus.Desc

	ReloadSuccesses *prometheus.Desc
	ReloadFailures  *prometheus.Desc

	QueueEventsCount *prometheus.Desc
}

func NewNodestatsCollector(client logstashclient.Client) *NodestatsCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodestatsCollector{
		client: client,

		pipelineSubcollector: NewPipelineSubcollector(),

		JvmThreadsCount:     descHelper.NewDescWithHelp("jvm_threads_count", "Number of live threads including both daemon and non-daemon threads."),
		JvmThreadsPeakCount: descHelper.NewDescWithHelp("jvm_threads_peak_count", "Peak live thread count since the Java virtual machine started or peak was reset."),

		JvmMemHeapUsedPercent:       descHelper.NewDescWithHelp("jvm_mem_heap_used_percent", "Percentage of the heap memory that is used."),
		JvmMemHeapCommittedBytes:    descHelper.NewDescWithHelp("jvm_mem_heap_committed_bytes", "Amount of heap memory in bytes that is committed for the Java virtual machine to use."),
		JvmMemHeapMaxBytes:          descHelper.NewDescWithHelp("jvm_mem_heap_max_bytes", "Maximum amount of heap memory in bytes that can be used for memory management."),
		JvmMemHeapUsedBytes:         descHelper.NewDescWithHelp("jvm_mem_heap_used_bytes", "Amount of used heap memory in bytes."),
		JvmMemNonHeapCommittedBytes: descHelper.NewDescWithHelp("jvm_mem_non_heap_committed_bytes", "Amount of non-heap memory in bytes that is committed for the Java virtual machine to use."),

		JvmMemPoolPeakUsedInBytes: descHelper.NewDescWithHelpAndLabels(
			"jvm_mem_pool_peak_used_bytes", "Peak used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolUsedInBytes: descHelper.NewDescWithHelpAndLabels(
			"jvm_mem_pool_used_bytes", "Currently used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolPeakMaxInBytes: descHelper.NewDescWithHelpAndLabels(
			"jvm_mem_pool_peak_max_bytes", "Highest value of bytes that were used in a given JVM memory pool.", "pool"),
		JvmMemPoolMaxInBytes: descHelper.NewDescWithHelpAndLabels(
			"jvm_mem_pool_max_bytes", "Maximum amount of bytes that can be used in a given JVM memory pool.", "pool"),
		JvmMemPoolCommittedInBytes: descHelper.NewDescWithHelpAndLabels(
			"jvm_mem_pool_committed_bytes", "Amount of bytes that are committed for the Java virtual machine to use in a given JVM memory pool.", "pool"),

		JvmUptimeMillis: descHelper.NewDescWithHelp("jvm_uptime_millis", "Uptime of the JVM in milliseconds."),

		ProcessOpenFileDescriptors:    descHelper.NewDescWithHelp("process_open_file_descriptors", "Number of currently open file descriptors."),
		ProcessMaxFileDescriptors:     descHelper.NewDescWithHelp("process_max_file_descriptors", "Limit of open file descriptors."),
		ProcessCpuPercent:             descHelper.NewDescWithHelp("process_cpu_percent", "CPU usage of the process."),
		ProcessCpuTotalMillis:         descHelper.NewDescWithHelp("process_cpu_total_millis", "Total CPU time used by the process."),
		ProcessCpuLoadAverageOneM:     descHelper.NewDescWithHelp("process_cpu_load_average_1m", "Total 1m system load average."),
		ProcessCpuLoadAverageFiveM:    descHelper.NewDescWithHelp("process_cpu_load_average_5m", "Total 5m system load average."),
		ProcessCpuLoadAverageFifteenM: descHelper.NewDescWithHelp("process_cpu_load_average_15m", "Total 15m system load average."),

		ProcessMemTotalVirtual: descHelper.NewDescWithHelp("process_mem_total_virtual", "Total virtual memory used by the process."),

		ReloadSuccesses: descHelper.NewDescWithHelp("reload_successes", "Number of successful reloads."),
		ReloadFailures:  descHelper.NewDescWithHelp("reload_failures", "Number of failed reloads."),

		QueueEventsCount: descHelper.NewDescWithHelp("queue_events_count", "Number of events in the queue."),
	}
}

func (c *NodestatsCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	nodeStats, err := c.client.GetNodeStats(ctx)
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

	// POOLS
	// young
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Young.PeakUsedInBytes), "young")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Young.UsedInBytes), "young")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Young.PeakMaxInBytes), "young")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Young.MaxInBytes), "young")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Young.CommittedInBytes), "young")
	// old
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Old.PeakUsedInBytes), "old")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Old.UsedInBytes), "old")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Old.PeakMaxInBytes), "old")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Old.MaxInBytes), "old")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Old.CommittedInBytes), "old")
	// survivor
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Survivor.PeakUsedInBytes), "survivor")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Survivor.UsedInBytes), "survivor")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Survivor.PeakMaxInBytes), "survivor")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Survivor.MaxInBytes), "survivor")
	ch <- prometheus.MustNewConstMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, float64(nodeStats.Jvm.Mem.Pools.Survivor.CommittedInBytes), "survivor")

	ch <- prometheus.MustNewConstMetric(c.JvmUptimeMillis, prometheus.GaugeValue, float64(nodeStats.Jvm.UptimeInMillis))

	ch <- prometheus.MustNewConstMetric(c.ProcessOpenFileDescriptors, prometheus.GaugeValue, float64(nodeStats.Process.OpenFileDescriptors))
	ch <- prometheus.MustNewConstMetric(c.ProcessMaxFileDescriptors, prometheus.GaugeValue, float64(nodeStats.Process.MaxFileDescriptors))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuPercent, prometheus.GaugeValue, float64(nodeStats.Process.CPU.Percent))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuTotalMillis, prometheus.GaugeValue, float64(nodeStats.Process.CPU.TotalInMillis))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuLoadAverageOneM, prometheus.GaugeValue, float64(nodeStats.Process.CPU.LoadAverage.OneM))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuLoadAverageFiveM, prometheus.GaugeValue, float64(nodeStats.Process.CPU.LoadAverage.FiveM))
	ch <- prometheus.MustNewConstMetric(c.ProcessCpuLoadAverageFifteenM, prometheus.GaugeValue, float64(nodeStats.Process.CPU.LoadAverage.FifteenM))
	ch <- prometheus.MustNewConstMetric(c.ProcessMemTotalVirtual, prometheus.GaugeValue, float64(nodeStats.Process.Mem.TotalVirtualInBytes))

	ch <- prometheus.MustNewConstMetric(c.ReloadSuccesses, prometheus.GaugeValue, float64(nodeStats.Reloads.Successes))
	ch <- prometheus.MustNewConstMetric(c.ReloadFailures, prometheus.GaugeValue, float64(nodeStats.Reloads.Failures))

	ch <- prometheus.MustNewConstMetric(c.QueueEventsCount, prometheus.GaugeValue, float64(nodeStats.Queue.EventsCount))

	for pipelineId, pipelineStats := range nodeStats.Pipelines {
		c.pipelineSubcollector.Collect(&pipelineStats, pipelineId, ch)
	}

	return nil
}

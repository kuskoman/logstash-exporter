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

	EventsIn                        *prometheus.Desc
	EventsFiltered                  *prometheus.Desc
	EventsOut                       *prometheus.Desc
	EventsDurationInMillis          *prometheus.Desc
	EventsQueuePushDurationInMillis *prometheus.Desc

	FlowInputCurrent              *prometheus.Desc
	FlowInputLifetime             *prometheus.Desc
	FlowFilterCurrent             *prometheus.Desc
	FlowFilterLifetime            *prometheus.Desc
	FlowOutputCurrent             *prometheus.Desc
	FlowOutputLifetime            *prometheus.Desc
	FlowQueueBackpressureCurrent  *prometheus.Desc
	FlowQueueBackpressureLifetime *prometheus.Desc
	FlowWorkerConcurrencyCurrent  *prometheus.Desc
	FlowWorkerConcurrencyLifetime *prometheus.Desc
}

func NewNodestatsCollector(client logstashclient.Client) *NodestatsCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodestatsCollector{
		client: client,

		pipelineSubcollector: NewPipelineSubcollector(),

		JvmThreadsCount:     descHelper.NewDescWithHelpAndLabels("jvm_threads_count", "Number of live threads including both daemon and non-daemon threads."),
		JvmThreadsPeakCount: descHelper.NewDescWithHelpAndLabels("jvm_threads_peak_count", "Peak live thread count since the Java virtual machine started or peak was reset."),

		JvmMemHeapUsedPercent:       descHelper.NewDescWithHelpAndLabels("jvm_mem_heap_used_percent", "Percentage of the heap memory that is used."),
		JvmMemHeapCommittedBytes:    descHelper.NewDescWithHelpAndLabels("jvm_mem_heap_committed_bytes", "Amount of heap memory in bytes that is committed for the Java virtual machine to use."),
		JvmMemHeapMaxBytes:          descHelper.NewDescWithHelpAndLabels("jvm_mem_heap_max_bytes", "Maximum amount of heap memory in bytes that can be used for memory management."),
		JvmMemHeapUsedBytes:         descHelper.NewDescWithHelpAndLabels("jvm_mem_heap_used_bytes", "Amount of used heap memory in bytes."),
		JvmMemNonHeapCommittedBytes: descHelper.NewDescWithHelpAndLabels("jvm_mem_non_heap_committed_bytes", "Amount of non-heap memory in bytes that is committed for the Java virtual machine to use."),

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

		JvmUptimeMillis: descHelper.NewDescWithHelpAndLabels("jvm_uptime_millis", "Uptime of the JVM in milliseconds."),

		ProcessOpenFileDescriptors:    descHelper.NewDescWithHelpAndLabels("process_open_file_descriptors", "Number of currently open file descriptors."),
		ProcessMaxFileDescriptors:     descHelper.NewDescWithHelpAndLabels("process_max_file_descriptors", "Limit of open file descriptors."),
		ProcessCpuPercent:             descHelper.NewDescWithHelpAndLabels("process_cpu_percent", "CPU usage of the process."),
		ProcessCpuTotalMillis:         descHelper.NewDescWithHelpAndLabels("process_cpu_total_millis", "Total CPU time used by the process."),
		ProcessCpuLoadAverageOneM:     descHelper.NewDescWithHelpAndLabels("process_cpu_load_average_1m", "Total 1m system load average."),
		ProcessCpuLoadAverageFiveM:    descHelper.NewDescWithHelpAndLabels("process_cpu_load_average_5m", "Total 5m system load average."),
		ProcessCpuLoadAverageFifteenM: descHelper.NewDescWithHelpAndLabels("process_cpu_load_average_15m", "Total 15m system load average."),

		ProcessMemTotalVirtual: descHelper.NewDescWithHelpAndLabels("process_mem_total_virtual", "Total virtual memory used by the process."),

		ReloadSuccesses: descHelper.NewDescWithHelpAndLabels("reload_successes", "Number of successful reloads."),
		ReloadFailures:  descHelper.NewDescWithHelpAndLabels("reload_failures", "Number of failed reloads."),

		QueueEventsCount: descHelper.NewDescWithHelpAndLabels("queue_events_count", "Number of events in the queue."),

		EventsIn:                        descHelper.NewDescWithHelpAndLabels("events_in", "Number of events received."),
		EventsFiltered:                  descHelper.NewDescWithHelpAndLabels("events_filtered", "Number of events filtered out."),
		EventsOut:                       descHelper.NewDescWithHelpAndLabels("events_out", "Number of events out."),
		EventsDurationInMillis:          descHelper.NewDescWithHelpAndLabels("events_duration_millis", "Duration of events processing in milliseconds."),
		EventsQueuePushDurationInMillis: descHelper.NewDescWithHelpAndLabels("events_queue_push_duration_millis", "Duration of events push to queue in milliseconds."),

		FlowInputCurrent:              descHelper.NewDescWithHelpAndLabels("flow_input_current", "Current number of events in the input queue."),
		FlowInputLifetime:             descHelper.NewDescWithHelpAndLabels("flow_input_lifetime", "Lifetime number of events in the input queue."),
		FlowFilterCurrent:             descHelper.NewDescWithHelpAndLabels("flow_filter_current", "Current number of events in the filter queue."),
		FlowFilterLifetime:            descHelper.NewDescWithHelpAndLabels("flow_filter_lifetime", "Lifetime number of events in the filter queue."),
		FlowOutputCurrent:             descHelper.NewDescWithHelpAndLabels("flow_output_current", "Current number of events in the output queue."),
		FlowOutputLifetime:            descHelper.NewDescWithHelpAndLabels("flow_output_lifetime", "Lifetime number of events in the output queue."),
		FlowQueueBackpressureCurrent:  descHelper.NewDescWithHelpAndLabels("flow_queue_backpressure_current", "Current number of events in the backpressure queue."),
		FlowQueueBackpressureLifetime: descHelper.NewDescWithHelpAndLabels("flow_queue_backpressure_lifetime", "Lifetime number of events in the backpressure queue."),
		FlowWorkerConcurrencyCurrent:  descHelper.NewDescWithHelpAndLabels("flow_worker_concurrency_current", "Current number of workers."),
		FlowWorkerConcurrencyLifetime: descHelper.NewDescWithHelpAndLabels("flow_worker_concurrency_lifetime", "Lifetime number of workers."),
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

	ch <- prometheus.MustNewConstMetric(c.EventsIn, prometheus.GaugeValue, float64(nodeStats.Events.In))
	ch <- prometheus.MustNewConstMetric(c.EventsFiltered, prometheus.GaugeValue, float64(nodeStats.Events.Filtered))
	ch <- prometheus.MustNewConstMetric(c.EventsOut, prometheus.GaugeValue, float64(nodeStats.Events.Out))
	ch <- prometheus.MustNewConstMetric(c.EventsDurationInMillis, prometheus.GaugeValue, float64(nodeStats.Events.DurationInMillis))
	ch <- prometheus.MustNewConstMetric(c.EventsQueuePushDurationInMillis, prometheus.GaugeValue, float64(nodeStats.Events.QueuePushDurationInMillis))

	ch <- prometheus.MustNewConstMetric(c.FlowInputCurrent, prometheus.GaugeValue, float64(nodeStats.Flow.InputThroughput.Current))
	ch <- prometheus.MustNewConstMetric(c.FlowInputLifetime, prometheus.GaugeValue, float64(nodeStats.Flow.InputThroughput.Lifetime))
	ch <- prometheus.MustNewConstMetric(c.FlowFilterCurrent, prometheus.GaugeValue, float64(nodeStats.Flow.FilterThroughput.Current))
	ch <- prometheus.MustNewConstMetric(c.FlowFilterLifetime, prometheus.GaugeValue, float64(nodeStats.Flow.FilterThroughput.Lifetime))
	ch <- prometheus.MustNewConstMetric(c.FlowOutputCurrent, prometheus.GaugeValue, float64(nodeStats.Flow.OutputThroughput.Current))
	ch <- prometheus.MustNewConstMetric(c.FlowOutputLifetime, prometheus.GaugeValue, float64(nodeStats.Flow.OutputThroughput.Lifetime))
	ch <- prometheus.MustNewConstMetric(c.FlowQueueBackpressureCurrent, prometheus.GaugeValue, float64(nodeStats.Flow.QueueBackpressure.Current))
	ch <- prometheus.MustNewConstMetric(c.FlowQueueBackpressureLifetime, prometheus.GaugeValue, float64(nodeStats.Flow.QueueBackpressure.Lifetime))
	ch <- prometheus.MustNewConstMetric(c.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, float64(nodeStats.Flow.WorkerConcurrency.Current))
	ch <- prometheus.MustNewConstMetric(c.FlowWorkerConcurrencyLifetime, prometheus.GaugeValue, float64(nodeStats.Flow.WorkerConcurrency.Lifetime))

	for pipelineId, pipelineStats := range nodeStats.Pipelines {
		c.pipelineSubcollector.Collect(&pipelineStats, pipelineId, ch)
	}

	return nil
}

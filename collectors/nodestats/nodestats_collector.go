package nodestats

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
	clients              []logstashclient.Client
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

func NewNodestatsCollector(clients []logstashclient.Client) *NodestatsCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodestatsCollector{
		clients: clients,

		pipelineSubcollector: NewPipelineSubcollector(),

		JvmThreadsCount:     descHelper.NewDesc("jvm_threads_count", "Number of live threads including both daemon and non-daemon threads."),
		JvmThreadsPeakCount: descHelper.NewDesc("jvm_threads_peak_count", "Peak live thread count since the Java virtual machine started or peak was reset."),

		JvmMemHeapUsedPercent:       descHelper.NewDesc("jvm_mem_heap_used_percent", "Percentage of the heap memory that is used."),
		JvmMemHeapCommittedBytes:    descHelper.NewDesc("jvm_mem_heap_committed_bytes", "Amount of heap memory in bytes that is committed for the Java virtual machine to use."),
		JvmMemHeapMaxBytes:          descHelper.NewDesc("jvm_mem_heap_max_bytes", "Maximum amount of heap memory in bytes that can be used for memory management."),
		JvmMemHeapUsedBytes:         descHelper.NewDesc("jvm_mem_heap_used_bytes", "Amount of used heap memory in bytes."),
		JvmMemNonHeapCommittedBytes: descHelper.NewDesc("jvm_mem_non_heap_committed_bytes", "Amount of non-heap memory in bytes that is committed for the Java virtual machine to use."),

		JvmMemPoolPeakUsedInBytes: descHelper.NewDesc(
			"jvm_mem_pool_peak_used_bytes", "Peak used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolUsedInBytes: descHelper.NewDesc(
			"jvm_mem_pool_used_bytes", "Currently used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolPeakMaxInBytes: descHelper.NewDesc(
			"jvm_mem_pool_peak_max_bytes", "Highest value of bytes that were used in a given JVM memory pool.", "pool"),
		JvmMemPoolMaxInBytes: descHelper.NewDesc(
			"jvm_mem_pool_max_bytes", "Maximum amount of bytes that can be used in a given JVM memory pool.", "pool"),
		JvmMemPoolCommittedInBytes: descHelper.NewDesc(
			"jvm_mem_pool_committed_bytes", "Amount of bytes that are committed for the Java virtual machine to use in a given JVM memory pool.", "pool"),

		JvmUptimeMillis: descHelper.NewDesc("jvm_uptime_millis", "Uptime of the JVM in milliseconds."),

		ProcessOpenFileDescriptors:    descHelper.NewDesc("process_open_file_descriptors", "Number of currently open file descriptors."),
		ProcessMaxFileDescriptors:     descHelper.NewDesc("process_max_file_descriptors", "Limit of open file descriptors."),
		ProcessCpuPercent:             descHelper.NewDesc("process_cpu_percent", "CPU usage of the process."),
		ProcessCpuTotalMillis:         descHelper.NewDesc("process_cpu_total_millis", "Total CPU time used by the process."),
		ProcessCpuLoadAverageOneM:     descHelper.NewDesc("process_cpu_load_average_1m", "Total 1m system load average."),
		ProcessCpuLoadAverageFiveM:    descHelper.NewDesc("process_cpu_load_average_5m", "Total 5m system load average."),
		ProcessCpuLoadAverageFifteenM: descHelper.NewDesc("process_cpu_load_average_15m", "Total 15m system load average."),

		ProcessMemTotalVirtual: descHelper.NewDesc("process_mem_total_virtual", "Total virtual memory used by the process."),

		ReloadSuccesses: descHelper.NewDesc("reload_successes", "Number of successful reloads."),
		ReloadFailures:  descHelper.NewDesc("reload_failures", "Number of failed reloads."),

		QueueEventsCount: descHelper.NewDesc("queue_events_count", "Number of events in the queue."),

		EventsIn:                        descHelper.NewDesc("events_in", "Number of events received."),
		EventsFiltered:                  descHelper.NewDesc("events_filtered", "Number of events filtered out."),
		EventsOut:                       descHelper.NewDesc("events_out", "Number of events out."),
		EventsDurationInMillis:          descHelper.NewDesc("events_duration_millis", "Duration of events processing in milliseconds."),
		EventsQueuePushDurationInMillis: descHelper.NewDesc("events_queue_push_duration_millis", "Duration of events push to queue in milliseconds."),

		FlowInputCurrent:              descHelper.NewDesc("flow_input_current", "Current number of events in the input queue."),
		FlowInputLifetime:             descHelper.NewDesc("flow_input_lifetime", "Lifetime number of events in the input queue."),
		FlowFilterCurrent:             descHelper.NewDesc("flow_filter_current", "Current number of events in the filter queue."),
		FlowFilterLifetime:            descHelper.NewDesc("flow_filter_lifetime", "Lifetime number of events in the filter queue."),
		FlowOutputCurrent:             descHelper.NewDesc("flow_output_current", "Current number of events in the output queue."),
		FlowOutputLifetime:            descHelper.NewDesc("flow_output_lifetime", "Lifetime number of events in the output queue."),
		FlowQueueBackpressureCurrent:  descHelper.NewDesc("flow_queue_backpressure_current", "Current number of events in the backpressure queue."),
		FlowQueueBackpressureLifetime: descHelper.NewDesc("flow_queue_backpressure_lifetime", "Lifetime number of events in the backpressure queue."),
		FlowWorkerConcurrencyCurrent:  descHelper.NewDesc("flow_worker_concurrency_current", "Current number of workers."),
		FlowWorkerConcurrencyLifetime: descHelper.NewDesc("flow_worker_concurrency_lifetime", "Lifetime number of workers."),
	}
}

func (c *NodestatsCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	wg := sync.WaitGroup{}
	wg.Add(len(c.clients))

	errorChannel := make(chan error, len(c.clients))

	for _, client := range c.clients {
		go func(client logstashclient.Client) {
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

func (c *NodestatsCollector) collectSingleInstance(client logstashclient.Client, ctx context.Context, ch chan<- prometheus.Metric) error {
	nodeStats, err := client.GetNodeStats(ctx)
	if err != nil {
		return err
	}

	endpoint := client.GetEndpoint()

	newFloatMetric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value float64, labels ...string) {
		labels = append(labels, endpoint)
		metric := prometheus.MustNewConstMetric(desc, metricType, value, labels...)

		ch <- metric
	}

	newInt64Metric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value int64, labels ...string) {
		newFloatMetric(desc, metricType, float64(value), labels...)
	}

	newIntMetric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value int, labels ...string) {
		newFloatMetric(desc, metricType, float64(value), labels...)
	}

	newIntMetric(c.JvmThreadsCount, prometheus.GaugeValue, nodeStats.Jvm.Threads.Count)
	newIntMetric(c.JvmThreadsPeakCount, prometheus.GaugeValue, nodeStats.Jvm.Threads.PeakCount)

	newIntMetric(c.JvmMemHeapUsedPercent, prometheus.GaugeValue, nodeStats.Jvm.Mem.HeapUsedPercent)
	newIntMetric(c.JvmMemHeapCommittedBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.HeapCommittedInBytes)
	newIntMetric(c.JvmMemHeapMaxBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.HeapMaxInBytes)
	newIntMetric(c.JvmMemHeapUsedBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.HeapUsedInBytes)
	newIntMetric(c.JvmMemNonHeapCommittedBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.NonHeapCommittedInBytes)

	// POOLS
	// young
	newIntMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Young.PeakUsedInBytes, "young")
	newIntMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Young.UsedInBytes, "young")
	newIntMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Young.PeakMaxInBytes, "young")
	newIntMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Young.MaxInBytes, "young")
	newIntMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Young.CommittedInBytes, "young")

	// old
	newIntMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Old.PeakUsedInBytes, "old")
	newIntMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Old.UsedInBytes, "old")
	newIntMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Old.PeakMaxInBytes, "old")
	newIntMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Old.MaxInBytes, "old")
	newIntMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Old.CommittedInBytes, "old")

	// survivor
	newIntMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Survivor.PeakUsedInBytes, "survivor")
	newIntMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Survivor.UsedInBytes, "survivor")
	newIntMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Survivor.PeakMaxInBytes, "survivor")
	newIntMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Survivor.MaxInBytes, "survivor")
	newIntMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, nodeStats.Jvm.Mem.Pools.Survivor.CommittedInBytes, "survivor")

	newIntMetric(c.JvmUptimeMillis, prometheus.GaugeValue, nodeStats.Jvm.UptimeInMillis)

	newInt64Metric(c.ProcessOpenFileDescriptors, prometheus.GaugeValue, nodeStats.Process.OpenFileDescriptors)
	newInt64Metric(c.ProcessMaxFileDescriptors, prometheus.GaugeValue, nodeStats.Process.MaxFileDescriptors)
	newInt64Metric(c.ProcessCpuPercent, prometheus.GaugeValue, nodeStats.Process.CPU.Percent)
	newInt64Metric(c.ProcessCpuTotalMillis, prometheus.GaugeValue, nodeStats.Process.CPU.TotalInMillis)
	newFloatMetric(c.ProcessCpuLoadAverageOneM, prometheus.GaugeValue, nodeStats.Process.CPU.LoadAverage.OneM)
	newFloatMetric(c.ProcessCpuLoadAverageFiveM, prometheus.GaugeValue, nodeStats.Process.CPU.LoadAverage.FiveM)
	newFloatMetric(c.ProcessCpuLoadAverageFifteenM, prometheus.GaugeValue, nodeStats.Process.CPU.LoadAverage.FifteenM)
	newInt64Metric(c.ProcessMemTotalVirtual, prometheus.GaugeValue, nodeStats.Process.Mem.TotalVirtualInBytes)

	newIntMetric(c.ReloadSuccesses, prometheus.GaugeValue, nodeStats.Reloads.Successes)
	newIntMetric(c.ReloadFailures, prometheus.GaugeValue, nodeStats.Reloads.Failures)

	newIntMetric(c.QueueEventsCount, prometheus.GaugeValue, nodeStats.Queue.EventsCount)

	newInt64Metric(c.EventsIn, prometheus.GaugeValue, nodeStats.Events.In)
	newInt64Metric(c.EventsFiltered, prometheus.GaugeValue, nodeStats.Events.Filtered)
	newInt64Metric(c.EventsOut, prometheus.GaugeValue, nodeStats.Events.Out)
	newInt64Metric(c.EventsDurationInMillis, prometheus.GaugeValue, nodeStats.Events.DurationInMillis)
	newInt64Metric(c.EventsQueuePushDurationInMillis, prometheus.GaugeValue, nodeStats.Events.QueuePushDurationInMillis)

	newFloatMetric(c.FlowInputCurrent, prometheus.GaugeValue, nodeStats.Flow.InputThroughput.Current)
	newFloatMetric(c.FlowInputLifetime, prometheus.GaugeValue, nodeStats.Flow.InputThroughput.Lifetime)
	newFloatMetric(c.FlowFilterCurrent, prometheus.GaugeValue, nodeStats.Flow.FilterThroughput.Current)
	newFloatMetric(c.FlowFilterLifetime, prometheus.GaugeValue, nodeStats.Flow.FilterThroughput.Lifetime)
	newFloatMetric(c.FlowOutputCurrent, prometheus.GaugeValue, nodeStats.Flow.OutputThroughput.Current)
	newFloatMetric(c.FlowOutputLifetime, prometheus.GaugeValue, nodeStats.Flow.OutputThroughput.Lifetime)
	newFloatMetric(c.FlowQueueBackpressureCurrent, prometheus.GaugeValue, nodeStats.Flow.QueueBackpressure.Current)
	newFloatMetric(c.FlowQueueBackpressureLifetime, prometheus.GaugeValue, nodeStats.Flow.QueueBackpressure.Lifetime)
	newFloatMetric(c.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, nodeStats.Flow.WorkerConcurrency.Current)
	newFloatMetric(c.FlowWorkerConcurrencyLifetime, prometheus.GaugeValue, nodeStats.Flow.WorkerConcurrency.Lifetime)

	for pipelineId, pipelineStats := range nodeStats.Pipelines {
		c.pipelineSubcollector.Collect(&pipelineStats, pipelineId, ch, endpoint)
	}

	return nil
}

package nodestats

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/internal/prometheus_helper"
	"github.com/kuskoman/logstash-exporter/pkg/config"
)

const subsystem = "stats"

var (
	namespace = config.PrometheusNamespace
)

// NodestatsCollector is a custom collector for the /_node/stats endpoint
type NodestatsCollector struct {
	clients              []logstash_client.Client
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

func NewNodestatsCollector(clients []logstash_client.Client) *NodestatsCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodestatsCollector{
		clients: clients,

		pipelineSubcollector: NewPipelineSubcollector(),

		JvmThreadsCount: descHelper.NewDesc("jvm_threads_count",
			"Number of live threads including both daemon and non-daemon threads."),
		JvmThreadsPeakCount: descHelper.NewDesc("jvm_threads_peak_count",
			"Peak live thread count since the Java virtual machine started or peak was reset."),

		JvmMemHeapUsedPercent: descHelper.NewDesc("jvm_mem_heap_used_percent",
			"Percentage of the heap memory that is used."),
		JvmMemHeapCommittedBytes: descHelper.NewDesc("jvm_mem_heap_committed_bytes",
			"Amount of heap memory in bytes that is committed for the Java virtual machine to use."),
		JvmMemHeapMaxBytes: descHelper.NewDesc("jvm_mem_heap_max_bytes",
			"Maximum amount of heap memory in bytes that can be used for memory management."),
		JvmMemHeapUsedBytes: descHelper.NewDesc("jvm_mem_heap_used_bytes",
			"Amount of used heap memory in bytes."),
		JvmMemNonHeapCommittedBytes: descHelper.NewDesc("jvm_mem_non_heap_committed_bytes",
			"Amount of non-heap memory in bytes that is committed for the Java virtual machine to use."),

		JvmMemPoolPeakUsedInBytes: descHelper.NewDesc("jvm_mem_pool_peak_used_bytes",
			"Peak used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolUsedInBytes: descHelper.NewDesc("jvm_mem_pool_used_bytes",
			"Currently used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolPeakMaxInBytes: descHelper.NewDesc("jvm_mem_pool_peak_max_bytes",
			"Highest value of bytes that were used in a given JVM memory pool.", "pool"),
		JvmMemPoolMaxInBytes: descHelper.NewDesc("jvm_mem_pool_max_bytes",
			"Maximum amount of bytes that can be used in a given JVM memory pool.", "pool"),
		JvmMemPoolCommittedInBytes: descHelper.NewDesc("jvm_mem_pool_committed_bytes",
			"Amount of bytes that are committed for the Java virtual machine to use in a given JVM memory pool.", "pool"),

		JvmUptimeMillis: descHelper.NewDesc("jvm_uptime_millis",
			"Uptime of the JVM in milliseconds."),

		ProcessOpenFileDescriptors: descHelper.NewDesc("process_open_file_descriptors",
			"Number of currently open file descriptors."),
		ProcessMaxFileDescriptors: descHelper.NewDesc("process_max_file_descriptors",
			"Limit of open file descriptors."),
		ProcessCpuPercent: descHelper.NewDesc("process_cpu_percent",
			"CPU usage of the process."),
		ProcessCpuTotalMillis:         descHelper.NewDesc("process_cpu_total_millis", "Total CPU time used by the process."),
		ProcessCpuLoadAverageOneM:     descHelper.NewDesc("process_cpu_load_average_1m", "Total 1m system load average."),
		ProcessCpuLoadAverageFiveM:    descHelper.NewDesc("process_cpu_load_average_5m", "Total 5m system load average."),
		ProcessCpuLoadAverageFifteenM: descHelper.NewDesc("process_cpu_load_average_15m", "Total 15m system load average."),

		ProcessMemTotalVirtual: descHelper.NewDesc("process_mem_total_virtual", "Total virtual memory used by the process."),

		ReloadSuccesses: descHelper.NewDesc("reload_successes", "Number of successful reloads."),
		ReloadFailures:  descHelper.NewDesc("reload_failures", "Number of failed reloads."), QueueEventsCount: descHelper.NewDesc("queue_events_count", "Number of events in the queue."),

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

func (collector *NodestatsCollector) collectSingleInstance(client logstash_client.Client, ctx context.Context, ch chan<- prometheus.Metric) error {
	nodeStats, err := client.GetNodeStats(ctx)
	if err != nil {
		return err
	}

	endpoint := client.GetEndpoint()
	metricsHelper := prometheus_helper.SimpleMetricsHelper{Channel: ch, Labels: []string{endpoint}}

	// ************ THREADS ************
	threadsStats := nodeStats.Jvm.Threads
	metricsHelper.NewIntMetric(collector.JvmThreadsCount, prometheus.GaugeValue, threadsStats.Count)
	metricsHelper.NewIntMetric(collector.JvmThreadsPeakCount, prometheus.GaugeValue, threadsStats.PeakCount)
	// *********************************

	// ************ MEMORY ************
	memStats := nodeStats.Jvm.Mem
	metricsHelper.NewIntMetric(collector.JvmMemHeapUsedPercent, prometheus.GaugeValue, memStats.HeapUsedPercent)
	metricsHelper.NewIntMetric(collector.JvmMemHeapCommittedBytes, prometheus.GaugeValue, memStats.HeapCommittedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemHeapMaxBytes, prometheus.GaugeValue, memStats.HeapMaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemHeapUsedBytes, prometheus.GaugeValue, memStats.HeapUsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemNonHeapCommittedBytes, prometheus.GaugeValue, memStats.NonHeapCommittedInBytes)

	//	  ********* POOLS *********
	//          *** YOUNG ***
	metricsHelper.Labels = []string{"young", endpoint}
	metricsHelper.NewIntMetric(collector.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, memStats.Pools.Young.PeakUsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolUsedInBytes, prometheus.GaugeValue, memStats.Pools.Young.UsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, memStats.Pools.Young.PeakMaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolMaxInBytes, prometheus.GaugeValue, memStats.Pools.Young.MaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, memStats.Pools.Young.CommittedInBytes)
	//          *************

	//           *** OLD ***
	metricsHelper.Labels = []string{"old", endpoint}
	metricsHelper.NewIntMetric(collector.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, memStats.Pools.Old.PeakUsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolUsedInBytes, prometheus.GaugeValue, memStats.Pools.Old.UsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, memStats.Pools.Old.PeakMaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolMaxInBytes, prometheus.GaugeValue, memStats.Pools.Old.MaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, memStats.Pools.Old.CommittedInBytes)
	//           ***********

	//         *** SURVIVOR ***
	metricsHelper.Labels = []string{"survivor", endpoint}
	metricsHelper.NewIntMetric(collector.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.PeakUsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolUsedInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.UsedInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.PeakMaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolMaxInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.MaxInBytes)
	metricsHelper.NewIntMetric(collector.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.CommittedInBytes)
	//         ****************
	//	  *************************
	// ********************************

	metricsHelper.Labels = []string{endpoint}

	// ************ UPTIME ************
	metricsHelper.NewIntMetric(collector.JvmUptimeMillis, prometheus.GaugeValue, nodeStats.Jvm.UptimeInMillis)
	// ********************************

	// ************ PROCESS ************
	procStats := nodeStats.Process
	metricsHelper.NewUInt64Metric(collector.ProcessOpenFileDescriptors, prometheus.GaugeValue, procStats.OpenFileDescriptors)
	metricsHelper.NewUInt64Metric(collector.ProcessMaxFileDescriptors, prometheus.GaugeValue, procStats.MaxFileDescriptors)
	metricsHelper.NewIntMetric(collector.ProcessCpuPercent, prometheus.GaugeValue, procStats.CPU.Percent)
	metricsHelper.NewUInt64Metric(collector.ProcessCpuTotalMillis, prometheus.CounterValue, procStats.CPU.TotalInMillis)
	metricsHelper.NewFloatMetric(collector.ProcessCpuLoadAverageOneM, prometheus.GaugeValue, procStats.CPU.LoadAverage.OneM)
	metricsHelper.NewFloatMetric(collector.ProcessCpuLoadAverageFiveM, prometheus.GaugeValue, procStats.CPU.LoadAverage.FiveM)
	metricsHelper.NewFloatMetric(collector.ProcessCpuLoadAverageFifteenM, prometheus.GaugeValue, procStats.CPU.LoadAverage.FifteenM)
	metricsHelper.NewUInt64Metric(collector.ProcessMemTotalVirtual, prometheus.GaugeValue, procStats.Mem.TotalVirtualInBytes)
	// *********************************

	// ************ RELOADS ************
	metricsHelper.NewIntMetric(collector.ReloadSuccesses, prometheus.CounterValue, nodeStats.Reloads.Successes)
	metricsHelper.NewIntMetric(collector.ReloadFailures, prometheus.CounterValue, nodeStats.Reloads.Failures)
	// *********************************

	// ************ EVENTS COUNT ************
	metricsHelper.NewIntMetric(collector.QueueEventsCount, prometheus.GaugeValue, nodeStats.Queue.EventsCount)
	// **************************************

	// ************ EVENTS ************
	eventsStats := nodeStats.Events
	metricsHelper.NewUInt64Metric(collector.EventsIn, prometheus.GaugeValue, eventsStats.In)
	metricsHelper.NewUInt64Metric(collector.EventsFiltered, prometheus.GaugeValue, eventsStats.Filtered)
	metricsHelper.NewUInt64Metric(collector.EventsOut, prometheus.GaugeValue, eventsStats.Out)
	metricsHelper.NewUInt64Metric(collector.EventsDurationInMillis, prometheus.GaugeValue, eventsStats.DurationInMillis)
	metricsHelper.NewUInt64Metric(collector.EventsQueuePushDurationInMillis, prometheus.GaugeValue, eventsStats.QueuePushDurationInMillis)
	// ********************************

	// ************ FLOW ************
	flowStats := nodeStats.Flow
	metricsHelper.NewFloatMetric(collector.FlowInputCurrent, prometheus.GaugeValue, flowStats.InputThroughput.Current)
	metricsHelper.NewFloatMetric(collector.FlowInputLifetime, prometheus.CounterValue, nodeStats.Flow.InputThroughput.Lifetime)
	metricsHelper.NewFloatMetric(collector.FlowFilterCurrent, prometheus.GaugeValue, nodeStats.Flow.FilterThroughput.Current)
	metricsHelper.NewFloatMetric(collector.FlowFilterLifetime, prometheus.CounterValue, nodeStats.Flow.FilterThroughput.Lifetime)
	metricsHelper.NewFloatMetric(collector.FlowOutputCurrent, prometheus.GaugeValue, nodeStats.Flow.OutputThroughput.Current)
	metricsHelper.NewFloatMetric(collector.FlowOutputLifetime, prometheus.CounterValue, nodeStats.Flow.OutputThroughput.Lifetime)
	metricsHelper.NewFloatMetric(collector.FlowQueueBackpressureCurrent, prometheus.GaugeValue, nodeStats.Flow.QueueBackpressure.Current)
	metricsHelper.NewFloatMetric(collector.FlowQueueBackpressureLifetime, prometheus.CounterValue, nodeStats.Flow.QueueBackpressure.Lifetime)
	metricsHelper.NewFloatMetric(collector.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, nodeStats.Flow.WorkerConcurrency.Current)
	metricsHelper.NewFloatMetric(collector.FlowWorkerConcurrencyLifetime, prometheus.CounterValue, nodeStats.Flow.WorkerConcurrency.Lifetime)
	// ******************************

	for pipelineId, pipelineStats := range nodeStats.Pipelines {
		collector.pipelineSubcollector.Collect(&pipelineStats, pipelineId, ch, endpoint)
	}

	return nil
}

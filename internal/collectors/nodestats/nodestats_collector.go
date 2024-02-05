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
	clients                         []logstash_client.Client
	pipelineSubcollector            *PipelineSubcollector

	JvmThreadsCount                 *prometheus.Desc
	JvmThreadsPeakCount             *prometheus.Desc

	JvmMemHeapUsedPercent           *prometheus.Desc
	JvmMemHeapCommittedBytes        *prometheus.Desc
	JvmMemHeapMaxBytes              *prometheus.Desc
	JvmMemHeapUsedBytes             *prometheus.Desc
	JvmMemNonHeapCommittedBytes     *prometheus.Desc

	JvmMemPoolPeakUsedInBytes       *prometheus.Desc
	JvmMemPoolUsedInBytes           *prometheus.Desc
	JvmMemPoolPeakMaxInBytes        *prometheus.Desc
	JvmMemPoolMaxInBytes            *prometheus.Desc
	JvmMemPoolCommittedInBytes      *prometheus.Desc

	JvmUptimeMillis                 *prometheus.Desc

	ProcessOpenFileDescriptors      *prometheus.Desc
	ProcessMaxFileDescriptors       *prometheus.Desc
	ProcessCpuPercent               *prometheus.Desc
	ProcessCpuTotalMillis           *prometheus.Desc
	ProcessCpuLoadAverageOneM       *prometheus.Desc
	ProcessCpuLoadAverageFiveM      *prometheus.Desc
	ProcessCpuLoadAverageFifteenM   *prometheus.Desc
	ProcessMemTotalVirtual          *prometheus.Desc

	ReloadSuccesses                 *prometheus.Desc
	ReloadFailures                  *prometheus.Desc

	QueueEventsCount                *prometheus.Desc

	EventsIn                        *prometheus.Desc
	EventsFiltered                  *prometheus.Desc
	EventsOut                       *prometheus.Desc
	EventsDurationInMillis          *prometheus.Desc
	EventsQueuePushDurationInMillis *prometheus.Desc

	FlowInputCurrent                *prometheus.Desc
	FlowInputLifetime               *prometheus.Desc
	FlowFilterCurrent               *prometheus.Desc
	FlowFilterLifetime              *prometheus.Desc
	FlowOutputCurrent               *prometheus.Desc
	FlowOutputLifetime              *prometheus.Desc
	FlowQueueBackpressureCurrent    *prometheus.Desc
	FlowQueueBackpressureLifetime   *prometheus.Desc
	FlowWorkerConcurrencyCurrent    *prometheus.Desc
	FlowWorkerConcurrencyLifetime   *prometheus.Desc
}

func NewNodestatsCollector(clients []logstash_client.Client) *NodestatsCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}

	return &NodestatsCollector{
		clients:                         clients,

		pipelineSubcollector:            NewPipelineSubcollector(),

		JvmThreadsCount:                 descHelper.NewDesc("jvm_threads_count",
            "Number of live threads including both daemon and non-daemon threads."),
		JvmThreadsPeakCount:             descHelper.NewDesc("jvm_threads_peak_count",
            "Peak live thread count since the Java virtual machine started or peak was reset."),

		JvmMemHeapUsedPercent:           descHelper.NewDesc("jvm_mem_heap_used_percent",
            "Percentage of the heap memory that is used."),
		JvmMemHeapCommittedBytes:        descHelper.NewDesc("jvm_mem_heap_committed_bytes",
            "Amount of heap memory in bytes that is committed for the Java virtual machine to use."),
		JvmMemHeapMaxBytes:              descHelper.NewDesc("jvm_mem_heap_max_bytes",
            "Maximum amount of heap memory in bytes that can be used for memory management."),
		JvmMemHeapUsedBytes:             descHelper.NewDesc("jvm_mem_heap_used_bytes",
            "Amount of used heap memory in bytes."),
		JvmMemNonHeapCommittedBytes:     descHelper.NewDesc("jvm_mem_non_heap_committed_bytes",
            "Amount of non-heap memory in bytes that is committed for the Java virtual machine to use."),

		JvmMemPoolPeakUsedInBytes:       descHelper.NewDesc("jvm_mem_pool_peak_used_bytes",
            "Peak used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolUsedInBytes:           descHelper.NewDesc("jvm_mem_pool_used_bytes",
            "Currently used bytes of a given JVM memory pool.", "pool"),
		JvmMemPoolPeakMaxInBytes:        descHelper.NewDesc("jvm_mem_pool_peak_max_bytes",
            "Highest value of bytes that were used in a given JVM memory pool.", "pool"),
		JvmMemPoolMaxInBytes:            descHelper.NewDesc("jvm_mem_pool_max_bytes",
            "Maximum amount of bytes that can be used in a given JVM memory pool.", "pool"),
		JvmMemPoolCommittedInBytes:      descHelper.NewDesc("jvm_mem_pool_committed_bytes",
            "Amount of bytes that are committed for the Java virtual machine to use in a given JVM memory pool.", "pool"),

		JvmUptimeMillis:                 descHelper.NewDesc("jvm_uptime_millis",
            "Uptime of the JVM in milliseconds."),

		ProcessOpenFileDescriptors:      descHelper.NewDesc("process_open_file_descriptors",
            "Number of currently open file descriptors."),
		ProcessMaxFileDescriptors:       descHelper.NewDesc("process_max_file_descriptors",
            "Limit of open file descriptors."),
		ProcessCpuPercent:               descHelper.NewDesc("process_cpu_percent",
            "CPU usage of the process."),
		ProcessCpuTotalMillis:           descHelper.NewDesc("process_cpu_total_millis", "Total CPU time used by the process."),
		ProcessCpuLoadAverageOneM:       descHelper.NewDesc("process_cpu_load_average_1m", "Total 1m system load average."),
		ProcessCpuLoadAverageFiveM:      descHelper.NewDesc("process_cpu_load_average_5m", "Total 5m system load average."),
		ProcessCpuLoadAverageFifteenM:   descHelper.NewDesc("process_cpu_load_average_15m", "Total 15m system load average."),

		ProcessMemTotalVirtual:          descHelper.NewDesc("process_mem_total_virtual", "Total virtual memory used by the process."),

		ReloadSuccesses:                 descHelper.NewDesc("reload_successes", "Number of successful reloads."),
		ReloadFailures:                  descHelper.NewDesc("reload_failures", "Number of failed reloads."), QueueEventsCount:                descHelper.NewDesc("queue_events_count", "Number of events in the queue."),

		EventsIn:                        descHelper.NewDesc("events_in", "Number of events received."),
		EventsFiltered:                  descHelper.NewDesc("events_filtered", "Number of events filtered out."),
		EventsOut:                       descHelper.NewDesc("events_out", "Number of events out."),
		EventsDurationInMillis:          descHelper.NewDesc("events_duration_millis", "Duration of events processing in milliseconds."),
		EventsQueuePushDurationInMillis: descHelper.NewDesc("events_queue_push_duration_millis", "Duration of events push to queue in milliseconds."),

		FlowInputCurrent:                descHelper.NewDesc("flow_input_current", "Current number of events in the input queue."),
		FlowInputLifetime:               descHelper.NewDesc("flow_input_lifetime", "Lifetime number of events in the input queue."),
		FlowFilterCurrent:               descHelper.NewDesc("flow_filter_current", "Current number of events in the filter queue."),
		FlowFilterLifetime:              descHelper.NewDesc("flow_filter_lifetime", "Lifetime number of events in the filter queue."),
		FlowOutputCurrent:               descHelper.NewDesc("flow_output_current", "Current number of events in the output queue."),
		FlowOutputLifetime:              descHelper.NewDesc("flow_output_lifetime", "Lifetime number of events in the output queue."),
		FlowQueueBackpressureCurrent:    descHelper.NewDesc("flow_queue_backpressure_current", "Current number of events in the backpressure queue."),
		FlowQueueBackpressureLifetime:   descHelper.NewDesc("flow_queue_backpressure_lifetime", "Lifetime number of events in the backpressure queue."),
		FlowWorkerConcurrencyCurrent:    descHelper.NewDesc("flow_worker_concurrency_current", "Current number of workers."),
		FlowWorkerConcurrencyLifetime:   descHelper.NewDesc("flow_worker_concurrency_lifetime", "Lifetime number of workers."),
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

func (c *NodestatsCollector) collectSingleInstance(client logstash_client.Client, ctx context.Context, ch chan<- prometheus.Metric) error {
	nodeStats, err := client.GetNodeStats(ctx)
	if err != nil {
		return err
	}


	endpoint := client.GetEndpoint()
	mh := prometheus_helper.SimpleMetricsHelper{Channel: ch, Labels: []string{endpoint}}

	// ************ THREADS ************
	threadsStats := nodeStats.Jvm.Threads
	mh.NewIntMetric(c.JvmThreadsCount, prometheus.GaugeValue, threadsStats.Count)
	mh.NewIntMetric(c.JvmThreadsPeakCount, prometheus.GaugeValue, threadsStats.PeakCount)
	// *********************************

	// ************ MEMORY ************
	memStats := nodeStats.Jvm.Mem
	mh.NewIntMetric(c.JvmMemHeapUsedPercent, prometheus.GaugeValue, memStats.HeapUsedPercent)
	mh.NewIntMetric(c.JvmMemHeapCommittedBytes, prometheus.GaugeValue, memStats.HeapCommittedInBytes)
	mh.NewIntMetric(c.JvmMemHeapMaxBytes, prometheus.GaugeValue, memStats.HeapMaxInBytes)
	mh.NewIntMetric(c.JvmMemHeapUsedBytes, prometheus.GaugeValue, memStats.HeapUsedInBytes)
	mh.NewIntMetric(c.JvmMemNonHeapCommittedBytes, prometheus.GaugeValue, memStats.NonHeapCommittedInBytes)

	//	  ********* POOLS *********
	//          *** YOUNG ***
	mh.Labels = []string{"young", endpoint}
	mh.NewIntMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, memStats.Pools.Young.PeakUsedInBytes)
	mh.NewIntMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, memStats.Pools.Young.UsedInBytes)
	mh.NewIntMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, memStats.Pools.Young.PeakMaxInBytes)
	mh.NewIntMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, memStats.Pools.Young.MaxInBytes)
	mh.NewIntMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, memStats.Pools.Young.CommittedInBytes)
	//          *************

	//           *** OLD ***
	mh.Labels = []string{"old", endpoint}
	mh.NewIntMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, memStats.Pools.Old.PeakUsedInBytes)
	mh.NewIntMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, memStats.Pools.Old.UsedInBytes)
	mh.NewIntMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, memStats.Pools.Old.PeakMaxInBytes)
	mh.NewIntMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, memStats.Pools.Old.MaxInBytes)
	mh.NewIntMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, memStats.Pools.Old.CommittedInBytes)
	//           ***********

	//         *** SURVIVOR ***
	mh.Labels = []string{"survivor", endpoint}
	mh.NewIntMetric(c.JvmMemPoolPeakUsedInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.PeakUsedInBytes)
	mh.NewIntMetric(c.JvmMemPoolUsedInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.UsedInBytes)
	mh.NewIntMetric(c.JvmMemPoolPeakMaxInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.PeakMaxInBytes)
	mh.NewIntMetric(c.JvmMemPoolMaxInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.MaxInBytes)
	mh.NewIntMetric(c.JvmMemPoolCommittedInBytes, prometheus.GaugeValue, memStats.Pools.Survivor.CommittedInBytes)
	//         ****************
	//	  *************************
	// ********************************

	mh.Labels = []string{endpoint}
	
	// ************ UPTIME ************
	mh.NewIntMetric(c.JvmUptimeMillis, prometheus.GaugeValue, nodeStats.Jvm.UptimeInMillis)
	// ********************************

	// ************ PROCESS ************
	procStats := nodeStats.Process
	mh.NewInt64Metric(c.ProcessOpenFileDescriptors, prometheus.GaugeValue, procStats.OpenFileDescriptors)
	mh.NewInt64Metric(c.ProcessMaxFileDescriptors, prometheus.GaugeValue, procStats.MaxFileDescriptors)
	mh.NewInt64Metric(c.ProcessCpuPercent, prometheus.GaugeValue, procStats.CPU.Percent)
	mh.NewInt64Metric(c.ProcessCpuTotalMillis, prometheus.GaugeValue, procStats.CPU.TotalInMillis)
	mh.NewFloatMetric(c.ProcessCpuLoadAverageOneM, prometheus.GaugeValue, procStats.CPU.LoadAverage.OneM)
	mh.NewFloatMetric(c.ProcessCpuLoadAverageFiveM, prometheus.GaugeValue, procStats.CPU.LoadAverage.FiveM)
	mh.NewFloatMetric(c.ProcessCpuLoadAverageFifteenM, prometheus.GaugeValue, procStats.CPU.LoadAverage.FifteenM)
	mh.NewInt64Metric(c.ProcessMemTotalVirtual, prometheus.GaugeValue, procStats.Mem.TotalVirtualInBytes)
	// *********************************

	// ************ RELOADS ************
	mh.NewIntMetric(c.ReloadSuccesses, prometheus.GaugeValue, nodeStats.Reloads.Successes)
	mh.NewIntMetric(c.ReloadFailures, prometheus.GaugeValue, nodeStats.Reloads.Failures)
	// *********************************

	// ************ EVENTS COUNT ************
	mh.NewIntMetric(c.QueueEventsCount, prometheus.GaugeValue, nodeStats.Queue.EventsCount)
	// **************************************

	// ************ EVENTS ************
	eventsStats := nodeStats.Events
	mh.NewInt64Metric(c.EventsIn, prometheus.GaugeValue, eventsStats.In)
	mh.NewInt64Metric(c.EventsFiltered, prometheus.GaugeValue, eventsStats.Filtered)
	mh.NewInt64Metric(c.EventsOut, prometheus.GaugeValue, eventsStats.Out)
	mh.NewInt64Metric(c.EventsDurationInMillis, prometheus.GaugeValue, eventsStats.DurationInMillis)
	mh.NewInt64Metric(c.EventsQueuePushDurationInMillis, prometheus.GaugeValue, eventsStats.QueuePushDurationInMillis)
	// ********************************

	// ************ FLOW ************
	flowStats := nodeStats.Flow
	mh.NewFloatMetric(c.FlowInputCurrent, prometheus.GaugeValue, flowStats.InputThroughput.Current)
	mh.NewFloatMetric(c.FlowInputLifetime, prometheus.GaugeValue, nodeStats.Flow.InputThroughput.Lifetime)
	mh.NewFloatMetric(c.FlowFilterCurrent, prometheus.GaugeValue, nodeStats.Flow.FilterThroughput.Current)
	mh.NewFloatMetric(c.FlowFilterLifetime, prometheus.GaugeValue, nodeStats.Flow.FilterThroughput.Lifetime)
	mh.NewFloatMetric(c.FlowOutputCurrent, prometheus.GaugeValue, nodeStats.Flow.OutputThroughput.Current)
	mh.NewFloatMetric(c.FlowOutputLifetime, prometheus.GaugeValue, nodeStats.Flow.OutputThroughput.Lifetime)
	mh.NewFloatMetric(c.FlowQueueBackpressureCurrent, prometheus.GaugeValue, nodeStats.Flow.QueueBackpressure.Current)
	mh.NewFloatMetric(c.FlowQueueBackpressureLifetime, prometheus.GaugeValue, nodeStats.Flow.QueueBackpressure.Lifetime)
	mh.NewFloatMetric(c.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, nodeStats.Flow.WorkerConcurrency.Current)
	mh.NewFloatMetric(c.FlowWorkerConcurrencyLifetime, prometheus.GaugeValue, nodeStats.Flow.WorkerConcurrency.Lifetime)
	// ******************************

	for pipelineId, pipelineStats := range nodeStats.Pipelines {
		c.pipelineSubcollector.Collect(&pipelineStats, pipelineId, ch, endpoint)
	}

	return nil
}

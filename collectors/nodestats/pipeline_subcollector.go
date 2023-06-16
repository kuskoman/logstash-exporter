package nodestats

import (
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
)

const (
	CollectorUnhealthy = 0
	CollectorHealthy   = 1
)

// PipelineSubcollector is a subcollector that collects metrics about the
// pipelines of a logstash node.
// The collector is created once for each pipeline of the node.
type PipelineSubcollector struct {
	Up                      *prometheus.Desc
	EventsOut               *prometheus.Desc
	EventsFiltered          *prometheus.Desc
	EventsIn                *prometheus.Desc
	EventsDuration          *prometheus.Desc
	EventsQueuePushDuration *prometheus.Desc

	ReloadsLastSuccessTimestamp *prometheus.Desc
	ReloadsLastFailureTimestamp *prometheus.Desc
	ReloadsSuccesses            *prometheus.Desc
	ReloadsFailures             *prometheus.Desc

	QueueEventsCount         *prometheus.Desc
	QueueEventsQueueSize     *prometheus.Desc
	QueueMaxQueueSizeInBytes *prometheus.Desc

	PipelinePluginEventsIn                *prometheus.Desc
	PipelinePluginEventsOut               *prometheus.Desc
	PipelinePluginEventsDuration          *prometheus.Desc
	PipelinePluginEventsQueuePushDuration *prometheus.Desc

	PipelinePluginDocumentsSuccesses            *prometheus.Desc
	PipelinePluginDocumentsNonRetryableFailures *prometheus.Desc
	PipelinePluginBulkRequestErrors             *prometheus.Desc
	PipelinePluginBulkRequestResponses          *prometheus.Desc

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

func NewPipelineSubcollector() *PipelineSubcollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: fmt.Sprintf("%s_pipeline", subsystem)}
	return &PipelineSubcollector{
		Up:                      descHelper.NewDescWithHelpAndLabels("up", "Whether the pipeline is up or not.", "pipeline"),
		EventsOut:               descHelper.NewDescWithHelpAndLabels("events_out", "Number of events that have been processed by this pipeline.", "pipeline"),
		EventsFiltered:          descHelper.NewDescWithHelpAndLabels("events_filtered", "Number of events that have been filtered out by this pipeline.", "pipeline"),
		EventsIn:                descHelper.NewDescWithHelpAndLabels("events_in", "Number of events that have been inputted into this pipeline.", "pipeline"),
		EventsDuration:          descHelper.NewDescWithHelpAndLabels("events_duration", "Time needed to process event.", "pipeline"),
		EventsQueuePushDuration: descHelper.NewDescWithHelpAndLabels("events_queue_push_duration", "Time needed to push event to queue.", "pipeline"),

		ReloadsSuccesses: descHelper.NewDescWithHelpAndLabels("reloads_successes", "Number of successful pipeline reloads.", "pipeline"),
		ReloadsFailures:  descHelper.NewDescWithHelpAndLabels("reloads_failures", "Number of failed pipeline reloads.", "pipeline"),

		ReloadsLastSuccessTimestamp: descHelper.NewDescWithHelpAndLabels("reloads_last_success_timestamp", "Timestamp of last successful pipeline reload.", "pipeline"),
		ReloadsLastFailureTimestamp: descHelper.NewDescWithHelpAndLabels("reloads_last_failure_timestamp", "Timestamp of last failed pipeline reload.", "pipeline"),

		QueueEventsCount:         descHelper.NewDescWithHelpAndLabels("queue_events_count", "Number of events in the queue.", "pipeline"),
		QueueEventsQueueSize:     descHelper.NewDescWithHelpAndLabels("queue_events_queue_size", "Number of events that the queue can accommodate", "pipeline"),
		QueueMaxQueueSizeInBytes: descHelper.NewDescWithHelpAndLabels("queue_max_size_in_bytes", "Maximum size of given queue in bytes.", "pipeline"),

		PipelinePluginEventsIn:                descHelper.NewDescWithHelpAndLabels("plugin_events_in", "Number of events received this pipeline.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginEventsOut:               descHelper.NewDescWithHelpAndLabels("plugin_events_out", "Number of events output by this pipeline.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginEventsDuration:          descHelper.NewDescWithHelpAndLabels("plugin_events_duration", "Time spent processing events in this plugin.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginEventsQueuePushDuration: descHelper.NewDescWithHelpAndLabels("plugin_events_queue_push_duration", "Time spent pushing events into the input queue.", "pipeline", "plugin_type", "plugin", "plugin_id"),

		PipelinePluginDocumentsSuccesses:            descHelper.NewDescWithHelpAndLabels("plugin_documents_successes", "Number of successful bulk requests.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginDocumentsNonRetryableFailures: descHelper.NewDescWithHelpAndLabels("plugin_documents_non_retryable_failures", "Number of output events with non-retryable failures.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginBulkRequestErrors:             descHelper.NewDescWithHelpAndLabels("plugin_bulk_requests_errors", "Number of bulk request errors.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginBulkRequestResponses:          descHelper.NewDescWithHelpAndLabels("plugin_bulk_requests_responses", "Bulk request HTTP response counts by code.", "pipeline", "plugin_type", "plugin", "plugin_id", "code"),

		FlowInputCurrent:              descHelper.NewDescWithHelpAndLabels("flow_input_current", "Current number of events in the input queue.", "pipeline"),
		FlowInputLifetime:             descHelper.NewDescWithHelpAndLabels("flow_input_lifetime", "Lifetime number of events in the input queue.", "pipeline"),
		FlowFilterCurrent:             descHelper.NewDescWithHelpAndLabels("flow_filter_current", "Current number of events in the filter queue.", "pipeline"),
		FlowFilterLifetime:            descHelper.NewDescWithHelpAndLabels("flow_filter_lifetime", "Lifetime number of events in the filter queue.", "pipeline"),
		FlowOutputCurrent:             descHelper.NewDescWithHelpAndLabels("flow_output_current", "Current number of events in the output queue.", "pipeline"),
		FlowOutputLifetime:            descHelper.NewDescWithHelpAndLabels("flow_output_lifetime", "Lifetime number of events in the output queue.", "pipeline"),
		FlowQueueBackpressureCurrent:  descHelper.NewDescWithHelpAndLabels("flow_queue_backpressure_current", "Current number of events in the backpressure queue.", "pipeline"),
		FlowQueueBackpressureLifetime: descHelper.NewDescWithHelpAndLabels("flow_queue_backpressure_lifetime", "Lifetime number of events in the backpressure queue.", "pipeline"),
		FlowWorkerConcurrencyCurrent:  descHelper.NewDescWithHelpAndLabels("flow_worker_concurrency_current", "Current number of workers.", "pipeline"),
		FlowWorkerConcurrencyLifetime: descHelper.NewDescWithHelpAndLabels("flow_worker_concurrency_lifetime", "Lifetime number of workers.", "pipeline"),
	}
}

func (collector *PipelineSubcollector) Collect(pipeStats *responses.SinglePipelineResponse, pipelineID string, ch chan<- prometheus.Metric) {
	collectingStart := time.Now()
	log.Printf("collecting pipeline stats for pipeline %s", pipelineID)

	ch <- prometheus.MustNewConstMetric(collector.EventsOut, prometheus.CounterValue, float64(pipeStats.Events.Out), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsFiltered, prometheus.CounterValue, float64(pipeStats.Events.Filtered), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsIn, prometheus.CounterValue, float64(pipeStats.Events.In), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsDuration, prometheus.CounterValue, float64(pipeStats.Events.DurationInMillis), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsQueuePushDuration, prometheus.CounterValue, float64(pipeStats.Events.QueuePushDurationInMillis), pipelineID)

	ch <- prometheus.MustNewConstMetric(collector.Up, prometheus.GaugeValue, float64(collector.isPipelineHealthy(pipeStats.Reloads)), pipelineID)

	ch <- prometheus.MustNewConstMetric(collector.ReloadsSuccesses, prometheus.CounterValue, float64(pipeStats.Reloads.Successes), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.ReloadsFailures, prometheus.CounterValue, float64(pipeStats.Reloads.Failures), pipelineID)

	if pipeStats.Reloads.LastSuccessTimestamp != nil {
		ch <- prometheus.NewMetricWithTimestamp(*pipeStats.Reloads.LastSuccessTimestamp, prometheus.MustNewConstMetric(collector.ReloadsLastSuccessTimestamp, prometheus.GaugeValue, 1, pipelineID))
	}
	if pipeStats.Reloads.LastFailureTimestamp != nil {
		ch <- prometheus.NewMetricWithTimestamp(*pipeStats.Reloads.LastFailureTimestamp, prometheus.MustNewConstMetric(collector.ReloadsLastFailureTimestamp, prometheus.GaugeValue, 1, pipelineID))
	}

	ch <- prometheus.MustNewConstMetric(collector.QueueEventsCount, prometheus.CounterValue, float64(pipeStats.Queue.EventsCount), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.QueueEventsQueueSize, prometheus.CounterValue, float64(pipeStats.Queue.QueueSizeInBytes), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.QueueMaxQueueSizeInBytes, prometheus.CounterValue, float64(pipeStats.Queue.MaxQueueSizeInBytes), pipelineID)

	flowStats := pipeStats.Flow
	ch <- prometheus.MustNewConstMetric(collector.FlowInputCurrent, prometheus.GaugeValue, float64(flowStats.InputThroughput.Current), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowInputLifetime, prometheus.CounterValue, float64(flowStats.InputThroughput.Lifetime), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowFilterCurrent, prometheus.GaugeValue, float64(flowStats.FilterThroughput.Current), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowFilterLifetime, prometheus.CounterValue, float64(flowStats.FilterThroughput.Lifetime), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowOutputCurrent, prometheus.GaugeValue, float64(flowStats.OutputThroughput.Current), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowOutputLifetime, prometheus.CounterValue, float64(flowStats.OutputThroughput.Lifetime), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowQueueBackpressureCurrent, prometheus.GaugeValue, float64(flowStats.QueueBackpressure.Current), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowQueueBackpressureLifetime, prometheus.CounterValue, float64(flowStats.QueueBackpressure.Lifetime), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, float64(flowStats.WorkerConcurrency.Current), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.FlowWorkerConcurrencyLifetime, prometheus.CounterValue, float64(flowStats.WorkerConcurrency.Lifetime), pipelineID)

	// Output error metrics
	for _, output := range pipeStats.Plugins.Outputs {
		pluginID := output.ID
		pluginType := "output"
		log.Printf("collecting output error stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, output.Name, pluginID)

		// Response codes returned by output Bulk Requests
		for code, count := range output.BulkRequests.Responses {
			ch <- prometheus.MustNewConstMetric(collector.PipelinePluginBulkRequestResponses, prometheus.CounterValue, float64(count), pipelineID, pluginType, output.Name, pluginID, code)
		}

		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginDocumentsSuccesses, prometheus.CounterValue, float64(output.Documents.Successes), pipelineID, pluginType, output.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginDocumentsNonRetryableFailures, prometheus.CounterValue, float64(output.Documents.NonRetryableFailures), pipelineID, pluginType, output.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginBulkRequestErrors, prometheus.CounterValue, float64(output.BulkRequests.WithErrors), pipelineID, pluginType, output.Name, pluginID)
	}

	// Pipeline plugins metrics
	for _, plugin := range pipeStats.Plugins.Inputs {
		pluginType := "input"
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsQueuePushDuration, prometheus.CounterValue, float64(plugin.Events.QueuePushDurationInMillis), pipelineID, pluginType, plugin.Name, plugin.ID)
	}

	for _, plugin := range pipeStats.Plugins.Codecs {
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, "codec", plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Encode.WritesIn), pipelineID, "codec:encode", plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Decode.WritesIn), pipelineID, "codec:decode", plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Decode.Out), pipelineID, "codec:decode", plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Encode.DurationInMillis), pipelineID, "codec:encode", plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Decode.DurationInMillis), pipelineID, "codec:decode", plugin.Name, plugin.ID)
	}

	for _, plugin := range pipeStats.Plugins.Filters {
		pluginType := "filter"
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Events.In), pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Events.DurationInMillis), pipelineID, pluginType, plugin.Name, plugin.ID)
	}

	for _, plugin := range pipeStats.Plugins.Outputs {
		pluginType := "output"
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Events.In), pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, plugin.ID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Events.DurationInMillis), pipelineID, pluginType, plugin.Name, plugin.ID)
	}

	collectingEnd := time.Now()
	log.Printf("collected pipeline stats for pipeline %s in %s", pipelineID, collectingEnd.Sub(collectingStart))
}

// isPipelineHealthy returns 1 if the pipeline is healthy, 0 if it is not
// A pipeline is considered healthy if:
//  1. last_failure_timestamp is nil
//  2. last_success_timestamp > last_failure_timestamp
//  3. last_failure_timestamp and last_success_timestamp are either missing (likely due to version incompatibility)
//     or set to the same value (likely due to a bug in the pipeline):
//     lacking information, assume healthy
//
// A pipeline is considered unhealthy if:
//  1. last_failure_timestamp is not nil and last_success_timestamp is nil
//  2. last_failure_timestamp > last_success_timestamp
func (collector *PipelineSubcollector) isPipelineHealthy(pipeReloadStats responses.PipelineReloadResponse) float64 {
	if pipeReloadStats.LastFailureTimestamp == nil {
		return CollectorHealthy
	}

	if pipeReloadStats.LastFailureTimestamp != nil && pipeReloadStats.LastSuccessTimestamp == nil {
		return CollectorUnhealthy
	}

	if pipeReloadStats.LastSuccessTimestamp.Before(*pipeReloadStats.LastFailureTimestamp) {
		return CollectorUnhealthy
	}

	if pipeReloadStats.LastSuccessTimestamp.After(*pipeReloadStats.LastFailureTimestamp) {
		return CollectorHealthy
	}

	return CollectorHealthy
}

package nodestats

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/internal/prometheus_helper"
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

	DeadLetterQueueMaxSizeInBytes *prometheus.Desc
	DeadLetterQueueSizeInBytes    *prometheus.Desc
	DeadLetterQueueDroppedEvents  *prometheus.Desc
	DeadLetterQueueExpiredEvents  *prometheus.Desc
}

func NewPipelineSubcollector() *PipelineSubcollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: fmt.Sprintf("%s_pipeline", subsystem)}
	return &PipelineSubcollector{
		Up:                      descHelper.NewDesc("up", "Whether the pipeline is up or not.", "pipeline"),
		EventsOut:               descHelper.NewDesc("events_out", "Number of events that have been processed by this pipeline.", "pipeline"),
		EventsFiltered:          descHelper.NewDesc("events_filtered", "Number of events that have been filtered out by this pipeline.", "pipeline"),
		EventsIn:                descHelper.NewDesc("events_in", "Number of events that have been inputted into this pipeline.", "pipeline"),
		EventsDuration:          descHelper.NewDesc("events_duration", "Time needed to process event.", "pipeline"),
		EventsQueuePushDuration: descHelper.NewDesc("events_queue_push_duration", "Time needed to push event to queue.", "pipeline"),

		ReloadsSuccesses: descHelper.NewDesc("reloads_successes", "Number of successful pipeline reloads.", "pipeline"),
		ReloadsFailures:  descHelper.NewDesc("reloads_failures", "Number of failed pipeline reloads.", "pipeline"),

		ReloadsLastSuccessTimestamp: descHelper.NewDesc("reloads_last_success_timestamp", "Timestamp of last successful pipeline reload.", "pipeline"),
		ReloadsLastFailureTimestamp: descHelper.NewDesc("reloads_last_failure_timestamp", "Timestamp of last failed pipeline reload.", "pipeline"),

		QueueEventsCount:         descHelper.NewDesc("queue_events_count", "Number of events in the queue.", "pipeline"),
		QueueEventsQueueSize:     descHelper.NewDesc("queue_events_queue_size", "Number of events that the queue can accommodate", "pipeline"),
		QueueMaxQueueSizeInBytes: descHelper.NewDesc("queue_max_size_in_bytes", "Maximum size of given queue in bytes.", "pipeline"),

		PipelinePluginEventsIn:                descHelper.NewDesc("plugin_events_in", "Number of events received this pipeline.", "plugin_type", "plugin", "plugin_id", "pipeline"),
		PipelinePluginEventsOut:               descHelper.NewDesc("plugin_events_out", "Number of events output by this pipeline.", "plugin_type", "plugin", "plugin_id", "pipeline"),
		PipelinePluginEventsDuration:          descHelper.NewDesc("plugin_events_duration", "Time spent processing events in this plugin.", "plugin_type", "plugin", "plugin_id", "pipeline"),
		PipelinePluginEventsQueuePushDuration: descHelper.NewDesc("plugin_events_queue_push_duration", "Time spent pushing events into the input queue.", "plugin_type", "plugin", "plugin_id", "pipeline"),

		PipelinePluginDocumentsSuccesses:            descHelper.NewDesc("plugin_documents_successes", "Number of successful bulk requests.", "plugin_type", "plugin", "plugin_id", "pipeline"),
		PipelinePluginDocumentsNonRetryableFailures: descHelper.NewDesc("plugin_documents_non_retryable_failures", "Number of output events with non-retryable failures.", "plugin_type", "plugin", "plugin_id", "pipeline"),
		PipelinePluginBulkRequestErrors:             descHelper.NewDesc("plugin_bulk_requests_errors", "Number of bulk request errors.", "plugin_type", "plugin", "plugin_id", "pipeline"),
		PipelinePluginBulkRequestResponses:          descHelper.NewDesc("plugin_bulk_requests_responses", "Bulk request HTTP response counts by code.", "plugin_type", "plugin", "plugin_id", "code", "pipeline"),

		FlowInputCurrent:              descHelper.NewDesc("flow_input_current", "Current number of events in the input queue.", "pipeline"),
		FlowInputLifetime:             descHelper.NewDesc("flow_input_lifetime", "Lifetime number of events in the input queue.", "pipeline"),
		FlowFilterCurrent:             descHelper.NewDesc("flow_filter_current", "Current number of events in the filter queue.", "pipeline"),
		FlowFilterLifetime:            descHelper.NewDesc("flow_filter_lifetime", "Lifetime number of events in the filter queue.", "pipeline"),
		FlowOutputCurrent:             descHelper.NewDesc("flow_output_current", "Current number of events in the output queue.", "pipeline"),
		FlowOutputLifetime:            descHelper.NewDesc("flow_output_lifetime", "Lifetime number of events in the output queue.", "pipeline"),
		FlowQueueBackpressureCurrent:  descHelper.NewDesc("flow_queue_backpressure_current", "Current number of events in the backpressure queue.", "pipeline"),
		FlowQueueBackpressureLifetime: descHelper.NewDesc("flow_queue_backpressure_lifetime", "Lifetime number of events in the backpressure queue.", "pipeline"),
		FlowWorkerConcurrencyCurrent:  descHelper.NewDesc("flow_worker_concurrency_current", "Current number of workers.", "pipeline"),
		FlowWorkerConcurrencyLifetime: descHelper.NewDesc("flow_worker_concurrency_lifetime", "Lifetime number of workers.", "pipeline"),

		DeadLetterQueueMaxSizeInBytes: descHelper.NewDesc("dead_letter_queue_max_size_in_bytes", "Maximum size of the dead letter queue in bytes.", "pipeline"),
		DeadLetterQueueSizeInBytes:    descHelper.NewDesc("dead_letter_queue_size_in_bytes", "Current size of the dead letter queue in bytes.", "pipeline"),
		DeadLetterQueueDroppedEvents:  descHelper.NewDesc("dead_letter_queue_dropped_events", "Number of events dropped by the dead letter queue.", "pipeline"),
		DeadLetterQueueExpiredEvents:  descHelper.NewDesc("dead_letter_queue_expired_events", "Number of events expired in the dead letter queue.", "pipeline"),
	}
}

func (subcollector *PipelineSubcollector) Collect(pipeStats *responses.SinglePipelineResponse, pipelineID string, ch chan<- prometheus.Metric, endpoint string) {
	collectingStart := time.Now()
	slog.Debug("collecting pipeline stats for pipeline", "pipelineID", pipelineID)

	metricsHelper := prometheus_helper.SimpleMetricsHelper{Channel: ch, Labels: []string{pipelineID, endpoint}}

	// ***** EVENTS *****
	metricsHelper.NewIntMetric(subcollector.EventsOut, prometheus.CounterValue, pipeStats.Events.Out)
	metricsHelper.NewIntMetric(subcollector.EventsFiltered, prometheus.CounterValue, pipeStats.Events.Filtered)
	metricsHelper.NewIntMetric(subcollector.EventsIn, prometheus.CounterValue, pipeStats.Events.In)
	metricsHelper.NewIntMetric(subcollector.EventsDuration, prometheus.GaugeValue, pipeStats.Events.DurationInMillis)
	metricsHelper.NewIntMetric(subcollector.EventsQueuePushDuration, prometheus.GaugeValue, pipeStats.Events.QueuePushDurationInMillis)
	// ******************

	// ***** UP *****
	metricsHelper.NewFloatMetric(subcollector.Up, prometheus.GaugeValue, subcollector.isPipelineHealthy(pipeStats.Reloads))
	// **************

	// ***** RELOADS *****
	metricsHelper.NewIntMetric(subcollector.ReloadsSuccesses, prometheus.CounterValue, pipeStats.Reloads.Successes)
	metricsHelper.NewIntMetric(subcollector.ReloadsFailures, prometheus.CounterValue, pipeStats.Reloads.Failures)

	if pipeStats.Reloads.LastSuccessTimestamp != nil {
		metricsHelper.NewTimestampMetric(subcollector.ReloadsLastSuccessTimestamp, prometheus.GaugeValue, *pipeStats.Reloads.LastSuccessTimestamp)
	}
	if pipeStats.Reloads.LastFailureTimestamp != nil {
		metricsHelper.NewTimestampMetric(subcollector.ReloadsLastFailureTimestamp, prometheus.GaugeValue, *pipeStats.Reloads.LastFailureTimestamp)
	}
	// *******************

	// ***** QUEUE *****
	metricsHelper.NewUInt64Metric(subcollector.QueueEventsCount, prometheus.CounterValue, pipeStats.Queue.EventsCount)
	metricsHelper.NewUInt64Metric(subcollector.QueueEventsQueueSize, prometheus.GaugeValue, pipeStats.Queue.QueueSizeInBytes)
	metricsHelper.NewUInt64Metric(subcollector.QueueMaxQueueSizeInBytes, prometheus.GaugeValue, pipeStats.Queue.MaxQueueSizeInBytes)
	// *****************

	// ***** FLOW *****
	flowStats := pipeStats.Flow
	metricsHelper.NewFloatMetric(subcollector.FlowInputCurrent, prometheus.GaugeValue, flowStats.InputThroughput.Current)
	metricsHelper.NewFloatMetric(subcollector.FlowInputLifetime, prometheus.CounterValue, flowStats.InputThroughput.Lifetime)
	metricsHelper.NewFloatMetric(subcollector.FlowFilterCurrent, prometheus.GaugeValue, flowStats.FilterThroughput.Current)
	metricsHelper.NewFloatMetric(subcollector.FlowFilterLifetime, prometheus.CounterValue, flowStats.FilterThroughput.Lifetime)
	metricsHelper.NewFloatMetric(subcollector.FlowOutputCurrent, prometheus.GaugeValue, flowStats.OutputThroughput.Current)
	metricsHelper.NewFloatMetric(subcollector.FlowOutputLifetime, prometheus.CounterValue, flowStats.OutputThroughput.Lifetime)
	metricsHelper.NewFloatMetric(subcollector.FlowQueueBackpressureCurrent, prometheus.GaugeValue, flowStats.QueueBackpressure.Current)
	metricsHelper.NewFloatMetric(subcollector.FlowQueueBackpressureLifetime, prometheus.CounterValue, flowStats.QueueBackpressure.Lifetime)
	metricsHelper.NewFloatMetric(subcollector.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, flowStats.WorkerConcurrency.Current)
	metricsHelper.NewFloatMetric(subcollector.FlowWorkerConcurrencyLifetime, prometheus.CounterValue, flowStats.WorkerConcurrency.Lifetime)
	// ****************

	// ***** DEAD LETTER QUEUE *****
	deadLetterQueueStats := pipeStats.DeadLetterQueue
	metricsHelper.NewIntMetric(subcollector.DeadLetterQueueMaxSizeInBytes, prometheus.GaugeValue, deadLetterQueueStats.MaxQueueSizeInBytes)
	metricsHelper.NewUInt64Metric(subcollector.DeadLetterQueueSizeInBytes, prometheus.GaugeValue, deadLetterQueueStats.QueueSizeInBytes)
	metricsHelper.NewUInt64Metric(subcollector.DeadLetterQueueDroppedEvents, prometheus.CounterValue, deadLetterQueueStats.DroppedEvents)
	metricsHelper.NewUInt64Metric(subcollector.DeadLetterQueueExpiredEvents, prometheus.CounterValue, deadLetterQueueStats.ExpiredEvents)
	// *****************************

	// ===== PLUGINS =====
	// ***** OUTPUTS *****
	for _, plugin := range pipeStats.Plugins.Outputs {
		pluginType := "output"
		slog.Debug("collecting outputs stats for pipeline", "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "pipelineID", pipelineID, "endpoint", endpoint)

		// Response codes returned by output Bulk Requests
		for code, count := range plugin.BulkRequests.Responses {
			metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, code, pipelineID, endpoint}
			metricsHelper.NewIntMetric(subcollector.PipelinePluginBulkRequestResponses, prometheus.CounterValue, count)
		}

		metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, pipelineID, endpoint}
		metricsHelper.NewIntMetric(subcollector.PipelinePluginDocumentsSuccesses, prometheus.CounterValue, plugin.Documents.Successes)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginDocumentsNonRetryableFailures, prometheus.CounterValue, plugin.Documents.NonRetryableFailures)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginBulkRequestErrors, prometheus.CounterValue, plugin.BulkRequests.WithErrors)
	}
	// *******************

	// ***** INPUTS *****
	for _, plugin := range pipeStats.Plugins.Inputs {
		pluginType := "input"
		slog.Debug("collecting pipeline plugin stats for pipeline", "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "pipelineID", pipelineID, "endpoint", endpoint)

		metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, pipelineID, endpoint}
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsOut, prometheus.CounterValue, plugin.Events.Out)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsQueuePushDuration, prometheus.GaugeValue, plugin.Events.QueuePushDurationInMillis)
	}
	// ******************

	// ***** CODECS *****
	for _, plugin := range pipeStats.Plugins.Codecs {
		pluginType := "codec"
		slog.Debug("collecting pipeline plugin stats for pipeline", "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "pipelineID", pipelineID, "endpoint", endpoint)

		pluginType = "codec:encode"
		metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, pipelineID, endpoint}
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsIn, prometheus.CounterValue, plugin.Encode.WritesIn)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsDuration, prometheus.CounterValue, plugin.Encode.DurationInMillis)

		pluginType = "codec:decode"
		metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, pipelineID, endpoint}
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsIn, prometheus.CounterValue, plugin.Decode.WritesIn)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsOut, prometheus.CounterValue, plugin.Decode.Out)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsDuration, prometheus.CounterValue, plugin.Decode.DurationInMillis)
	}
	// ******************

	// ***** FILTERS *****
	for _, plugin := range pipeStats.Plugins.Filters {
		pluginType := "filter"
		slog.Debug("collecting pipeline plugin stats for pipeline", "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "pipelineID", pipelineID, "endpoint", endpoint)

		metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, pipelineID, endpoint}
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsIn, prometheus.CounterValue, plugin.Events.In)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsOut, prometheus.CounterValue, plugin.Events.Out)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsDuration, prometheus.CounterValue, plugin.Events.DurationInMillis)
	}
	// *******************

	// ***** OUTPUTS *****
	for _, plugin := range pipeStats.Plugins.Outputs {
		pluginType := "output"
		slog.Debug("collecting pipeline plugin stats for pipeline", "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "pipelineID", pipelineID, "endpoint", endpoint)

		metricsHelper.Labels = []string{pluginType, plugin.Name, plugin.ID, pipelineID, endpoint}
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsIn, prometheus.CounterValue, plugin.Events.In)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsOut, prometheus.CounterValue, plugin.Events.Out)
		metricsHelper.NewIntMetric(subcollector.PipelinePluginEventsDuration, prometheus.CounterValue, plugin.Events.DurationInMillis)
	}
	// *******************
	// ===================

	collectingEnd := time.Now()
	slog.Debug("collected pipeline stats for pipeline", "duration", collectingEnd.Sub(collectingStart), "pipelineID", pipelineID, "endpoint", endpoint)
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
func (subcollector *PipelineSubcollector) isPipelineHealthy(pipeReloadStats responses.PipelineReloadResponse) float64 {
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

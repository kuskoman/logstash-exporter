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

		PipelinePluginEventsIn:                descHelper.NewDesc("plugin_events_in", "Number of events received this pipeline.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginEventsOut:               descHelper.NewDesc("plugin_events_out", "Number of events output by this pipeline.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginEventsDuration:          descHelper.NewDesc("plugin_events_duration", "Time spent processing events in this plugin.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginEventsQueuePushDuration: descHelper.NewDesc("plugin_events_queue_push_duration", "Time spent pushing events into the input queue.", "pipeline", "plugin_type", "plugin", "plugin_id"),

		PipelinePluginDocumentsSuccesses:            descHelper.NewDesc("plugin_documents_successes", "Number of successful bulk requests.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginDocumentsNonRetryableFailures: descHelper.NewDesc("plugin_documents_non_retryable_failures", "Number of output events with non-retryable failures.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginBulkRequestErrors:             descHelper.NewDesc("plugin_bulk_requests_errors", "Number of bulk request errors.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginBulkRequestResponses:          descHelper.NewDesc("plugin_bulk_requests_responses", "Bulk request HTTP response counts by code.", "pipeline", "plugin_type", "plugin", "plugin_id", "code"),

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

func (collector *PipelineSubcollector) Collect(pipeStats *responses.SinglePipelineResponse, pipelineID string, ch chan<- prometheus.Metric, endpoint string) {
	collectingStart := time.Now()
	slog.Debug("collecting pipeline stats for pipeline", "pipelineID", pipelineID)

	newFloatMetric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value float64, labels ...string) {
		labels = append(labels, pipelineID, endpoint)
		metric := prometheus.MustNewConstMetric(desc, metricType, value, labels...)

		ch <- metric
	}

	newTimestampMetric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value time.Time, labels ...string) {
		labels = append(labels, pipelineID, endpoint)
		metric := prometheus.NewMetricWithTimestamp(value, prometheus.MustNewConstMetric(desc, metricType, 1, labels...))

		ch <- metric
	}

	newIntMetric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value int, labels ...string) {
		newFloatMetric(desc, metricType, float64(value), labels...)
	}

	newInt64Metric := func(desc *prometheus.Desc, metricType prometheus.ValueType, value int64, labels ...string) {
		newFloatMetric(desc, metricType, float64(value), labels...)
	}

	newIntMetric(collector.EventsOut, prometheus.CounterValue, pipeStats.Events.Out)
	newIntMetric(collector.EventsFiltered, prometheus.CounterValue, pipeStats.Events.Filtered)
	newIntMetric(collector.EventsIn, prometheus.CounterValue, pipeStats.Events.In)
	newIntMetric(collector.EventsDuration, prometheus.CounterValue, pipeStats.Events.DurationInMillis)
	newIntMetric(collector.EventsQueuePushDuration, prometheus.CounterValue, pipeStats.Events.QueuePushDurationInMillis)

	newFloatMetric(collector.Up, prometheus.GaugeValue, collector.isPipelineHealthy(pipeStats.Reloads))

	newIntMetric(collector.ReloadsSuccesses, prometheus.CounterValue, pipeStats.Reloads.Successes)
	newIntMetric(collector.ReloadsFailures, prometheus.CounterValue, pipeStats.Reloads.Failures)

	if pipeStats.Reloads.LastSuccessTimestamp != nil {
		newTimestampMetric(collector.ReloadsLastSuccessTimestamp, prometheus.GaugeValue, *pipeStats.Reloads.LastSuccessTimestamp)
	}
	if pipeStats.Reloads.LastFailureTimestamp != nil {
		newTimestampMetric(collector.ReloadsLastFailureTimestamp, prometheus.GaugeValue, *pipeStats.Reloads.LastFailureTimestamp)
	}

	newInt64Metric(collector.QueueEventsCount, prometheus.CounterValue, pipeStats.Queue.EventsCount)
	newInt64Metric(collector.QueueEventsQueueSize, prometheus.CounterValue, pipeStats.Queue.QueueSizeInBytes)
	newInt64Metric(collector.QueueMaxQueueSizeInBytes, prometheus.CounterValue, pipeStats.Queue.MaxQueueSizeInBytes)

	flowStats := pipeStats.Flow
	newFloatMetric(collector.FlowInputCurrent, prometheus.GaugeValue, flowStats.InputThroughput.Current)
	newFloatMetric(collector.FlowInputLifetime, prometheus.CounterValue, flowStats.InputThroughput.Lifetime)
	newFloatMetric(collector.FlowFilterCurrent, prometheus.GaugeValue, flowStats.FilterThroughput.Current)
	newFloatMetric(collector.FlowFilterLifetime, prometheus.CounterValue, flowStats.FilterThroughput.Lifetime)
	newFloatMetric(collector.FlowOutputCurrent, prometheus.GaugeValue, flowStats.OutputThroughput.Current)
	newFloatMetric(collector.FlowOutputLifetime, prometheus.CounterValue, flowStats.OutputThroughput.Lifetime)
	newFloatMetric(collector.FlowQueueBackpressureCurrent, prometheus.GaugeValue, flowStats.QueueBackpressure.Current)
	newFloatMetric(collector.FlowQueueBackpressureLifetime, prometheus.CounterValue, flowStats.QueueBackpressure.Lifetime)
	newFloatMetric(collector.FlowWorkerConcurrencyCurrent, prometheus.GaugeValue, flowStats.WorkerConcurrency.Current)
	newFloatMetric(collector.FlowWorkerConcurrencyLifetime, prometheus.CounterValue, flowStats.WorkerConcurrency.Lifetime)

	deadLetterQueueStats := pipeStats.DeadLetterQueue
	newIntMetric(collector.DeadLetterQueueMaxSizeInBytes, prometheus.GaugeValue, deadLetterQueueStats.MaxQueueSizeInBytes)
	newInt64Metric(collector.DeadLetterQueueSizeInBytes, prometheus.GaugeValue, deadLetterQueueStats.QueueSizeInBytes)
	newInt64Metric(collector.DeadLetterQueueDroppedEvents, prometheus.CounterValue, deadLetterQueueStats.DroppedEvents)
	newInt64Metric(collector.DeadLetterQueueExpiredEvents, prometheus.CounterValue, deadLetterQueueStats.ExpiredEvents)

	// Output error metrics
	for _, output := range pipeStats.Plugins.Outputs {
		pluginID := output.ID
		pluginType := "output"
		slog.Debug("collecting output error stats for pipeline", "pipelineID", pipelineID, "plugin type", pluginType, "name", output.Name, "id", pluginID)

		// Response codes returned by output Bulk Requests
		for code, count := range output.BulkRequests.Responses {
			ch <- prometheus.MustNewConstMetric(collector.PipelinePluginBulkRequestResponses, prometheus.CounterValue, float64(count), pipelineID, pluginType, output.Name, pluginID, code, endpoint)
		}

		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginDocumentsSuccesses, prometheus.CounterValue, float64(output.Documents.Successes), pipelineID, pluginType, output.Name, pluginID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginDocumentsNonRetryableFailures, prometheus.CounterValue, float64(output.Documents.NonRetryableFailures), pipelineID, pluginType, output.Name, pluginID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginBulkRequestErrors, prometheus.CounterValue, float64(output.BulkRequests.WithErrors), pipelineID, pluginType, output.Name, pluginID, endpoint)
	}

	// Pipeline plugins metrics
	for _, plugin := range pipeStats.Plugins.Inputs {
		pluginType := "input"
		slog.Debug("collecting pipeline plugin stats for pipeline", "pipelineID", pipelineID, "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "endpoint", endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsQueuePushDuration, prometheus.CounterValue, float64(plugin.Events.QueuePushDurationInMillis), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
	}

	for _, plugin := range pipeStats.Plugins.Codecs {
		slog.Debug("collecting pipeline plugin stats for pipeline", "pipelineID", pipelineID, "plugin type", "codec", "name", plugin.Name, "id", plugin.ID, "endpoint", endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Encode.WritesIn), pipelineID, "codec:encode", plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Decode.WritesIn), pipelineID, "codec:decode", plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Decode.Out), pipelineID, "codec:decode", plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Encode.DurationInMillis), pipelineID, "codec:encode", plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Decode.DurationInMillis), pipelineID, "codec:decode", plugin.Name, plugin.ID, endpoint)
	}

	for _, plugin := range pipeStats.Plugins.Filters {
		pluginType := "filter"
		slog.Debug("collecting pipeline plugin stats for pipeline", "pipelineID", pipelineID, "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "endpoint", endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Events.In), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Events.DurationInMillis), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
	}

	for _, plugin := range pipeStats.Plugins.Outputs {
		pluginType := "output"
		slog.Debug("collecting pipeline plugin stats for pipeline", "pipelineID", pipelineID, "plugin type", pluginType, "name", plugin.Name, "id", plugin.ID, "endpoint", endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Events.In), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Events.DurationInMillis), pipelineID, pluginType, plugin.Name, plugin.ID, endpoint)
	}

	collectingEnd := time.Now()
	slog.Debug("collected pipeline stats for pipeline", "pipelineID", pipelineID, "duration", collectingEnd.Sub(collectingStart), "endpoint", endpoint)
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

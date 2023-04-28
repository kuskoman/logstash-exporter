package nodestats

import (
	"fmt"
	"log"
	"strings"
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

	PipelinePluginDocumentsSuccesses    *prometheus.Desc
	PipelinePluginDocumentsNonRetryableFailures *prometheus.Desc
	PipelinePluginBulkRequestErrors    *prometheus.Desc
	PipelinePluginBulkRequestResponses *prometheus.Desc
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

		PipelinePluginDocumentsSuccesses: descHelper.NewDescWithHelpAndLabels("plugin_documents_successes", "Number of successful bulk requests.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginDocumentsNonRetryableFailures: descHelper.NewDescWithHelpAndLabels("plugin_documents_non_retryable_failures", "Number of output events with non-retryable failures.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginBulkRequestErrors:    descHelper.NewDescWithHelpAndLabels("plugin_bulk_requests_errors", "Number of bulk request errors.", "pipeline", "plugin_type", "plugin", "plugin_id"),
		PipelinePluginBulkRequestResponses: descHelper.NewDescWithHelpAndLabels("plugin_bulk_requests_responses", "Bulk request HTTP response counts by code.", "pipeline", "plugin_type", "plugin", "plugin_id", "code"),
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

	// Output error metrics
	for _, output := range pipeStats.Plugins.Outputs {
		pluginID := TruncatePluginId(output.ID)
		pluginType := "output"
		log.Printf("collecting output error stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, output.Name, pluginID)
		nonRetryableFailures := 0
		bulkRequestSuccesses := 0
		bulkRequestErrors := 0
		if &output.Documents != nil {
			bulkRequestSuccesses = output.Documents.Successes
			nonRetryableFailures = output.Documents.NonRetryableFailures
		}
		// Response codes returned by output Bulk Requests
		if &output.BulkRequests != nil {
			bulkRequestErrors = output.BulkRequests.WithErrors
			for code, count := range output.BulkRequests.Responses {
				ch <- prometheus.MustNewConstMetric(collector.PipelinePluginBulkRequestResponses, prometheus.CounterValue, float64(count), pipelineID, pluginType, output.Name, pluginID, code)
			}
		}
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginDocumentsSuccesses, prometheus.CounterValue, float64(bulkRequestSuccesses), pipelineID, pluginType, output.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginDocumentsNonRetryableFailures, prometheus.CounterValue, float64(nonRetryableFailures), pipelineID, pluginType, output.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginBulkRequestErrors, prometheus.CounterValue, float64(bulkRequestErrors), pipelineID, pluginType, output.Name, pluginID)
	}

	// Pipeline plugins metrics
	for _, plugin := range pipeStats.Plugins.Inputs {
		pluginID := TruncatePluginId(plugin.ID)
		pluginType := "input"
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsQueuePushDuration, prometheus.CounterValue, float64(plugin.Events.QueuePushDurationInMillis), pipelineID, pluginType, plugin.Name, pluginID)
	}

	for _, plugin := range pipeStats.Plugins.Codecs {
		pluginID := TruncatePluginId(plugin.ID)
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, "codec", plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Encode.WritesIn), pipelineID, "codec:encode", plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Decode.WritesIn), pipelineID, "codec:decode", plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Decode.Out), pipelineID, "codec:decode", plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Encode.DurationInMillis), pipelineID, "codec:encode", plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Decode.DurationInMillis), pipelineID, "codec:decode", plugin.Name, pluginID)
	}

	for _, plugin := range pipeStats.Plugins.Filters {
		pluginID := TruncatePluginId(plugin.ID)
		pluginType := "filter"
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Events.In), pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Events.DurationInMillis), pipelineID, pluginType, plugin.Name, pluginID)
	}

	for _, plugin := range pipeStats.Plugins.Outputs {
		pluginID := TruncatePluginId(plugin.ID)
		pluginType := "output"
		log.Printf("collecting pipeline plugin stats for pipeline %s :: plugin type:%s name:%s id:%s", pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsIn, prometheus.CounterValue, float64(plugin.Events.In), pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsOut, prometheus.CounterValue, float64(plugin.Events.Out), pipelineID, pluginType, plugin.Name, pluginID)
		ch <- prometheus.MustNewConstMetric(collector.PipelinePluginEventsDuration, prometheus.CounterValue, float64(plugin.Events.DurationInMillis), pipelineID, pluginType, plugin.Name, pluginID)
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

// Plugins have non-unique names, so use both name and id as labels
// By default ids are a 36-char UUID, optionally prefixed the a plugin type, or a 64-char SHA256 hash
// If the id is set by the user, keep it. If it's a UUID, truncate it to the last 8 chars (1% chance of collision per 9291)
func TruncatePluginId(pluginID string) string {
	// If the pluginId is < 32 chars, it's likely a user-defined id.
	if len(pluginID) < 32 {
		return pluginID
	}
	noDashes := strings.Replace(pluginID, "-", "", -1)
	return noDashes[len(noDashes)-8:]
}

package nodestats

import (
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
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
}

func NewPipelineSubcollector() *PipelineSubcollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: fmt.Sprintf("%s_pipeline", subsystem)}
	return &PipelineSubcollector{
		Up:                      descHelper.NewDescWithHelpAndLabel("up", "Whether the pipeline is up or not.", "pipeline_id"),
		EventsOut:               descHelper.NewDescWithHelpAndLabel("events_out", "Number of events that have been processed by this pipeline.", "pipeline_id"),
		EventsFiltered:          descHelper.NewDescWithHelpAndLabel("events_filtered", "Number of events that have been filtered out by this pipeline.", "pipeline_id"),
		EventsIn:                descHelper.NewDescWithHelpAndLabel("events_in", "Number of events that have been inputted into this pipeline.", "pipeline_id"),
		EventsDuration:          descHelper.NewDescWithHelpAndLabel("events_duration", "Time needed to process event.", "pipeline_id"),
		EventsQueuePushDuration: descHelper.NewDescWithHelpAndLabel("events_queue_push_duration", "Time needed to push event to queue.", "pipeline_id"),

		ReloadsSuccesses: descHelper.NewDescWithHelpAndLabel("reloads_successes", "Number of successful pipeline reloads.", "pipeline_id"),
		ReloadsFailures:  descHelper.NewDescWithHelpAndLabel("reloads_failures", "Number of failed pipeline reloads.", "pipeline_id"),

		ReloadsLastSuccessTimestamp: descHelper.NewDescWithHelpAndLabel("reloads_last_success_timestamp", "Timestamp of last successful pipeline reload.", "pipeline_id"),
		ReloadsLastFailureTimestamp: descHelper.NewDescWithHelpAndLabel("reloads_last_failure_timestamp", "Timestamp of last failed pipeline reload.", "pipeline_id"),

		QueueEventsCount:         descHelper.NewDescWithHelpAndLabel("queue_events_count", "Number of events in the queue.", "pipeline_id"),
		QueueEventsQueueSize:     descHelper.NewDescWithHelpAndLabel("queue_events_queue_size", "Number of events that the queue can accommodate", "pipeline_id"),
		QueueMaxQueueSizeInBytes: descHelper.NewDescWithHelpAndLabel("queue_max_size_in_bytes", "Maximum size of given queue in bytes.", "pipeline_id"),
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

	collectingEnd := time.Now()
	log.Printf("collected pipeline stats for pipeline %s in %s", pipelineID, collectingEnd.Sub(collectingStart))
}

func (collector *PipelineSubcollector) isPipelineHealthy(pipeReloadStats responses.PipelineReloadResponse) float64 {
	// 1. If last failure timestamp (or both) are nil, the pipeline is healthy
	if pipeReloadStats.LastFailureTimestamp == nil {
		return 1
	// 2. If last_failure_timestamp is set and last success timestamp is nil, the pipeline is unhealthy
	} else if pipeReloadStats.LastFailureTimestamp != nil && pipeReloadStats.LastSuccessTimestamp == nil {
		return 0
	// 3. If last_success_timestamp < last_failure_timestamp, the pipeline is unhealthy
	} else if pipeReloadStats.LastSuccessTimestamp.Before(*pipeReloadStats.LastFailureTimestamp) {
		return 0
	// 4. If last_success_timestamp > last_failure_timestamp, the pipeline is healthy
	} else if pipeReloadStats.LastSuccessTimestamp.After(*pipeReloadStats.LastFailureTimestamp) {
		return 1
	}
	// Missing field, likely due to version incompatibility - lacking information, assume healthy
	return 1
}

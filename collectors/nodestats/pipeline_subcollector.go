package nodestats

import (
	"log"
	"time"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/prometheus/client_golang/prometheus"
)

type PipelineSubcollector struct {
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
	return &PipelineSubcollector{
		EventsOut:               descHelper.NewDescWithLabels("pipeline_events_out", []string{"pipeline_id"}),
		EventsFiltered:          descHelper.NewDescWithLabels("pipeline_events_filtered", []string{"pipeline_id"}),
		EventsIn:                descHelper.NewDescWithLabels("pipeline_events_in", []string{"pipeline_id"}),
		EventsDuration:          descHelper.NewDescWithLabels("pipeline_events_duration", []string{"pipeline_id"}),
		EventsQueuePushDuration: descHelper.NewDescWithLabels("pipeline_events_queue_push_duration", []string{"pipeline_id"}),

		ReloadsSuccesses: descHelper.NewDescWithLabels("pipeline_reloads_successes", []string{"pipeline_id"}),
		ReloadsFailures:  descHelper.NewDescWithLabels("pipeline_reloads_failures", []string{"pipeline_id"}),

		QueueEventsCount:         descHelper.NewDescWithLabels("pipeline_queue_events_count", []string{"pipeline_id"}),
		QueueEventsQueueSize:     descHelper.NewDescWithLabels("pipeline_queue_events_queue_size", []string{"pipeline_id"}),
		QueueMaxQueueSizeInBytes: descHelper.NewDescWithLabels("pipeline_queue_max_size_in_bytes", []string{"pipeline_id"}),
	}
}

func (collector *PipelineSubcollector) Collect(pipeStats *responses.SinglePipelineResponse, pipelineID string, ch chan<- prometheus.Metric) error {
	collectingStart := time.Now()
	log.Printf("collecting pipeline stats for pipeline %s", pipelineID)

	ch <- prometheus.MustNewConstMetric(collector.EventsOut, prometheus.CounterValue, float64(pipeStats.Events.Out), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsFiltered, prometheus.CounterValue, float64(pipeStats.Events.Filtered), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsIn, prometheus.CounterValue, float64(pipeStats.Events.In), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsDuration, prometheus.CounterValue, float64(pipeStats.Events.DurationInMillis), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsQueuePushDuration, prometheus.CounterValue, float64(pipeStats.Events.QueuePushDurationInMillis), pipelineID)

	// todo: add restart timestamps

	ch <- prometheus.MustNewConstMetric(collector.ReloadsSuccesses, prometheus.CounterValue, float64(pipeStats.Reloads.Successes), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.ReloadsFailures, prometheus.CounterValue, float64(pipeStats.Reloads.Failures), pipelineID)

	ch <- prometheus.MustNewConstMetric(collector.QueueEventsCount, prometheus.CounterValue, float64(pipeStats.Queue.EventsCount), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.QueueEventsQueueSize, prometheus.CounterValue, float64(pipeStats.Queue.QueueSizeInBytes), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.QueueMaxQueueSizeInBytes, prometheus.CounterValue, float64(pipeStats.Queue.MaxQueueSizeInBytes), pipelineID)

	collectingEnd := time.Now()
	log.Printf("collected pipeline stats for pipeline %s in %s", pipelineID, collectingEnd.Sub(collectingStart))
	return nil
}

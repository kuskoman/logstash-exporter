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

	InputsEventsOut               *prometheus.Desc
	InputsEventsQueuePushDuration *prometheus.Desc

	FiltersEventsIn       *prometheus.Desc
	FiltersEventsOut      *prometheus.Desc
	FiltersEventsDuration *prometheus.Desc

	OutputsEventsIn       *prometheus.Desc
	OutputsEventsOut      *prometheus.Desc
	OutputsEventsDuration *prometheus.Desc
}

func NewPipelineSubcollector() *PipelineSubcollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: fmt.Sprintf("%s_pipeline", subsystem)}
	return &PipelineSubcollector{
		EventsOut:               descHelper.NewDescWithHelpAndLabels("events_out", "Number of events that have been processed by this pipeline.", "pipeline_id"),
		EventsFiltered:          descHelper.NewDescWithHelpAndLabels("events_filtered", "Number of events that have been filtered out by this pipeline.", "pipeline_id"),
		EventsIn:                descHelper.NewDescWithHelpAndLabels("events_in", "Number of events that have been inputted into this pipeline.", "pipeline_id"),
		EventsDuration:          descHelper.NewDescWithHelpAndLabels("events_duration", "Time needed to process event.", "pipeline_id"),
		EventsQueuePushDuration: descHelper.NewDescWithHelpAndLabels("events_queue_push_duration", "Time needed to push event to queue.", "pipeline_id"),

		ReloadsSuccesses: descHelper.NewDescWithHelpAndLabels("reloads_successes", "Number of successful pipeline reloads.", "pipeline_id"),
		ReloadsFailures:  descHelper.NewDescWithHelpAndLabels("reloads_failures", "Number of failed pipeline reloads.", "pipeline_id"),

		ReloadsLastSuccessTimestamp: descHelper.NewDescWithHelpAndLabels("reloads_last_success_timestamp", "Timestamp of last successful pipeline reload.", "pipeline_id"),
		ReloadsLastFailureTimestamp: descHelper.NewDescWithHelpAndLabels("reloads_last_failure_timestamp", "Timestamp of last failed pipeline reload.", "pipeline_id"),

		QueueEventsCount:         descHelper.NewDescWithHelpAndLabels("queue_events_count", "Number of events in the queue.", "pipeline_id"),
		QueueEventsQueueSize:     descHelper.NewDescWithHelpAndLabels("queue_events_queue_size", "Number of events that the queue can accommodate", "pipeline_id"),
		QueueMaxQueueSizeInBytes: descHelper.NewDescWithHelpAndLabels("queue_max_size_in_bytes", "Maximum size of given queue in bytes.", "pipeline_id"),

		InputsEventsOut:               descHelper.NewDescWithHelpAndLabels("inputs_events_out", "Number of input events that have been processed by this pipeline.", "pipeline_id", "input"),
		InputsEventsQueuePushDuration: descHelper.NewDescWithHelpAndLabels("inputs_events_duration", "Time spent processing input events.", "pipeline_id", "input"),

		FiltersEventsIn:       descHelper.NewDescWithHelpAndLabels("filters_events_in", "Number of filter events that have been processed by this pipeline.", "pipeline_id", "input"),
		FiltersEventsOut:      descHelper.NewDescWithHelpAndLabels("filters_events_out", "Number of filter events that have been processed by this pipeline.", "pipeline_id", "input"),
		FiltersEventsDuration: descHelper.NewDescWithHelpAndLabels("filters_events_duration", "Time spent processing filter events", "pipeline_id", "input"),

		OutputsEventsIn:       descHelper.NewDescWithHelpAndLabels("outputs_events_in", "Number of output events that have been processed by this pipeline.", "pipeline_id", "input"),
		OutputsEventsOut:      descHelper.NewDescWithHelpAndLabels("outputs_events_out", "Number of output events that have been processed by this pipeline.", "pipeline_id", "input"),
		OutputsEventsDuration: descHelper.NewDescWithHelpAndLabels("outputs_events_duration", "Time spent processing output events", "pipeline_id", "input"),
	}
}

func (collector *PipelineSubcollector) Collect(pipeStats *responses.SinglePipelineResponse, pipelineID string, ch chan<- prometheus.Metric) {
	collectingStart := time.Now()
	log.Printf("collecting pipeline stats for pipeline %s", pipelineID)
	// TODO: The Event durations may be better as histogram observations

	ch <- prometheus.MustNewConstMetric(collector.EventsOut, prometheus.CounterValue, float64(pipeStats.Events.Out), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsFiltered, prometheus.CounterValue, float64(pipeStats.Events.Filtered), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsIn, prometheus.CounterValue, float64(pipeStats.Events.In), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsDuration, prometheus.CounterValue, float64(pipeStats.Events.DurationInMillis), pipelineID)
	ch <- prometheus.MustNewConstMetric(collector.EventsQueuePushDuration, prometheus.GaugeValue, float64(pipeStats.Events.QueuePushDurationInMillis), pipelineID)

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

	// Pipeline plugins metrics
	for _, input := range pipeStats.Plugins.Inputs {
		log.Printf("collecting pipeline plugin stats for pipeline %s :: input %s", pipelineID, input.Name)
		ch <- prometheus.MustNewConstMetric(collector.InputsEventsOut, prometheus.CounterValue, float64(input.Events.Out), pipelineID, input.Name)
		ch <- prometheus.MustNewConstMetric(collector.InputsEventsQueuePushDuration, prometheus.GaugeValue, float64(input.Events.QueuePushDurationInMillis), pipelineID, input.Name)
	}

	for _, filter := range pipeStats.Plugins.Filters {
		log.Printf("collecting pipeline plugin stats for pipeline %s :: filter %s", pipelineID, filter.Name)
		ch <- prometheus.MustNewConstMetric(collector.FiltersEventsIn, prometheus.CounterValue, float64(filter.Events.In), pipelineID, filter.Name)
		ch <- prometheus.MustNewConstMetric(collector.FiltersEventsOut, prometheus.CounterValue, float64(filter.Events.Out), pipelineID, filter.Name)
		ch <- prometheus.MustNewConstMetric(collector.FiltersEventsDuration, prometheus.GaugeValue, float64(filter.Events.DurationInMillis), pipelineID, filter.Name)
	}

	for _, output := range pipeStats.Plugins.Outputs {
		log.Printf("collecting pipeline plugin stats for pipeline %s :: output %s", pipelineID, output.Name)
		ch <- prometheus.MustNewConstMetric(collector.OutputsEventsIn, prometheus.CounterValue, float64(output.Events.In), pipelineID, output.Name)
		ch <- prometheus.MustNewConstMetric(collector.OutputsEventsOut, prometheus.CounterValue, float64(output.Events.Out), pipelineID, output.Name)
		ch <- prometheus.MustNewConstMetric(collector.OutputsEventsDuration, prometheus.GaugeValue, float64(output.Events.DurationInMillis), pipelineID, output.Name)
	}

	collectingEnd := time.Now()
	log.Printf("collected pipeline stats for pipeline %s in %s", pipelineID, collectingEnd.Sub(collectingStart))
}

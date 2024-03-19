package prometheus_helper

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// SimpleDescHelper is a helper struct that can be used to create prometheus.Desc objects
type SimpleDescHelper struct {
	Namespace string
	Subsystem string
}

// NewDescWithLabel creates a new prometheus.Desc with the namespace and subsystem.
// Labels are used to differentiate between different sources of the same metric.
// Labels are always appended with "hostname" to differentiate between different instances.
func (h *SimpleDescHelper) NewDesc(name string, help string, labels ...string) *prometheus.Desc {
	labels = append(labels, "hostname")
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, labels, nil)
}

// ExtractFqName extracts the fqName from a prometheus.Desc string.
// This is useful for testing collectors.
func ExtractFqName(metric string) (string, error) {
	regex := regexp.MustCompile(`fqName:\s*"([a-zA-Z0-9_-]+)"`)
	matches := regex.FindStringSubmatch(metric)
	if len(matches) < 2 {
		return "", errors.New("failed to extract fqName from metric string")
	}
	return matches[1], nil
}

// ExtractValueFromMetric extracts the value from a prometheus.Metric object.
// It creates a custom collector and registry, registers the given metric, and then collects
// the metric value using the registry.
// Returns the extracted float64 value from the metric's Gauge.
func ExtractValueFromMetric(metric prometheus.Metric) (float64, error) {
	var dtoMetric dto.Metric
	err := metric.Write(&dtoMetric)
	if err != nil {
		return 0, fmt.Errorf("error writing metric: %v", err)
	}

	gauge := dtoMetric.GetGauge()
	if gauge == nil {
		return 0, errors.New("the metric is not a Gauge")
	}

	return gauge.GetValue(), nil
}

// SimpleMetricsHelper is a helper struct that can be used to channel new prometheus.Metric objects
type SimpleMetricsHelper struct {
	Channel chan<- prometheus.Metric
	Labels  []string
}

// NewFloatMetric appends new metric with the desc and metricType, value
// optional Labels could be specified through property setter
func (mh *SimpleMetricsHelper) NewFloatMetric(desc *prometheus.Desc, metricType prometheus.ValueType, value float64) {
	metric := prometheus.MustNewConstMetric(desc, metricType, value, mh.Labels...)
	mh.Channel <- metric
}

// NewIntMetric same as NewFloatMetric but for 'int' type
func (mh *SimpleMetricsHelper) NewIntMetric(desc *prometheus.Desc, metricType prometheus.ValueType, value int) {
	mh.NewFloatMetric(desc, metricType, float64(value))
}

// NewInt64Metric same as NewFloatMetric but for 'int64' type
func (mh *SimpleMetricsHelper) NewInt64Metric(desc *prometheus.Desc, metricType prometheus.ValueType, value int64) {
	mh.NewFloatMetric(desc, metricType, float64(value))
}

// newTimestampMetric same as NewFloatMetric but for setting Timestamp value
func (mh *SimpleMetricsHelper) NewTimestampMetric(desc *prometheus.Desc, metricType prometheus.ValueType, value time.Time) {
	metric := prometheus.NewMetricWithTimestamp(value, prometheus.MustNewConstMetric(desc, metricType, 1, mh.Labels...))
	mh.Channel <- metric
}

// ExtractValueFromMetric extracts the timestamp from a prometheus.Metric object.
// Useful for testing NewTimestampMetric method of SimpleMetricsHelper
// Returns the extracted timestamp in milliseconds of 'int64' type
func extractTimestampMsFromMetric(metric prometheus.Metric) (int64, error) {
	var dtoMetric dto.Metric
	err := metric.Write(&dtoMetric)
	if err != nil {
		return 0, fmt.Errorf("error writing metric: %v", err)
	}

	return dtoMetric.GetTimestampMs(), nil
}

package prometheus_helper

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// SimpleDescHelper is a helper struct that can be used to create prometheus.Desc objects
type SimpleDescHelper struct {
	Namespace string
	Subsystem string
}

// NewDescWithHelp creates a new prometheus.Desc with the namespace and subsystem.
func (h *SimpleDescHelper) NewDescWithHelp(name string, help string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, nil, nil)
}

// NewDescWithLabel creates a new prometheus.Desc with the namespace and subsystem.
// Labels are used to differentiate between different sources of the same metric.
func (h *SimpleDescHelper) NewDescWithHelpAndLabels(name string, help string, labels ...string) *prometheus.Desc {
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

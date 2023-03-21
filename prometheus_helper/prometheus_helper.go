package prometheus_helper

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type SimpleDescHelper struct {
	Namespace string
	Subsystem string
}

// NewDesc creates a new prometheus.Desc with the namespace and subsystem. The help text is set to the name.
func (h *SimpleDescHelper) NewDesc(name string) *prometheus.Desc {
	help := name
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, nil, nil)
}

// NewDescWithHelp creates a new prometheus.Desc with the namespace and subsystem.
func (h *SimpleDescHelper) NewDescWithHelp(name string, help string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, nil, nil)
}

// NewDescWithLabel creates a new prometheus.Desc with the namespace and subsystem.
// Labels are used to differentiate between different sources of the same metric.
func (h *SimpleDescHelper) NewDescWithHelpAndLabel(name, help, label string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, []string{label}, nil)
}

// ExtractFqName extracts the fqName from a prometheus.Desc string.
// This is useful for testing collectors.
func ExtractFqName(metric string) (string, error) {
	regex := regexp.MustCompile(`fqName:\s*"([a-zA-Z_-]+)"`)
	matches := regex.FindStringSubmatch(metric)
	if len(matches) < 2 {
		return "", errors.New("failed to extract fqName from metric string")
	}
	return matches[1], nil
}

// CustomCollector is a custom prometheus.Collector that collects only the given metric.
type CustomCollector struct {
	metric prometheus.Metric
}

// Describe implements the prometheus.Collector interface.
func (c *CustomCollector) Describe(ch chan<- *prometheus.Desc) {
	c.metric.Desc()
}

// Collect implements the prometheus.Collector interface.
func (c *CustomCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.metric
}

// ExtractValueFromMetric extracts the value from a prometheus.Metric object.
// It creates a custom collector and registry, registers the given metric, and then collects
// the metric value using the registry.
// Returns the extracted float64 value from the metric's Gauge.
func ExtractValueFromMetric(metric prometheus.Metric) (float64, error) {
	// Custom collector that collects only the given metric.
	collector := &CustomCollector{
		metric: metric,
	}

	// Create a custom registry and register the collector.
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	if err != nil {
		return 0, err
	}

	var metricValue float64
	metricChannel := make(chan prometheus.Metric)
	go func() {
		registry.Collect(metricChannel)
		close(metricChannel)
	}()

	for collectedMetric := range metricChannel {
		if collectedMetric.Desc().String() == metric.Desc().String() {
			var dtoMetric dto.Metric
			err = collectedMetric.Write(&dtoMetric)
			if err != nil {
				return 0, fmt.Errorf("error writing metric: %v", err)
			}
			metricValue = dtoMetric.GetGauge().GetValue()
			break
		}
	}

	return metricValue, nil
}

package e2e

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Metrics is a parsed Prometheus text exposition
type Metrics struct {
	families map[string]*dto.MetricFamily
}

// ParseMetrics parses a Prometheus text exposition
func ParseMetrics(text string) (*Metrics, error) {
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}
	return &Metrics{families: families}, nil
}

// Has reports whether a metric family with the given name exists
func (m *Metrics) Has(name string) bool {
	family, ok := m.families[name]
	return ok && len(family.GetMetric()) > 0
}

// Names returns the sorted names of all metric families
func (m *Metrics) Names() []string {
	names := make([]string, 0, len(m.families))
	for name := range m.families {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Family returns the metric family with the given name, or nil
func (m *Metrics) Family(name string) *dto.MetricFamily {
	return m.families[name]
}

// Value returns the value of the first sample of the named metric whose
// labels include all given labels
func (m *Metrics) Value(name string, labels map[string]string) (float64, error) {
	family, ok := m.families[name]
	if !ok {
		return 0, fmt.Errorf("metric %s not found", name)
	}

	for _, metric := range family.GetMetric() {
		if !hasLabels(metric, labels) {
			continue
		}
		switch {
		case metric.GetGauge() != nil:
			return metric.GetGauge().GetValue(), nil
		case metric.GetCounter() != nil:
			return metric.GetCounter().GetValue(), nil
		case metric.GetUntyped() != nil:
			return metric.GetUntyped().GetValue(), nil
		case metric.GetSummary() != nil:
			return float64(metric.GetSummary().GetSampleCount()), nil
		case metric.GetHistogram() != nil:
			return float64(metric.GetHistogram().GetSampleCount()), nil
		}
	}

	return 0, fmt.Errorf("metric %s with labels %v not found", name, labels)
}

// LabelValues returns the sorted distinct values of a label across all
// metric families
func (m *Metrics) LabelValues(label string) []string {
	seen := map[string]bool{}
	for _, family := range m.families {
		for _, metric := range family.GetMetric() {
			for _, pair := range metric.GetLabel() {
				if pair.GetName() == label {
					seen[pair.GetValue()] = true
				}
			}
		}
	}

	values := make([]string, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

// Instances returns the Logstash instances the metrics were collected from
func (m *Metrics) Instances() []string {
	return m.LabelValues("instance_name")
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// InstanceName returns the instance_name label the exporter uses for an
// unnamed instance with the given URL, e.g. http_127_0_0_1_9600
func InstanceName(url string) string {
	return strings.Trim(nonAlphanumeric.ReplaceAllString(url, "_"), "_")
}

func hasLabels(metric *dto.Metric, labels map[string]string) bool {
	for name, value := range labels {
		found := false
		for _, pair := range metric.GetLabel() {
			if pair.GetName() == name && pair.GetValue() == value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

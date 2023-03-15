package prometheus_helper

import (
	"errors"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
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

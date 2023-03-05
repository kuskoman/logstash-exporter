package helpers

import (
	"errors"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
)

type SimpleDescHelper struct {
	Namespace string
	Subsystem string
}

func (h *SimpleDescHelper) NewDesc(name string) *prometheus.Desc {
	help := name
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, nil, nil)
}

func (h *SimpleDescHelper) NewDescWithHelp(name string, help string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, nil, nil)
}

func (h *SimpleDescHelper) NewDescWithLabels(name string, labels []string) *prometheus.Desc {
	help := name
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, labels, nil)
}

func ExtractFqName(metric string) (string, error) {
	regex := regexp.MustCompile(`fqName:\s*"([a-zA-Z_-]+)"`)
	matches := regex.FindStringSubmatch(metric)
	if len(matches) < 2 {
		return "", errors.New("failed to extract fqName from metric string")
	}
	return matches[1], nil
}

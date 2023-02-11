package helpers

import "github.com/prometheus/client_golang/prometheus"

type SimpleDescHelper struct {
	Namespace string
	Subsystem string
}

func (h *SimpleDescHelper) NewDesc(name string) *prometheus.Desc {
	help := name
	return prometheus.NewDesc(prometheus.BuildFQName(h.Namespace, h.Subsystem, name), help, nil, nil)
}

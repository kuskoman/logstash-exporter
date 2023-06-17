package exporter

import (
	"context"

	"github.com/kuskoman/logstash-exporter/config"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystem = "exporter_self"
	namespace = config.PrometheusNamespace
)

// ExporterInfoCollector is a collector for exposing the exporter's own information
type ExporterInfoCollector struct {
	BuildInfo *prometheus.Desc
	Up        *prometheus.Desc
}

// NewExporterInfoCollector creates a new SelfinfoCollector
func NewExporterInfoCollector() *ExporterInfoCollector {
	descHelper := prometheus_helper.SimpleDescHelper{Namespace: namespace, Subsystem: subsystem}
	return &ExporterInfoCollector{
		BuildInfo: descHelper.NewDescWithHelpAndLabels("build_info", "A metric with a constant '1' value labeled by version, git commit, go version, build arch, build os, and build date of the exporter.", "version", "git_commit", "go_version", "build_arch", "build_os", "build_date"),
		Up:        descHelper.NewDescWithHelp("up", "A metric that returns 1 if the exporter is up, 0 otherwise."),
	}
}

func (c *ExporterInfoCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	versionInfo := config.GetVersionInfo()
	ch <- prometheus.MustNewConstMetric(c.BuildInfo, prometheus.GaugeValue, 1, versionInfo.Version, versionInfo.GitCommit, versionInfo.GoVersion, versionInfo.BuildArch, versionInfo.BuildOS, versionInfo.BuildDate)
	ch <- prometheus.MustNewConstMetric(c.Up, prometheus.GaugeValue, 1)

	return nil
}

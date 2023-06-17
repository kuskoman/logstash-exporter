package exporter

import (
	"context"
	"fmt"
	"testing"

	"github.com/kuskoman/logstash-exporter/config"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
	"github.com/prometheus/client_golang/prometheus"
)

func TestCollect(t *testing.T) {
	t.Parallel()
	collector := NewExporterInfoCollector()
	ch := make(chan prometheus.Metric)
	ctx := context.Background()

	go func() {
		err := collector.Collect(ctx, ch)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		close(ch)
	}()

	expectedMetrics := []string{
		"logstash_exporter_self_build_info",
		"logstash_exporter_self_up",
	}

	var foundMetrics []string
	for metric := range ch {
		if metric == nil {
			t.Errorf("expected metric %s not to be nil", metric.Desc().String())
		}

		foundMetricDesc := metric.Desc().String()
		foundMetricFqName, err := prometheus_helper.ExtractFqName(foundMetricDesc)
		if err != nil {
			t.Errorf("failed to extract fqName from metric %s", foundMetricDesc)
		}

		foundMetrics = append(foundMetrics, foundMetricFqName)
	}

	for _, expectedMetric := range expectedMetrics {
		found := false
		for _, foundMetric := range foundMetrics {
			if foundMetric == expectedMetric {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected metric %s to be found", expectedMetric)
		}
	}
}

func TestBuildInfoMetric(t *testing.T) {
	t.Parallel()
	collector := NewExporterInfoCollector()
	ch := make(chan prometheus.Metric)
	ctx := context.Background()

	go func() {
		err := collector.Collect(ctx, ch)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		close(ch)
	}()

	buildInfoRequestName := fmt.Sprintf("%s_%s_build_info", namespace, subsystem)
	buildInfoMetricFound := false

	for metric := range ch {
		foundMetricDesc := metric.Desc().String()
		foundMetricFqName, err := prometheus_helper.ExtractFqName(foundMetricDesc)
		if err != nil {
			t.Errorf("failed to extract fqName from metric %s", foundMetricDesc)
		}

		if foundMetricFqName == buildInfoRequestName {
			buildInfoMetricFound = true
			labels, err := prometheus_helper.ExtractLabelsFromMetric(metric)
			if err != nil {
				t.Errorf("Failed to extract labels from metric: %v", err)
			}

			versionInfo := config.GetVersionInfo()
			if labels["version"] != versionInfo.Version ||
				labels["git_commit"] != versionInfo.GitCommit ||
				labels["go_version"] != versionInfo.GoVersion ||
				labels["build_arch"] != versionInfo.BuildArch ||
				labels["build_os"] != versionInfo.BuildOS ||
				labels["build_date"] != versionInfo.BuildDate {
				t.Errorf("Build info metric has incorrect labels")
			}
		}
	}

	if !buildInfoMetricFound {
		t.Errorf("Expected metric %s to be found", buildInfoRequestName)
	}
}

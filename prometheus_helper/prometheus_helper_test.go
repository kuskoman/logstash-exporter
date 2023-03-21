package prometheus_helper

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestSimpleDescHelper(t *testing.T) {
	helper := &SimpleDescHelper{
		Namespace: "logstash_exporter",
		Subsystem: "test",
	}

	t.Run("NewDesc", func(t *testing.T) {
		desc := helper.NewDesc("metric")
		expectedDesc := "Desc{fqName: \"logstash_exporter_test_metric\", help: \"metric\", constLabels: {}, variableLabels: []}"
		if desc.String() != expectedDesc {
			t.Errorf("incorrect metric description, expected %s but got %s", expectedDesc, desc.String())
		}
	})

	t.Run("NewDescWithHelp", func(t *testing.T) {
		desc := helper.NewDescWithHelp("metric", "help")
		expectedDesc := "Desc{fqName: \"logstash_exporter_test_metric\", help: \"help\", constLabels: {}, variableLabels: []}"
		if desc.String() != expectedDesc {
			t.Errorf("incorrect metric description, expected %s but got %s", expectedDesc, desc.String())
		}
	})
}

func TestExtractFqdnName(t *testing.T) {
	t.Run("should properly extract fqdn from valid metric description", func(t *testing.T) {
		helper := &SimpleDescHelper{
			Namespace: "logstash_exporter",
			Subsystem: "test",
		}

		metricSubname := "fqdn_metric"

		descriptors := []*prometheus.Desc{
			helper.NewDesc(metricSubname),
			helper.NewDescWithHelp(metricSubname, "help"),
			helper.NewDescWithHelpAndLabel(metricSubname, "help", "label"),
		}

		for _, desc := range descriptors {
			fqdn, err := ExtractFqName(desc.String())
			if err != nil {
				t.Errorf("failed to extract fqName from metric %s", desc)
			}

			if fqdn != fmt.Sprintf("logstash_exporter_test_%s", metricSubname) {
				t.Errorf("incorrect fqdn, expected %s but got %s", "logstash_exporter_test_"+metricSubname, fqdn)
			}
		}
	})

	t.Run("should return error if metric description is invalid", func(t *testing.T) {
		_, err := ExtractFqName("invalid metric description")
		if err == nil {
			t.Errorf("expected error but got nil")
		}
	})
}

func TestExtractValueFromMetric(t *testing.T) {
	t.Run("should extract value from a metric", func(t *testing.T) {
		metricDesc := prometheus.NewDesc("test_metric", "test metric help", nil, nil)
		metricValue := 42.0
		metric := prometheus.MustNewConstMetric(metricDesc, prometheus.GaugeValue, metricValue)

		extractedValue, err := ExtractValueFromMetric(metric)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if extractedValue != metricValue {
			t.Errorf("Expected extracted value to be %f, got %f", metricValue, extractedValue)
		}
	})

	t.Run("should return an error when unable to write metric", func(t *testing.T) {
		metricDesc := prometheus.NewDesc("test_metric", "test metric help", nil, nil)
		exampleErr := fmt.Errorf("example error")
		invalidMetric := prometheus.NewInvalidMetric(metricDesc, exampleErr)

		customCollector := &CustomCollector{
			metric: invalidMetric,
		}

		registry := prometheus.NewRegistry()
		err := registry.Register(customCollector)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		extractedValue, err := ExtractValueFromMetric(invalidMetric)

		if err == nil {
			t.Errorf("Expected error but got nil")
		}

		if extractedValue != 0 {
			t.Errorf("Expected extracted value to be 0, got %f", extractedValue)
		}
	})
}

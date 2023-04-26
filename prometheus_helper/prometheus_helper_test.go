package prometheus_helper

import (
	"errors"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestSimpleDescHelper(t *testing.T) {
	helper := &SimpleDescHelper{
		Namespace: "logstash_exporter",
		Subsystem: "test",
	}

	t.Run("NewDescWithHelp", func(t *testing.T) {
		desc := helper.NewDescWithHelp("metric", "help")
		expectedDesc := "Desc{fqName: \"logstash_exporter_test_metric\", help: \"help\", constLabels: {}, variableLabels: []}"
		if desc.String() != expectedDesc {
			t.Errorf("incorrect metric description, expected %s but got %s", expectedDesc, desc.String())
		}
	})

	t.Run("NewDescWithHelpAndLabel", func(t *testing.T) {
		desc := helper.NewDescWithHelpAndLabels("metric", "help", "customLabel")
		expectedDesc := "Desc{fqName: \"logstash_exporter_test_metric\", help: \"help\", constLabels: {}, variableLabels: [{customLabel <nil>}]}"
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
			helper.NewDescWithHelp(metricSubname, "help"),
			helper.NewDescWithHelpAndLabels(metricSubname, "help", "label"),
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

type badMetricStub struct{}

func (m *badMetricStub) Desc() *prometheus.Desc {
	return nil
}

func (m *badMetricStub) Write(*dto.Metric) error {
	return errors.New("writing metric failed")
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

	t.Run("should return error if writing metric fails", func(t *testing.T) {
		badMetric := &badMetricStub{}
		val, err := ExtractValueFromMetric(badMetric)

		if err == nil {
			t.Errorf("Expected error, but got nil")
		}

		if val != 0 {
			t.Errorf("Expected value to be 0, got %f", val)
		}
	})

	t.Run("should return error if the metric is not a Gauge", func(t *testing.T) {
		metricDesc := prometheus.NewDesc("test_counter_metric", "test counter metric help", nil, nil)
		metricValue := 42.0
		metric := prometheus.MustNewConstMetric(metricDesc, prometheus.CounterValue, metricValue)

		val, err := ExtractValueFromMetric(metric)
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}

		if val != 0 {
			t.Errorf("Expected value to be 0, got %f", val)
		}
	})
}

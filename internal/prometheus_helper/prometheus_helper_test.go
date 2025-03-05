package prometheus_helper

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestSimpleDescHelper(t *testing.T) {
	helper := &SimpleDescHelper{
		Namespace: "logstash_exporter",
		Subsystem: "test",
	}

	t.Run("NewDescWithHelpAndLabel", func(t *testing.T) {
		desc := helper.NewDesc("metric", "help", "customLabel")
		expectedDesc := "Desc{fqName: \"logstash_exporter_test_metric\", help: \"help\", constLabels: {}, variableLabels: {customLabel,hostname,name}}"
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
			helper.NewDesc(metricSubname, "help"),
			helper.NewDesc(metricSubname, "help", "label"),
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
			t.Errorf("unexpected error: %v", err)
		}

		if extractedValue != metricValue {
			t.Errorf("expected extracted value to be %f, got %f", metricValue, extractedValue)
		}
	})

	t.Run("should return error if writing metric fails", func(t *testing.T) {
		badMetric := &badMetricStub{}
		val, err := ExtractValueFromMetric(badMetric)

		if err == nil {
			t.Errorf("expected error, but got nil")
		}

		if val != 0 {
			t.Errorf("expected value to be 0, got %f", val)
		}
	})

	t.Run("should return error if the metric is not a Gauge", func(t *testing.T) {
		metricDesc := prometheus.NewDesc("test_counter_metric", "test counter metric help", nil, nil)
		metricValue := 42.0
		metric := prometheus.MustNewConstMetric(metricDesc, prometheus.CounterValue, metricValue)

		val, err := ExtractValueFromMetric(metric)
		if err == nil {
			t.Errorf("expected error, but got nil")
		}

		if val != 0 {
			t.Errorf("expected value to be 0, got %f", val)
		}
	})
}

func TestSimpleMetricsHelper(t *testing.T) {
	t.Run("should create a new metric", func(t *testing.T) {
		metricName := "test_metric"
		metricDesc := prometheus.NewDesc(metricName, "test metric help", nil, nil)
		metricValue := 42.0

		ch := make(chan prometheus.Metric)

		go func() {
			helper := &SimpleMetricsHelper{
				Channel: ch,
				Labels:  []string{},
			}
			helper.NewFloatMetric(metricDesc, prometheus.GaugeValue, metricValue)
		}()

		metric := <-ch

		fqName, err := ExtractFqName(metric.Desc().String())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if metricName != fqName {
			t.Errorf("expected extracted name to be %s, got %s", metricName, fqName)
		}

		val, err := ExtractValueFromMetric(metric)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != metricValue {
			t.Errorf("expected extracted value to be %f, got %f", metricValue, val)
		}
	})

	t.Run("should create a new metric with labels", func(t *testing.T) {
		helper := &SimpleDescHelper{
			Namespace: "logstash_exporter",
			Subsystem: "test",
		}

		metricDesc := helper.NewDesc("metric", "help", "customLabel")
		metricValue := 42.0

		ch := make(chan prometheus.Metric)

		go func() {
			helper := &SimpleMetricsHelper{
				Channel: ch,
				Labels:  []string{"customLabelValue", "hostnameEndpoint", "instanceName"},
			}
			helper.NewFloatMetric(metricDesc, prometheus.GaugeValue, metricValue)
		}()

		metric := <-ch

		desc := metric.Desc()
		if metricDesc.String() != desc.String() {
			t.Errorf("incorrect metric description, expected %s but got %s", metricDesc, desc.String())
		}

		val, err := ExtractValueFromMetric(metric)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != metricValue {
			t.Errorf("expected extracted value to be %f, got %f", metricValue, val)
		}
	})

	t.Run("should create metrics with different value types", func(t *testing.T) {
		metricName := "test_metric"
		metricDesc := prometheus.NewDesc(metricName, "test metric help", nil, nil)
		metricValue := 42.0

		ch := make(chan prometheus.Metric, 3)

		helper := &SimpleMetricsHelper{
			Channel: ch,
			Labels:  []string{},
		}
		helper.NewFloatMetric(metricDesc, prometheus.GaugeValue, metricValue)
		helper.NewIntMetric(metricDesc, prometheus.GaugeValue, int(metricValue))
		helper.NewInt64Metric(metricDesc, prometheus.GaugeValue, int64(metricValue))

		close(ch)

		for metric := range ch {
			val, err := ExtractValueFromMetric(metric)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if val != metricValue {
				t.Errorf("expected extracted value to be %f, got %f", metricValue, val)
			}
		}
	})

	t.Run("should create timestamp metric", func(t *testing.T) {
		metricName := "test_metric"
		metricDesc := prometheus.NewDesc(metricName, "test metric help", nil, nil)
		metricValue := time.UnixMilli(42)

		ch := make(chan prometheus.Metric)

		go func() {
			helper := &SimpleMetricsHelper{
				Channel: ch,
				Labels:  []string{},
			}
			helper.NewTimestampMetric(metricDesc, prometheus.CounterValue, metricValue)
		}()

		metric := <-ch

		fqName, err := ExtractFqName(metric.Desc().String())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if metricName != fqName {
			t.Errorf("expected extracted name to be %s, got %s", metricName, fqName)
		}

		val, err := extractTimestampMsFromMetric(metric)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != metricValue.UnixMilli() {
			t.Errorf("expected extracted value to be %d, got %d", metricValue.UnixMilli(), val)
		}
	})
}

func TestExtractTimestampMsFromMetric(t *testing.T) {
	t.Run("should extract timestamp from a metric", func(t *testing.T) {
		metricDesc := prometheus.NewDesc("test_metric", "test metric help", nil, nil)
		metricType := prometheus.GaugeValue
		metricValue := time.UnixMilli(42)
		metric := prometheus.NewMetricWithTimestamp(metricValue, prometheus.MustNewConstMetric(metricDesc, metricType, 1))

		extractedValue, err := extractTimestampMsFromMetric(metric)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if extractedValue != metricValue.UnixMilli() {
			t.Errorf("expected extracted value to be %d, got %d", metricValue.UnixMilli(), extractedValue)
		}
	})

	t.Run("should return error if writing metric fails", func(t *testing.T) {
		badMetric := &badMetricStub{}
		val, err := extractTimestampMsFromMetric(badMetric)

		if err == nil {
			t.Errorf("expected error, but got nil")
		}

		if val != 0 {
			t.Errorf("expected value to be 0, got %d", val)
		}
	})

}

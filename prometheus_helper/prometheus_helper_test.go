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
}

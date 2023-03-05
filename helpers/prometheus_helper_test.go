package helpers

import (
	"fmt"
	"testing"
)

func TestSimpleDescHelper(t *testing.T) {
	helper := &SimpleDescHelper{
		Namespace: "logstash_exporter",
		Subsystem: "test",
	}

	desc := helper.NewDesc("metric")
	expectedDesc := "Desc{fqName: \"logstash_exporter_test_metric\", help: \"metric\", constLabels: {}, variableLabels: []}"
	if desc.String() != expectedDesc {
		t.Errorf("incorrect metric description, expected %s but got %s", expectedDesc, desc.String())
	}
}

func TestExtractFqdnName(t *testing.T) {
	helper := &SimpleDescHelper{
		Namespace: "logstash_exporter",
		Subsystem: "test",
	}

	metricSubname := "fqdn_metric"
	desc := helper.NewDesc(metricSubname).String()
	fqdn, err := ExtractFqName(desc)
	if err != nil {
		t.Errorf("failed to extract fqName from metric %s", desc)
	}

	if fqdn != fmt.Sprintf("logstash_exporter_test_%s", metricSubname) {
		t.Errorf("incorrect fqdn, expected %s but got %s", "logstash_exporter_test_"+metricSubname, fqdn)
	}
}

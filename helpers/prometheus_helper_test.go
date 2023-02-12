package helpers

import (
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

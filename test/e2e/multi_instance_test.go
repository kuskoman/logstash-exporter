package e2e

import (
	"testing"
)

func TestMultiInstance(t *testing.T) {
	t.Parallel()

	mockLogstash1 := NewMockLogstashServer(t)
	mockLogstash2 := NewMockLogstashServer(t)

	exporter := StartExporter(t, NewConfigBuilder().
		AddInstance(mockLogstash1.URL()).
		AddInstance(mockLogstash2.URL()))

	metrics := exporter.WaitForInstances(2)

	t.Run("exports_metrics_for_each_instance", func(t *testing.T) {
		for _, mock := range []*MockLogstashServer{mockLogstash1, mockLogstash2} {
			if _, err := metrics.Value("logstash_info_build", map[string]string{"instance_name": InstanceName(mock.URL())}); err != nil {
				t.Errorf("instance %s: %v", mock.URL(), err)
			}
		}
	})

	t.Run("queries_every_instance", func(t *testing.T) {
		for _, mock := range []*MockLogstashServer{mockLogstash1, mockLogstash2} {
			if mock.RequestCount(nodeStatsPath) == 0 {
				t.Errorf("instance %s received no requests", mock.URL())
			}
		}
	})
}

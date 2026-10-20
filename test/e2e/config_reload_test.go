package e2e

import (
	"testing"
)

func TestConfigReload(t *testing.T) {
	t.Parallel()

	t.Run("adding_an_instance_starts_scraping_it", func(t *testing.T) {
		t.Parallel()

		mockLogstash1 := NewMockLogstashServer(t)
		mockLogstash2 := NewMockLogstashServer(t)

		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash1.URL()), WithWatch())
		exporter.WaitForInstances(1)

		exporter.WriteConfig(NewConfigBuilder().
			AddInstance(mockLogstash1.URL()).
			AddInstance(mockLogstash2.URL()))

		exporter.WaitForInstances(2)
		if mockLogstash2.RequestCount(nodeStatsPath) == 0 {
			t.Error("expected the new instance to be queried after reload")
		}
	})
}

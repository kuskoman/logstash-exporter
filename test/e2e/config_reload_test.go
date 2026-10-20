package e2e

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// reloadSettleTime is how long tests wait to make sure a change that should
// not trigger a reload really did not. The file watcher debounces events for
// 100ms, so this leaves plenty of margin.
const reloadSettleTime = 1 * time.Second

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

	t.Run("removing_an_instance_stops_scraping_it", func(t *testing.T) {
		t.Parallel()

		mockLogstash1 := NewMockLogstashServer(t)
		mockLogstash2 := NewMockLogstashServer(t)

		exporter := StartExporter(t, NewConfigBuilder().
			AddInstance(mockLogstash1.URL()).
			AddInstance(mockLogstash2.URL()), WithWatch())
		exporter.WaitForInstances(2)

		exporter.WriteConfig(NewConfigBuilder().AddInstance(mockLogstash1.URL()))
		exporter.WaitForInstances(1)

		mockLogstash2.ResetRequests()
		if _, err := exporter.Metrics(); err != nil {
			t.Fatal(err)
		}
		if count := mockLogstash2.RequestCount(""); count != 0 {
			t.Errorf("expected the removed instance not to be queried, got %d requests", count)
		}
	})

	t.Run("changing_the_port_moves_the_server", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()), WithWatch())
		exporter.WaitForInstances(1)

		oldPort := exporter.Port
		newPort, err := GetFreePort()
		if err != nil {
			t.Fatal(err)
		}

		exporter.WriteConfig(NewConfigBuilder().AddInstance(mockLogstash.URL()).WithServerPort(newPort))
		exporter.Port = newPort
		exporter.WaitForInstances(1)

		exporter.Port = oldPort
		if _, err := exporter.Get("/version"); err == nil {
			t.Errorf("expected the old port %d to be closed after reload", oldPort)
		}
	})

	t.Run("invalid_config_keeps_the_previous_configuration", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()), WithWatch())
		exporter.WaitForInstances(1)

		writeFile(t, exporter.ConfigPath, "logstash: [this is not valid")

		assertKeepsServing(t, exporter, InstanceName(mockLogstash.URL()))
	})

	t.Run("empty_config_keeps_the_previous_configuration", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()), WithWatch())
		exporter.WaitForInstances(1)

		writeFile(t, exporter.ConfigPath, "")

		assertKeepsServing(t, exporter, InstanceName(mockLogstash.URL()))
	})

	t.Run("valid_config_after_an_invalid_one_is_applied", func(t *testing.T) {
		t.Parallel()

		mockLogstash1 := NewMockLogstashServer(t)
		mockLogstash2 := NewMockLogstashServer(t)

		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash1.URL()), WithWatch())
		exporter.WaitForInstances(1)

		writeFile(t, exporter.ConfigPath, "logstash: [this is not valid")
		time.Sleep(reloadSettleTime)

		exporter.WriteConfig(NewConfigBuilder().
			AddInstance(mockLogstash1.URL()).
			AddInstance(mockLogstash2.URL()))
		exporter.WaitForInstances(2)
	})

	t.Run("unchanged_content_does_not_restart_the_server", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()), WithWatch())
		exporter.WaitForInstances(1)

		content, err := os.ReadFile(exporter.ConfigPath)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, exporter.ConfigPath, string(content))
		time.Sleep(reloadSettleTime)

		if strings.Contains(exporter.Logs(), "config has changed, reloading server") {
			t.Error("expected no reload when the content did not change")
		}
	})

	t.Run("atomic_save_by_rename_is_detected", func(t *testing.T) {
		t.Parallel()

		mockLogstash1 := NewMockLogstashServer(t)
		mockLogstash2 := NewMockLogstashServer(t)

		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash1.URL()), WithWatch())
		exporter.WaitForInstances(1)

		// Editors like vim and most config management tools write a
		// temporary file and rename it over the original.
		cb := NewConfigBuilder().
			AddInstance(mockLogstash1.URL()).
			AddInstance(mockLogstash2.URL()).
			WithServerPort(exporter.Port)
		tempPath := filepath.Join(filepath.Dir(exporter.ConfigPath), ".config.yml.tmp")
		writeFile(t, tempPath, cb.Build())
		if err := os.Rename(tempPath, exporter.ConfigPath); err != nil {
			t.Fatal(err)
		}

		exporter.WaitForInstances(2)
	})

	t.Run("kubernetes_configmap_update_is_detected", func(t *testing.T) {
		t.Parallel()

		mockLogstash1 := NewMockLogstashServer(t)
		mockLogstash2 := NewMockLogstashServer(t)

		// Kubernetes mounts a ConfigMap as a directory where config.yml is a
		// symlink to ..data/config.yml, and ..data is a symlink to a
		// timestamped directory. An update writes a new timestamped
		// directory and atomically swaps the ..data symlink.
		configMapDir := t.TempDir()
		writeConfigMapVersion(t, configMapDir, "..2026_01_01_00_00_00.000000001",
			NewConfigBuilder().AddInstance(mockLogstash1.URL()))
		if err := os.Symlink(filepath.Join("..data", "config.yml"), filepath.Join(configMapDir, "config.yml")); err != nil {
			t.Fatal(err)
		}

		exporter := StartExporterWithConfigFile(t, filepath.Join(configMapDir, "config.yml"),
			func(port int) {
				writeConfigMapVersion(t, configMapDir, "..2026_01_01_00_00_00.000000001",
					NewConfigBuilder().AddInstance(mockLogstash1.URL()).WithServerPort(port))
			}, WithWatch())
		exporter.WaitForInstances(1)

		writeConfigMapVersion(t, configMapDir, "..2026_01_01_00_01_00.000000002",
			NewConfigBuilder().
				AddInstance(mockLogstash1.URL()).
				AddInstance(mockLogstash2.URL()).
				WithServerPort(exporter.Port))

		exporter.WaitForInstances(2)
	})

	t.Run("reload_is_ignored_without_the_watch_flag", func(t *testing.T) {
		t.Parallel()

		mockLogstash1 := NewMockLogstashServer(t)
		mockLogstash2 := NewMockLogstashServer(t)

		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash1.URL()))
		exporter.WaitForInstances(1)

		exporter.WriteConfig(NewConfigBuilder().
			AddInstance(mockLogstash1.URL()).
			AddInstance(mockLogstash2.URL()))

		Consistently(t, reloadSettleTime, func() error {
			metrics, err := exporter.Metrics()
			if err != nil {
				return err
			}
			if instances := metrics.Instances(); len(instances) != 1 {
				return fmt.Errorf("expected config not to be reloaded, got instances %v", instances)
			}
			return nil
		})
	})
}

// assertKeepsServing checks that the exporter keeps serving metrics for the
// given instance and stays on its port
func assertKeepsServing(t *testing.T, exporter *Exporter, instance string) {
	t.Helper()

	Consistently(t, reloadSettleTime, func() error {
		resp, err := exporter.Get("/metrics")
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("/metrics returned %d", resp.StatusCode)
		}
		metrics, err := ParseMetrics(resp.Body)
		if err != nil {
			return err
		}
		if instances := metrics.Instances(); len(instances) != 1 || instances[0] != instance {
			return fmt.Errorf("expected instance %s, got %v", instance, instances)
		}
		return nil
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// writeConfigMapVersion writes config.yml into a new timestamped directory and
// atomically points ..data at it, the way the kubelet updates ConfigMaps
func writeConfigMapVersion(t *testing.T, configMapDir, version string, cb *ConfigBuilder) {
	t.Helper()

	versionDir := filepath.Join(configMapDir, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(versionDir, "config.yml"), cb.Build())

	tempLink := filepath.Join(configMapDir, "..data_tmp")
	_ = os.Remove(tempLink)
	if err := os.Symlink(version, tempLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tempLink, filepath.Join(configMapDir, "..data")); err != nil {
		t.Fatal(err)
	}
}

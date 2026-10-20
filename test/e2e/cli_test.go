package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const cliTimeout = 10 * time.Second

func TestCLIFlags(t *testing.T) {
	t.Parallel()

	t.Run("version_prints_the_version_and_exits", func(t *testing.T) {
		t.Parallel()

		result := RunExporter(t, cliTimeout, "-version")
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\n%s", result.ExitCode, result.Output)
		}
		if strings.TrimSpace(result.Output) == "" {
			t.Error("expected the version to be printed")
		}
	})

	t.Run("help_lists_the_flags_and_exits", func(t *testing.T) {
		t.Parallel()

		result := RunExporter(t, cliTimeout, "-help")
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\n%s", result.ExitCode, result.Output)
		}
		for _, flag := range []string{"-config", "-watch", "-version", "-help"} {
			if !strings.Contains(result.Output, flag) {
				t.Errorf("expected help to describe %s, got:\n%s", flag, result.Output)
			}
		}
	})

	t.Run("unknown_flag_fails", func(t *testing.T) {
		t.Parallel()

		result := RunExporter(t, cliTimeout, "-does-not-exist")
		if result.ExitCode == 0 {
			t.Errorf("expected a non-zero exit code for an unknown flag\n%s", result.Output)
		}
		if !strings.Contains(result.Output, "does-not-exist") {
			t.Errorf("expected the error to name the unknown flag, got:\n%s", result.Output)
		}
	})
}

func TestInvalidConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("missing_config_file_fails", func(t *testing.T) {
		t.Parallel()

		missing := filepath.Join(t.TempDir(), "missing.yml")
		result := RunExporter(t, cliTimeout, "-config", missing)
		if result.ExitCode == 0 {
			t.Errorf("expected a non-zero exit code\n%s", result.Output)
		}
		if !strings.Contains(result.Output, "missing.yml") {
			t.Errorf("expected the error to name the missing file, got:\n%s", result.Output)
		}
	})

	t.Run("invalid_yaml_fails", func(t *testing.T) {
		t.Parallel()

		configPath := WriteConfigFile(t, "logstash: [this is not valid")
		result := RunExporter(t, cliTimeout, "-config", configPath)
		if result.ExitCode == 0 {
			t.Errorf("expected a non-zero exit code\n%s", result.Output)
		}
		if !strings.Contains(result.Output, "yaml") {
			t.Errorf("expected a YAML error, got:\n%s", result.Output)
		}
	})

	t.Run("port_already_in_use_fails", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		running := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))

		configPath := WriteConfigFile(t, NewConfigBuilder().
			AddInstance(mockLogstash.URL()).
			WithServerPort(running.Port).
			Build())
		result := RunExporter(t, cliTimeout, "-config", configPath)
		if result.ExitCode == 0 {
			t.Errorf("expected a non-zero exit code\n%s", result.Output)
		}
		if !strings.Contains(result.Output, "address already in use") {
			t.Errorf("expected an address in use error, got:\n%s", result.Output)
		}
	})
}

func TestConfigurationSources(t *testing.T) {
	t.Parallel()

	t.Run("config_location_from_environment", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		port, err := GetFreePort()
		if err != nil {
			t.Fatal(err)
		}
		configPath := WriteConfigFile(t, NewConfigBuilder().
			AddInstance(mockLogstash.URL()).
			WithServerPort(port).
			Build())

		// Start without -config so the location comes from the environment
		exporter := startProcess(t, "", port, &exporterSettings{
			scheme: "http",
			env:    []string{"EXPORTER_CONFIG_LOCATION=" + configPath},
		})
		t.Cleanup(exporter.cleanup)
		if err := exporter.waitUntilServing(startupTimeout); err != nil {
			t.Fatalf("exporter did not start: %v\n%s", err, exporter.Logs())
		}

		exporter.WaitForInstances(1)
	})

	t.Run("deprecated_servers_key_still_works", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().
			AddInstance(mockLogstash.URL()).
			UseLegacyServersKey())

		exporter.WaitForInstances(1)
		if !strings.Contains(exporter.Logs(), "deprecated") {
			t.Error("expected a deprecation warning for the servers key")
		}
	})

	t.Run("empty_config_file_uses_defaults", func(t *testing.T) {
		t.Parallel()

		// The default port 9198 may be taken on the machine running the
		// tests, so only check that the defaults are applied.
		configPath := WriteConfigFile(t, "")
		exporter := startProcess(t, configPath, 9198, &exporterSettings{scheme: "http"})
		t.Cleanup(exporter.cleanup)

		Eventually(t, startupTimeout, func() error {
			logs := exporter.Logs()
			if strings.Contains(logs, "port=9198") || strings.Contains(logs, "address already in use") {
				return nil
			}
			return fmt.Errorf("exporter did not try the default port yet")
		})
	})
}

func TestGracefulShutdown(t *testing.T) {
	t.Parallel()

	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT} {
		t.Run(signal.String(), func(t *testing.T) {
			t.Parallel()

			mockLogstash := NewMockLogstashServer(t)
			exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()), WithWatch())
			exporter.WaitForInstances(1)

			if err := exporter.Signal(signal); err != nil {
				t.Fatal(err)
			}

			code, err := exporter.WaitForExit(shutdownTimeout)
			if err != nil {
				t.Fatal(err)
			}
			if code != 0 {
				t.Errorf("expected exit code 0 after %s, got %d", signal, code)
			}
			if !strings.Contains(exporter.Logs(), "shutting down") {
				t.Errorf("expected the exporter to log its shutdown")
			}
			if strings.Contains(exporter.Logs(), `"level":"ERROR"`) {
				t.Errorf("expected a graceful shutdown to log no errors")
			}
			if _, err := exporter.Get("/version"); err == nil {
				t.Error("expected the server to stop after shutdown")
			}
		})
	}
}

// Ensure the binary exists before running long tests that only shell out
func TestMain(m *testing.M) {
	binaryPath := filepath.Join("..", "..", "test", "e2e", "bin", "logstash-exporter-e2e")
	if _, err := os.Stat(binaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "e2e binary not found at %s. Please run 'make e2e-prepare' first\n", binaryPath)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

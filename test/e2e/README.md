# End-to-End Tests

End-to-end tests for the logstash-exporter that run the actual binary against mock Logstash servers.

## Overview

The e2e tests:

- Build and run the actual exporter binary (not programmatic imports)
- Use mock Logstash servers serving fixture data from `/fixtures`
- Run in parallel with unique ports per test
- Test config reloads, multi-instance monitoring, and HTTP endpoints
- Do not require real Logstash instances

## Test Files

- `single_instance_test.go` - Single Logstash instance scenarios
- `multi_instance_test.go` - Multiple Logstash instances
- `config_reload_test.go` - Configuration hot-reload
- `endpoints_test.go` - HTTP endpoints (/metrics, /healthcheck, /version)
- `helpers.go` - Test utilities
- `mock_logstash.go` - Mock Logstash HTTP server

## Running Tests

### Using Makefile (Recommended)

```bash
# Build binary and run all e2e tests
make e2e

# Or run separately:
make e2e-prepare  # Build the binary
make e2e-run      # Run the tests

# Clean e2e artifacts
make e2e-clean
```

### Using Go Test Directly

```bash
# First, build the binary
make e2e-prepare

# Then run tests
go test -v github.com/kuskoman/logstash-exporter/test/e2e -parallel 4

# Run with race detection
go test -v -race github.com/kuskoman/logstash-exporter/test/e2e

# Run specific test
go test -v github.com/kuskoman/logstash-exporter/test/e2e -run TestSingleInstance

# Verify stability
go test -v github.com/kuskoman/logstash-exporter/test/e2e -count=3
```

**Note:** The binary must be built using `make e2e-prepare` before running tests. Tests will fail if the binary is not found.

## Key Features

**Binary Execution**: Tests use a pre-built binary (via `make e2e-prepare`) ensuring it's always up-to-date.

**Parallel Safe**: Each test gets a unique port via `GetFreePort()`.

**Mock Logstash**: Serves fixture data at `/` (node_info.json) and `/_node/stats` (node_stats.json).

**Config Builder**: Fluent API for creating test configurations.

## Example Test

```go
func TestExample(t *testing.T) {
    t.Parallel()

    port, _ := GetFreePort()
    mockLogstash, _ := NewMockLogstashServer()
    defer mockLogstash.Close()

    config := NewConfigBuilder().
        AddInstance(mockLogstash.URL()).
        WithServerPort(port).
        Build()

    exporter := StartExporter(t, TestConfig(t, config), port)
    defer exporter.Stop()

    time.Sleep(2 * time.Second)
    metrics := FetchMetrics(t, exporter.GetMetricsURL())
    AssertMetricExists(t, metrics, "logstash_info")
}
```

See `/TESTING-STANDARDS.md` for project conventions.

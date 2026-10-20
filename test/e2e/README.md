# End-to-End Tests

End-to-end tests for the logstash-exporter that run the actual binary against mock Logstash servers.

## Overview

The e2e tests:

- Build and run the actual exporter binary (not programmatic imports)
- Use mock Logstash servers serving fixture data from `/fixtures`
- Run in parallel, each exporter on its own port
- Wait for conditions by polling instead of sleeping for a fixed time
- Do not require real Logstash instances

## Test Files

- `single_instance_test.go` - Single Logstash instance scenarios
- `multi_instance_test.go` - Multiple Logstash instances
- `config_reload_test.go` - Configuration hot reload, including saves by rename and Kubernetes ConfigMap updates
- `endpoints_test.go` - HTTP endpoints (`/metrics`, `/healthcheck`, `/version`, `/`)
- `logstash_failures_test.go` - Unreachable, failing, slow and invalid Logstash instances
- `security_test.go` - Basic auth and TLS for the exporter and for Logstash
- `cli_test.go` - Command-line flags, invalid configuration and graceful shutdown
- `server_options_test.go` - Server read and write timeouts
- `metrics_output_test.go` - Snapshot and values of the exported metrics

The snapshot of the metrics output is stored in `__snapshots__`. Update it with
`UPDATE_SNAPS=true go test ./test/e2e -run TestMetricsOutput` after an intended change.

Test utilities:

- `helpers.go` - Starting and stopping the exporter, HTTP helpers
- `config_builder.go` - Fluent builder for exporter configuration files
- `mock_logstash.go` - Mock Logstash HTTP API
- `metrics.go` - Parsing and querying the metrics output
- `eventually.go` - Polling helpers (`Eventually`, `Consistently`)

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

# Run with race detection (requires cgo); CI does this
GOFLAGS=-race make e2e-prepare
go test -v -race github.com/kuskoman/logstash-exporter/test/e2e

# Run specific test
go test -v github.com/kuskoman/logstash-exporter/test/e2e -run TestSingleInstance

# Check for flakiness
go test github.com/kuskoman/logstash-exporter/test/e2e -count=20 -shuffle=on
```

**Note:** The binary must be built using `make e2e-prepare` before running tests. Tests will fail if the binary is not found.

## Key Features

**Binary Execution**: Tests use a pre-built binary (via `make e2e-prepare`) ensuring it's always up-to-date.

**Exporter lifecycle**: `StartExporter` writes the config, picks a free port, starts the binary and waits until it serves HTTP. If another process takes the port first, it retries on a new one. The exporter is stopped when the test ends, and its logs are printed only if the test failed.

**Mock Logstash**: Serves fixture data at `/` (node_info.json) and `/_node/stats` (node_stats.json). It records requests, and can return errors (`FailNext`, `SetStatusCode`), respond slowly (`SetDelay`) or serve other data (`SetResponse`). It is safe for concurrent use and closed automatically.

**Polling**: Use `exporter.WaitForMetrics`, `exporter.WaitForInstances` or `Eventually` instead of `time.Sleep`.

## Example Test

```go
func TestExample(t *testing.T) {
    t.Parallel()

    mockLogstash := NewMockLogstashServer(t)
    exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))

    metrics := exporter.WaitForInstances(1)
    if !metrics.Has("logstash_info_build") {
        t.Error("expected logstash_info_build to be exported")
    }
}
```

See `/TESTING-STANDARDS.md` for project conventions.

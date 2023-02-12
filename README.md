# Logstash-exporter

Export metrics from Logstash to Prometheus.
The project was created as rewrite of existing awesome application
[logstash_exporter](https://github.com/BonnierNews/logstash_exporter),
which was also written in Go, but it was not maintained for a long time.
A lot of code was reused from the original project.

**This project is under development and is not ready for use.**

## Building

### Makefile

#### Available Commands

- `make all`: Builds binary executables for Linux, macOS, and Windows and saves them in the out directory.
- `make run`: Runs the Go Exporter application.
- `make build-<OS>`: Builds a binary executable for the specified OS (`<OS>` can be `linux`, `darwin`, or `windows`).
- `make clean`: Deletes all binary executables in the out directory.
- `make (default)`: Runs the Go Exporter application.

#### File Structure

The main Go Exporter application is located in the cmd/exporter/main.go file.
The binary executables are saved in the out directory.

#### Example Usage

Build binary executables for all supported operating systems:

    make all

Run the Go Exporter application:

    make run

Build a binary executable for macOS:

    make build-darwin

Delete all binary executables:

    make clean

## Metrics

Table of exported metrics:

<!-- METRICS_TABLE_START -->

| Name | Type | Description |
| ----------- | ----------- | ----------- |
| logstash_exporter_build_info | gauge | A metric with a constant '1' value labeled by version, revision, branch, and goversion from which logstash_exporter was built. |
| logstash_info_build | counter | A metric with a constant '1' value labeled by build date, sha, and snapshot. |
| logstash_info_node | counter | A metric with a constant '1' value labeled by node name, version, host, http_address, and id. |
| logstash_info_pipeline_batch_delay | counter | pipeline_batch_delay |
| logstash_info_pipeline_batch_size | counter | pipeline_batch_size |
| logstash_info_pipeline_workers | counter | pipeline_workers |
| logstash_info_status | counter | A metric with a constant '1' value labeled by status. |
| logstash_stats_jvm_mem_heap_committed_bytes | gauge | jvm_mem_heap_committed_bytes |
| logstash_stats_jvm_mem_heap_max_bytes | gauge | jvm_mem_heap_max_bytes |
| logstash_stats_jvm_mem_heap_used_bytes | gauge | jvm_mem_heap_used_bytes |
| logstash_stats_jvm_mem_heap_used_percent | gauge | jvm_mem_heap_used_percent |
| logstash_stats_jvm_mem_non_heap_committed_bytes | gauge | jvm_mem_non_heap_committed_bytes |
| logstash_stats_jvm_threads_count | gauge | jvm_threads_count |
| logstash_stats_jvm_threads_peak_count | gauge | jvm_threads_peak_count |
| logstash_stats_jvm_uptime_millis | gauge | jvm_uptime_millis |
| logstash_stats_process_cpu_percent | gauge | process_cpu_percent |
| logstash_stats_process_cpu_total_millis | gauge | process_cpu_total_millis |
| logstash_stats_process_max_file_descriptors | gauge | process_max_file_descriptors |
| logstash_stats_process_mem_total_virtual | gauge | process_mem_total_virtual |
| logstash_stats_process_open_file_descriptors | gauge | process_open_file_descriptors |
| logstash_stats_queue_events_count | gauge | queue_events_count |
| logstash_stats_reload_failures | gauge | reload_failures |
| logstash_stats_reload_successes | gauge | reload_successes |

<!-- METRICS_TABLE_END -->

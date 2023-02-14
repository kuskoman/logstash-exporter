# Logstash-exporter

Export metrics from Logstash to Prometheus.
The project was created as rewrite of existing awesome application
[logstash_exporter](https://github.com/BonnierNews/logstash_exporter),
which was also written in Go, but it was not maintained for a long time.
A lot of code was reused from the original project.

**This project is currently under development it is not recommended to use it in production (yet).**

## Usage

### Running the app

The application can be run in two ways:

- using the binary executable
- using the Docker image

#### Binary Executable

The binary executable can be downloaded from the [releases page](https://github.com/kuskoman/logstash-exporter/releases).
Linux binary is available under `https://github.com/kuskoman/logstash-exporter/releases/download/v${VERSION}/logstash-exporter-linux`.
The binary can be run without additional arguments, as the configuration is loaded from the `.env` file and environment variables.

It is recommended to use the binary executable in combination with the [systemd](https://systemd.io/) service.
The application should not require any of root privileges, so it is recommended to run it as a non-root user.

#### Docker Image

The Docker image is available under `kuskoman/logstash-exporter:<tag>`.
You can pull the image using the following command:

    docker pull kuskoman/logstash-exporter:<tag>

You can browse tags on the [Docker Hub](https://hub.docker.com/r/kuskoman/logstash-exporter/tags).

The Docker image can be run using the following command:

    docker run -d \
        -p 9198:9198 \
        -e LOGSTASH_URL=http://logstash:9600 \
        kuskoman/logstash-exporter:<tag>

### Endpoints

- `/metrics`: Exposes metrics in Prometheus format.
- `/health`: Returns 200 if app runs properly.

### Configuration

The application can be configured using the following environment variables, which are also loaded from `.env` file:

| Variable Name  | Description                                   | Default Value           |
| -------------- | --------------------------------------------- | ----------------------- |
| `LOGSTASH_URL` | URL to Logstash API                           | `http://localhost:9600` |
| `PORT`         | Port on which the application will be exposed | `9198`                  |

All configuration variables can be checked in the [config directory](./config/).

## Building

### Makefile

#### Available Commands

- `make all`: Builds binary executables for Linux, macOS, and Windows and saves them in the out directory.
- `make run`: Runs the Go Exporter application.
- `make build-<OS>`: Builds a binary executable for the specified OS (`<OS>` can be linux, darwin, or windows).
- `make build-docker`: Builds a Docker image for the Go Exporter application.
- `make clean`: Deletes all binary executables in the out directory.
- `make test`: Runs all tests.
- `make compose`: Starts a Docker-compose configuration.
- `make wait-for-compose`: Starts a Docker-compose configuration and waits for it to be ready.
- `make compose-down`: Stops a Docker-compose configuration.
- `make verify-metrics`: Verifies the metrics from the Go Exporter application.
- `make pull`: Pulls the Docker image from the registry.
- `make logs`: Shows logs from the Docker-compose configuration.
- `make minify`: Minifies the binary executables.
- `make help`: Shows the available commands.

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

Build the Docker image:

    make build-docker

Delete all binary executables:

    make clean

Run all tests:

    make test

Start the Docker-compose configuration:

    make compose

Start the Docker-compose configuration and wait for it to be ready:

    make wait-for-compose

Stop the Docker-compose configuration:

    make compose-down

Verify the metrics from the Exporter application:

    make verify-metrics

Pull the Docker image from the registry:

    make pull

Show logs from the Docker-compose configuration:

    make logs

Minify the binary executables:

    make minify

Show the available commands:

    make help

## Helper Scripts

Application repository contains some helper scripts, which can be used to improve process
of building, testing, and running the application. These scripts are not useful for the end user,
but they can be useful for all potential contributors.
The helper scripts are located in the [scripts](./scripts/) directory.

### add_metrics_to_readme.sh

This [script](./scripts/add_metrics_to_readme.sh) is used to add metrics table to the README.md file.
Usage:

    ./scripts/add_metrics_to_readme.sh

### create_release_notes.sh

This [script](./scripts/create_release_notes.sh) is used to create release notes for the GitHub release.
Used primarily by the [CI workflow](./.github/workflows/go-application.yml).

### verify-metrics.sh

This [script](./scripts/verify-metrics.sh) is used to verify the metrics from the Go Exporter application.
Can be used both locally and in the CI workflow.

    ./scripts/verify-metrics.sh

## Testing process

The application contains both unit and integration tests. All the tests are executed in the CI workflow.

### Unit Tests

Unit tests are located in the same directories as the tested files.
To run all unit tests, use the following command:

    make test

### Integration Tests

Integration tests checks if Prometheus metrics are exposed properly.
To run them you must setup development [docker-compose](./docker-compose.yml) file.

    make wait-for-compose

Then you can run the tests:

    make verify-metrics

## Roadmap

These are the features that are planned to be implemented in the future:

- [ ] Parse missing metrics
- [ ] Add description to all metrics
- [ ] Improve test coverage
- [ ] Build Helm chart
- [ ] Automatically add release notes to GitHub release

Feel free to create an issue if you have any suggestions, ideas or questions.

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
| logstash_info_up | gauge | A metric that returns 1 if the node is up, 0 otherwise. |
| logstash_stats_jvm_mem_heap_committed_bytes | gauge | jvm_mem_heap_committed_bytes |
| logstash_stats_jvm_mem_heap_max_bytes | gauge | jvm_mem_heap_max_bytes |
| logstash_stats_jvm_mem_heap_used_bytes | gauge | jvm_mem_heap_used_bytes |
| logstash_stats_jvm_mem_heap_used_percent | gauge | jvm_mem_heap_used_percent |
| logstash_stats_jvm_mem_non_heap_committed_bytes | gauge | jvm_mem_non_heap_committed_bytes |
| logstash_stats_jvm_threads_count | gauge | jvm_threads_count |
| logstash_stats_jvm_threads_peak_count | gauge | jvm_threads_peak_count |
| logstash_stats_jvm_uptime_millis | gauge | jvm_uptime_millis |
| logstash_stats_pipeline_events_duration | counter | pipeline_events_duration |
| logstash_stats_pipeline_events_filtered | counter | pipeline_events_filtered |
| logstash_stats_pipeline_events_in | counter | pipeline_events_in |
| logstash_stats_pipeline_events_out | counter | pipeline_events_out |
| logstash_stats_pipeline_events_queue_push_duration | counter | pipeline_events_queue_push_duration |
| logstash_stats_pipeline_queue_events_count | counter | pipeline_queue_events_count |
| logstash_stats_pipeline_queue_events_queue_size | counter | pipeline_queue_events_queue_size |
| logstash_stats_pipeline_queue_max_size_in_bytes | counter | pipeline_queue_max_size_in_bytes |
| logstash_stats_pipeline_reloads_failures | counter | pipeline_reloads_failures |
| logstash_stats_pipeline_reloads_successes | counter | pipeline_reloads_successes |
| logstash_stats_process_cpu_percent | gauge | process_cpu_percent |
| logstash_stats_process_cpu_total_millis | gauge | process_cpu_total_millis |
| logstash_stats_process_max_file_descriptors | gauge | process_max_file_descriptors |
| logstash_stats_process_mem_total_virtual | gauge | process_mem_total_virtual |
| logstash_stats_process_open_file_descriptors | gauge | process_open_file_descriptors |
| logstash_stats_queue_events_count | gauge | queue_events_count |
| logstash_stats_reload_failures | gauge | reload_failures |
| logstash_stats_reload_successes | gauge | reload_successes |

<!-- METRICS_TABLE_END -->

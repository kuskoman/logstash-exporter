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

Each binary should contain a SHA256 checksum file, which can be used to verify the integrity of the binary.

    VERSION="test-tag" \
    OS="linux" \
    wget "https://github.com/kuskoman/logstash-exporter/releases/download/${VERSION}/logstash-exporter-${OS}" && \
    wget "https://github.com/kuskoman/logstash-exporter/releases/download/${VERSION}/logstash-exporter-${OS}.sha256" && \
    sha256sum -c logstash-exporter-${OS}.sha256

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
| `HOST`         | Host on which the application will be exposed | empty string            |

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

- [x] Parse missing metrics (if you find any useful missing metrics, please create an issue)
- [x] Add description to all metrics
- [x] Improve test coverage
- [ ] Build Helm chart
- [x] Automatically add release notes to GitHub release

Feel free to create an issue if you have any suggestions, ideas or questions.

## Metrics

Table of exported metrics:

<!-- METRICS_TABLE_START -->

| Name | Type | Description |
| ----------- | ----------- | ----------- |
| logstash_exporter_build_info | gauge | A metric with a constant '1' value labeled by version, revision, branch, goversion from which logstash_exporter was built, and the goos and goarch for the build. |
| logstash_info_build | counter | A metric with a constant '1' value labeled by build date, sha, and snapshot. |
| logstash_info_node | counter | A metric with a constant '1' value labeled by node name, version, host, http_address, and id. |
| logstash_info_pipeline_batch_delay | counter | Amount of time to wait for events to fill the batch before sending to the filter and output stages. |
| logstash_info_pipeline_batch_size | counter | Number of events to retrieve from the input queue before sending to the filter and output stages. |
| logstash_info_pipeline_workers | counter | Number of worker threads that will process pipeline events. |
| logstash_info_status | counter | A metric with a constant '1' value labeled by status. |
| logstash_info_up | gauge | A metric that returns 1 if the node is up, 0 otherwise. |
| logstash_stats_jvm_mem_heap_committed_bytes | gauge | Amount of heap memory in bytes that is committed for the Java virtual machine to use. |
| logstash_stats_jvm_mem_heap_max_bytes | gauge | Maximum amount of heap memory in bytes that can be used for memory management. |
| logstash_stats_jvm_mem_heap_used_bytes | gauge | Amount of used heap memory in bytes. |
| logstash_stats_jvm_mem_heap_used_percent | gauge | Percentage of the heap memory that is used. |
| logstash_stats_jvm_mem_non_heap_committed_bytes | gauge | Amount of non-heap memory in bytes that is committed for the Java virtual machine to use. |
| logstash_stats_jvm_mem_pool_committed_bytes | gauge | Amount of bytes that are committed for the Java virtual machine to use in a given JVM memory pool. |
| logstash_stats_jvm_mem_pool_max_bytes | gauge | Maximum amount of bytes that can be used in a given JVM memory pool. |
| logstash_stats_jvm_mem_pool_peak_max_bytes | gauge | Highest value of bytes that were used in a given JVM memory pool. |
| logstash_stats_jvm_mem_pool_peak_used_bytes | gauge | Peak used bytes of a given JVM memory pool. |
| logstash_stats_jvm_mem_pool_used_bytes | gauge | Currently used bytes of a given JVM memory pool. |
| logstash_stats_jvm_threads_count | gauge | Number of live threads including both daemon and non-daemon threads. |
| logstash_stats_jvm_threads_peak_count | gauge | Peak live thread count since the Java virtual machine started or peak was reset. |
| logstash_stats_jvm_uptime_millis | gauge | Uptime of the JVM in milliseconds. |
| logstash_stats_pipeline_events_duration | counter | Time needed to process event. |
| logstash_stats_pipeline_events_filtered | counter | Number of events that have been filtered out by this pipeline. |
| logstash_stats_pipeline_events_in | counter | Number of events that have been inputted into this pipeline. |
| logstash_stats_pipeline_events_out | counter | Number of events that have been processed by this pipeline. |
| logstash_stats_pipeline_events_queue_push_duration | counter | Time needed to push event to queue. |
| logstash_stats_pipeline_queue_events_count | counter | Number of events in the queue. |
| logstash_stats_pipeline_queue_events_queue_size | counter | Number of events that the queue can accommodate |
| logstash_stats_pipeline_queue_max_size_in_bytes | counter | Maximum size of given queue in bytes. |
| logstash_stats_pipeline_reloads_failures | counter | Number of failed pipeline reloads. |
| logstash_stats_pipeline_reloads_successes | counter | Number of successful pipeline reloads. |
| logstash_stats_process_cpu_percent | gauge | CPU usage of the process. |
| logstash_stats_process_cpu_total_millis | gauge | Total CPU time used by the process. |
| logstash_stats_process_max_file_descriptors | gauge | Limit of open file descriptors. |
| logstash_stats_process_mem_total_virtual | gauge | Total virtual memory used by the process. |
| logstash_stats_process_open_file_descriptors | gauge | Number of currently open file descriptors. |
| logstash_stats_queue_events_count | gauge | Number of events in the queue. |
| logstash_stats_reload_failures | gauge | Number of failed reloads. |
| logstash_stats_reload_successes | gauge | Number of successful reloads. |

<!-- METRICS_TABLE_END -->

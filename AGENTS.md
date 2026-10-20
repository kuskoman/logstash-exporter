# AGENTS.md

Guidance for AI coding agents and human contributors working on logstash-exporter.
It complements [CONTRIBUTING.md](./CONTRIBUTING.md) and [TESTING-STANDARDS.md](./TESTING-STANDARDS.md).

> The roadmap below reflects the state of `master` in October 2026. Update it as items are done.

## Project overview

logstash-exporter exports Logstash metrics (node info and node stats APIs) to Prometheus.
There are two binaries:

- **exporter** (`cmd/exporter`): scrapes a static list of Logstash instances from the config file.
- **controller** (`cmd/controller`): runs in Kubernetes and discovers Logstash pods and services
  from annotations (`logstash-exporter.io/url`), then scrapes them.

Both share the same HTTP server (`/metrics`, `/healthcheck`, `/version`) and startup code.

## Branching and releases

| Branch   | Releases               | Status                                    |
| -------- | ---------------------- | ----------------------------------------- |
| `v1`     | `v1.x.y`               | Maintained until further notice           |
| `master` | `v2.x.y`, `v2.x.y-preN` | v2, currently in pre-release (`-preN`)   |

- Fixes that apply to both lines need a separate PR for each branch. Never merge `master` into `v1`.
- Until v2.0.0 is released, v2 is not stable: breaking changes to metrics, configuration and the
  Helm chart are allowed on `master`, but must be called out in the release notes. After v2.0.0
  they require a new major version.
- Releases are created with `./scripts/release_tag.sh` on the release branch. It bumps the chart
  version, regenerates the chart README, commits and tags. It does not push.
- Pushing a tag runs the release jobs in `.github/workflows/go-application.yml`: binaries, Docker
  images (Docker Hub and ghcr.io), and a GitHub release whose notes come from
  `./scripts/create_release_notes.sh`.
- Both scripts only consider tags of the same major version (`v1.*` on `v1`, `v2.*` on `master`),
  because `master` still contains the old v1 tags.
- Scripts in `scripts/` must work with the bash 3.2 and BSD tools shipped with macOS
  (no `sed -i` without a suffix, no `\s` in sed, no associative arrays).

## Repository layout

| Path | Contents |
| ---- | -------- |
| `cmd/exporter`, `cmd/controller` | Entry points |
| `internal/startup_manager` | Lifecycle: config loading, hot reload, server start and graceful shutdown |
| `internal/file_watcher` | Config file watching for `-watch` (debounced, handles renames and ConfigMap symlink swaps) |
| `internal/collectors` | Prometheus collectors for node info and node stats |
| `internal/fetcher` | Logstash API client and response types |
| `internal/k8s_controller` | Kubernetes discovery for the controller |
| `internal/server` | HTTP server and endpoints |
| `pkg/config`, `pkg/tls`, `pkg/collector_manager` | Configuration, TLS and auth, collector orchestration |
| `chart/` | Helm chart, with unit tests in `chart/tests` |
| `.helm/logstash-integration-test-chart` | Chart used by the kind integration test in CI |
| `test/e2e` | End-to-end tests that run the real binary against a mock Logstash |
| `fixtures/` | Logstash API responses and TLS certificates used by tests |
| `scripts/` | Release, README and verification scripts |

## Commands

| Task | Command |
| ---- | ------- |
| Unit tests | `make test` (uses `-race`, which needs cgo) or `go test ./...` without e2e |
| E2E tests | `make e2e` (builds `test/e2e/bin/logstash-exporter-e2e` first) |
| E2E with race detector, as in CI | `GOFLAGS=-race make e2e-prepare && go test -race -shuffle=on ./test/e2e` |
| Lint | `golangci-lint run ./...` (v2, CI uses `latest`) |
| Helm unit tests | `make helm-test` (plugin: `helm plugin install https://github.com/helm-unittest/helm-unittest --verify=false`) |
| Helm README and schema | `make install-helm-readme` once, then `make helm-readme` |
| Makefile section of the README | `make update-readme-descriptions` |
| Update unit test snapshots | `make update-snapshots` |
| Update the e2e metrics snapshot | `UPDATE_SNAPS=true go test ./test/e2e -run TestMetricsOutput` |
| Full local stack (Logstash, Elasticsearch, Prometheus, Grafana) | `make wait-for-compose`, then `CI=true make verify-metrics` |

CI fails if `go mod tidy`, `make helm-readme` or `make update-readme-descriptions` would change a file.

## Development practices

### Go code

- Follow the existing style. Return errors instead of panicking, wrap them with context, and log
  with `log/slog`.
- Inject dependencies through small interfaces or factory functions so that tests can replace them.
  `StartupManager` is the reference: config source, server factory and Prometheus registerer are injected.
- Avoid package-level side effects and global state. Register collectors through an injected
  `prometheus.Registerer`. The default registry is only used in production wiring.
- Keep behaviour for existing users: config keys that are removed must still load (unknown keys are ignored).

### Tests

- Tests must exercise the real code. Do not copy production methods into test helpers or test a
  reimplementation of the logic. Mock the dependencies instead.
- When fixing a bug, add a test that fails without the fix and check that it does.
- Use `t.Parallel()` and per-test state (temporary directories, free ports via `127.0.0.1:0`,
  private Prometheus registries). Never bind fixed ports such as 9198 in tests.
- In e2e tests, poll with `Eventually` or `exporter.WaitForMetrics` instead of `time.Sleep`.
  Use the helpers in `test/e2e` (`StartExporter`, `NewMockLogstashServer`, `ConfigBuilder`).
- Tests should pass with `-race -count=5 -shuffle=on`. CI runs a nightly flakiness check.
- Helm unit tests assert exact rendered values with `equal` (not only `isKind`), and use full
  YAML blocks instead of regexes for multi-line values.
- Kubernetes behaviour is tested in CI (kind job in `go-application.yml`), not on local clusters.

### Metrics

- Adding, renaming or removing a metric changes what users scrape. Update
  `scripts/snapshots/metric_names.txt`, the metrics table in `README.md` and the e2e metrics
  snapshot, and mention it in the release notes.
- Renaming or removing a metric after v2.0.0 is a breaking change.
- There is no official Grafana dashboard for v2. Community dashboards are welcome, but metric
  changes do not need to keep any dashboard in sync.

### Helm chart

- After changing `chart/values.yaml`, regenerate `chart/README.md` and `chart/schema.json` with
  `make helm-readme`, and add tests in `chart/tests`.
- Changes that affect existing installations (resource names, defaults) must be called out in the
  release notes.

### Commits and pull requests

- Small, focused commits with a message that explains the problem and the reason for the change.
- One logical change per commit: production fixes, their tests, and refactors in separate commits
  where practical.
- Work on one issue or roadmap item per branch. When the branch completes a roadmap item, mark it
  done (`- [x]`) in the roadmap below in the same branch, so the roadmap is always up to date.
  If the branch is already frozen for review, mark it in a separate docs branch instead.

## Supported versions

- **Logstash:** only the latest 9.x is officially supported and tested (see `docker-compose.yml`).
- **Go:** the version in `go.mod`.
- **Kubernetes and Helm:** whatever the kind job in CI uses (Helm 4).

## Roadmap to a stable v2.0.0

v2 has been in pre-release since December 2023 (`v2.0.0-pre1` … `v2.0.0-pre11`). The goals below
are what remains before v2 can be recommended over v1.

### Blockers for v2.0.0

- [ ] **Kubernetes controller is production-ready.** It is part of v2.0.0, so it needs:
  - [ ] Unit tests for `internal/k8s_controller` (currently 0% coverage), e.g. with
        `k8s.io/client-go/kubernetes/fake`: pod and service add, update and delete, annotation
        parsing, `$(pod_ip)` substitution, namespace filtering, non-running pods.
  - [x] Investigate and fix #455 (annotated Logstash ignored).
  - [ ] Keep the kind CI check (`scripts/verify_k8s_controller.sh`) green, and extend it to cover
        pod removal and service discovery.
  - [x] Decide what happens to the static `logstash.instances` defaults in controller mode, where
        the default `http://localhost:9600` instance is scraped as well.
- [x] **Build info is empty (#502).** `logstash_exporter_build_info` reads
      `github.com/prometheus/common/version`, but the build only sets `pkg/config.Version` and
      `GitCommit`. Set the `prometheus/common/version` variables in the `Makefile` and
      `Dockerfile` ldflags, and add an e2e test that checks the labels.
- [x] **Chart and image versions are out of sync (#493).** The chart is published on every push
      to `master` (`publish-helm-chart.yml`), so chart 2.0.0 was released while no `v2.0.0` image
      exists. Publish the chart from release tags only, with `appVersion` matching the tag.
- [ ] **Freeze the public interface.** Breaking changes are still allowed before v2.0.0, so this is
      the last chance to rename metrics, labels, config keys, chart values and chart resource names.
      Afterwards they are frozen until v3.
- [ ] **Release v2.0.0 (#519)**, update the README (remove "V2 is in beta") and the migration guide.
- [x] **Clarify the Grafana dashboard in the README.** The "Grafana Dashboard" section links a
      community dashboard built for v1 metrics. State that it is community-maintained and that v2
      has no official dashboard.

### Should do before or shortly after v2.0.0

- [ ] **Organize the loose files in the repository root.** The root has 22 tracked files next to
      the source directories. Proposal:
  - Keep at the root what GitHub and tools expect there: `README.md`, `LICENSE`,
    `CONTRIBUTING.md`, `SECURITY.md`, `AGENTS.md`, `CLAUDE.md`, `Makefile`, `go.mod`, `go.sum`,
    `docker-compose.yml`, `codecov.yml`, `.air.toml`.
  - Move the guides to `docs/`: `HELM-DEPLOY.md`, `STANDALONE.md`, `TLS-CONFIGURATION.md`,
    `TESTING-STANDARDS.md`.
  - Move `config.example.yml` next to the other examples, for example `examples/`.
  - Decide whether `Dockerfile`, `Dockerfile.controller` and `Dockerfile.dev` move to a
    directory such as `build/`. This touches the Makefile, `docker-compose.yml` and the CI
    workflows.
  - Update every reference: README links, `AGENTS.md`, `CONTRIBUTING.md`, the chart README,
    the Makefile and the workflows.

- [x] Verify that #447 (`Infinity` in node stats) is fixed on `master` and close it. All flow
      fields use `InfinityFloat` now.
- [x] `automountServiceAccountToken` option in the chart (#499).
- [ ] Replace archived actions in the release jobs: `actions/create-release` and
      `actions/upload-release-asset` (for example with `softprops/action-gh-release`), and replace
      the deprecated `::set-output` in the prerelease step with `$GITHUB_OUTPUT`.
- [ ] Replace the deprecated Elastic Logstash Helm chart (last version 8.5.1) in
      `.helm/logstash-integration-test-chart` with a plain Deployment.
- [ ] Remove the `ndeloof/install-compose-action` legacy wrapper and use `docker compose`.
- [ ] Add a Logstash 9.x fixture captured from a real instance (`fixtures/` is from 8.15.2).
- [x] Format the files `gofmt -l` reports and add a format check to the lint job.
- [ ] Raise coverage in `pkg/config` (56%), `internal/file_utils` (61%),
      `pkg/collector_manager` (65%) and `internal/fetcher/logstash_client` (67%).

### Later

- [ ] Export the plugin-level flow metrics (`worker_utilization`, `worker_millis_per_event`),
      see the TODO in `internal/collectors/nodestats/pipeline_subcollector.go`.
- [ ] Compatibility matrix of metrics across Logstash versions (#88).
- [x] Clean up untagged images in the GitHub package registry (#302).
- [ ] `/healthcheck` only checks that Logstash answers with HTTP 200; consider also checking that
      the response is valid.

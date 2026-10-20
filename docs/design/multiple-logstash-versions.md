# Supporting multiple Logstash versions

> **Status: not planned.** The exporter supports only the latest Logstash 9.x. This document
> records how support for Logstash 7, 8 and 9 could be added if that changes.

## How the versions differ

The node info (`/`) and node stats (`/_node/stats`) APIs keep their structure across versions,
but fields are added and removed, also in minor releases.

Differences between 8.15 and 9.5, measured by comparing captured responses:

- Removed in 9: the internal `.monitoring-logstash` pipeline (legacy monitoring was removed).
- New in 9: per-pipeline batch statistics (`batch.byte_size`, `batch.event_count`), the pipeline
  settings (`pipeline.workers`, `batch_size`, `batch_delay`) and `worker_millis_per_event` of
  output plugins.

Differences between 7 and 8, from the Elastic documentation (not verified in this repository):

- Logstash 7 has no `flow` section. Flow metrics were added in 8.5.
- Some fields only exist in later 8.x releases, for example the health report added in 8.16.

So the question is not only "7, 8 or 9", but which fields a given Logstash returns.

## Problem in the current code

The response types in `internal/fetcher/responses` use plain values, and the collectors emit every
metric unconditionally. A field that Logstash does not return is parsed as 0 and exported as 0.
Against Logstash 7, every `flow_*` metric would be exported as 0 instead of being absent, which
cannot be told apart from a real 0. This has to be fixed before more than one version is supported.

## Proposed design

One exporter and one set of metric names. The Logstash version only decides which series exist.

1. **Detect the version per instance.** The node info API already returns `version`, which the
   exporter fetches for `logstash_info_node`. Cache the major and minor version per instance,
   refresh it after a failed scrape or a configuration reload, and allow an optional per-instance
   override in the configuration.
2. **Export only what Logstash returned.** Parse optional fields as pointers (or check that they
   are present) and skip the metric when the field is missing. This covers most version
   differences without version checks in the collectors.
3. **Version gates only where presence is not enough**, for example when a field exists in two
   versions with a different meaning. Each metric definition can declare an optional minimum and
   maximum Logstash version, in one table.
4. **Adapters in the fetcher layer for real schema conflicts**, where the same field has a
   different type or structure. A small adapter per version converts the response to one internal
   model, so the collectors stay version-agnostic. Only add this when the first conflict appears.
5. **Same metric names in all versions.** Versions differ in which series exist, never in names.
   `logstash_info_node{version="..."}` lets users filter or group by version.

## Testing

- One fixture per supported major version, captured without edits from a real instance, for
  example `fixtures/7.17/`, `fixtures/8.19/`, `fixtures/9.5/`. This replaces the current rule of a
  single fixture version.
- One e2e metrics snapshot per version. The differences between the snapshots are the
  compatibility matrix (#88), which can be generated from them instead of written by hand.
- The docker-compose integration test runs as a matrix over the supported versions, each with its
  own metric names snapshot for `scripts/verify_metrics.sh`.

## Cost

- The compose integration test takes about three times as long.
- Three fixtures and snapshots to refresh when Logstash changes.
- Logstash 7 is at or near the end of Elastic's support; check the Elastic support matrix before
  investing in it. Supporting 8 and 9 covers most users.

## First step if this is picked up

Stop exporting missing fields as 0 (point 2). It is useful even with a single supported version,
because Logstash 9 minor releases also add and remove fields.

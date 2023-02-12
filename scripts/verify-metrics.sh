#! /usr/bin/env bash

PROMETHEUS_RESPONSE=$(curl -s -X GET http://localhost:9090/api/v1/label/__name__/values)
STATUS=$(echo $PROMETHEUS_RESPONSE | jq -r '.status')
if [ "$STATUS" != "success" ]; then
    echo "Prometheus API returned an error: $PROMETHEUS_RESPONSE"
    exit 1
fi

LOGSTASH_METRICS=$(echo $PROMETHEUS_RESPONSE | jq -r '.data[]' | grep -E '^logstash_')
SCRIPT_LOCATION=$(dirname "$0")
SNAPSHOT_DIR="$SCRIPT_LOCATION/snapshots"
SNAPSHOT_FILE="$SNAPSHOT_DIR/metric_names.txt"

mkdir -p "$SNAPSHOT_DIR"

if [ ! -f "$SNAPSHOT_FILE" ]; then
    echo "Snapshot file $SNAPSHOT_FILE does not exist. Creating it."
    echo "$LOGSTASH_METRICS" > "$SNAPSHOT_FILE"
fi

SNAPSHOT_METRICS=$(cat "$SNAPSHOT_FILE")

echo "Checking that all metrics are in the snapshot file $SNAPSHOT_FILE"
for METRIC in $LOGSTASH_METRICS; do
    echo "Checking existence of metric $METRIC"
    if [[ ! "$SNAPSHOT_METRICS" =~ "$METRIC" ]]; then
        if [ "$CI" == "true" ]; then
            echo "Metric $METRIC is not in the snapshot file $SNAPSHOT_FILE"
            exit 1
        else
            echo "Metric $METRIC is not in the snapshot file $SNAPSHOT_FILE. Updating it."
            echo "$LOGSTASH_METRICS" >> "$SNAPSHOT_FILE"
        fi
    fi
done

for METRIC in $LOGSTASH_METRICS; do
    echo "Checking metric endpoint for $METRIC"
    PROMETHEUS_RESPONSE=$(curl -s -X GET http://localhost:9090/api/v1/series?match[]=$METRIC)
    STATUS=$(echo $PROMETHEUS_RESPONSE | jq -r '.status')
    if [ "$STATUS" != "success" ]; then
        echo "Prometheus API returned an error: $PROMETHEUS_RESPONSE"
        exit 1
    fi
done

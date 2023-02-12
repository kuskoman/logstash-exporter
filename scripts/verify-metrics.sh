#! /usr/bin/env bash

prometheus_response=$(curl -s -X GET http://localhost:9090/api/v1/label/__name__/values)
status=$(echo $prometheus_response | jq -r '.status')
if [ "$status" != "success" ]; then
    echo "Prometheus API returned an error: $prometheus_response"
    exit 1
fi

logstash_metrics=$(echo $prometheus_response | jq -r '.data[]' | grep -E '^logstash_')
script_location=$(dirname "$0")
snapshot_dir="$script_location/snapshots"
snapshot_file="$snapshot_dir/metric_names.txt"

mkdir -p "$snapshot_dir"

if [ ! -f "$snapshot_file" ]; then
    echo "Snapshot file $snapshot_file does not exist. Creating it."
    echo "$logstash_metrics" > "$snapshot_file"
fi

snapshot_metrics=$(cat "$snapshot_file")

echo "Checking that all metrics are in the snapshot file $snapshot_file"
for metric in $logstash_metrics; do
    echo "Checking existence of metric $metric"
    if [[ ! "$snapshot_metrics" =~ "$metric" ]]; then
        if [ "$CI" == "true" ]; then
            echo "Metric $metric is not in the snapshot file $snapshot_file"
            exit 1
        else
            echo "Metric $metric is not in the snapshot file $snapshot_file. Updating it."
            echo "$logstash_metrics" >> "$snapshot_file"
        fi
    fi
done

for metric in $logstash_metrics; do
    echo "Checking metric endpoint for $metric"
    prometheus_response=$(curl -s -X GET http://localhost:9090/api/v1/series?match[]=$metric)
    status=$(echo $prometheus_response | jq -r '.status')
    if [ "$status" != "success" ]; then
        echo "Prometheus API returned an error: $prometheus_response"
        exit 1
    fi
done

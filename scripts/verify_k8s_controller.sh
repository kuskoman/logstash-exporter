#!/usr/bin/env bash

# Verifies that the exporter running in Kubernetes controller mode discovers
# an annotated Logstash pod and exports its metrics.
#
# Usage: verify_k8s_controller.sh <controller service name> [timeout seconds]

set -euo pipefail

service_name="${1:?usage: $0 <controller service name> [timeout seconds]}"
timeout_seconds="${2:-300}"
annotation="logstash-exporter.io/url"
local_port=19198

logstash_pod=$(kubectl get pods -o json |
    jq -r --arg annotation "$annotation" \
        '.items[] | select(.metadata.annotations[$annotation] != null) | "\(.metadata.namespace)/\(.metadata.name)"' |
    head -n 1)

if [ -z "$logstash_pod" ]; then
    echo "No pod with the $annotation annotation found"
    exit 1
fi
echo "Expecting the controller to discover $logstash_pod"

kubectl port-forward "service/$service_name" "$local_port:9198" >/dev/null 2>&1 &
port_forward_pid=$!
trap 'kill "$port_forward_pid" 2>/dev/null || true' EXIT

expected_metric="logstash_info_up\\{hostname=\"[^\"]*\",instance_name=\"$logstash_pod\"\\} 1"
deadline=$((SECONDS + timeout_seconds))

while [ "$SECONDS" -lt "$deadline" ]; do
    metrics=$(curl -sf "http://127.0.0.1:$local_port/metrics" || true)
    if grep -Eq "^$expected_metric$" <<<"$metrics"; then
        echo "Controller exports metrics for $logstash_pod"
        grep -E "^logstash_info_up" <<<"$metrics"
        exit 0
    fi
    sleep 5
done

echo "Controller did not export logstash_info_up 1 for $logstash_pod within ${timeout_seconds}s"
echo "Last logstash_info_up values:"
grep -E "^logstash_info_up" <<<"${metrics:-}" || echo "(none)"
exit 1

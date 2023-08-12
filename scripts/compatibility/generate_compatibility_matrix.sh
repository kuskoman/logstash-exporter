#! /usr/bin/env bash

set -euo pipefail

getMetrics() {
    port=$1
    curl -Ss "http://localhost:${port}/api/v1/targets/metadata" | jq -c '.data[] | select(.metric | startswith("logstash"))' | while read -r line; do
        echo "$line" | jq -r '.metric'
    done
}

wait_for_prometheus() {
    port=$1
    echo "Waiting for Prometheus at port ${port} to be ready..." >&2
    while true; do
        metrics=$(curl -Ss "http://localhost:${port}/api/v1/targets/metadata")
        if [[ ! -z $(echo "$metrics" | jq '.data[]') ]]; then
            break
        fi
        echo "Prometheus at port ${port} is not ready yet, waiting..." >&2
        sleep 1
    done
    echo "Prometheus at port ${port} is ready!" >&2
}

script_dir=$(dirname "$(readlink -f "$0")")
generator_script_path="$script_dir/generate_compatibility_docker_compose.py"
output_json=$(python3 $generator_script_path)
dockerfile_path="$script_dir/../../docker-compose.compatibility.yml"
docker-compose -f $dockerfile_path up -d

declare -A version_ports
for key in $(echo $output_json | jq -r 'keys[]'); do
    version_ports[$key]=$(echo $output_json | jq -r --arg key "$key" '.[$key]')
    wait_for_prometheus ${version_ports[$key]}
done

all_metrics=""
for port in ${version_ports[@]}; do
    all_metrics+=$(getMetrics $port)
    all_metrics+="\n"
done

all_metrics=$(echo -e "$all_metrics" | sort -u)

FILE="$script_dir/../../COMPATIBILITY_MATRIX.md"
echo "| Metric | $(echo "${!version_ports[@]}" | tr ' ' '\t') |" > $FILE
echo "| ----------- | $(echo "${!version_ports[@]}" | sed 's/.*/-----------/') |" >> $FILE

for metric in $all_metrics; do
    row="| $metric |"
    for version in "${!version_ports[@]}"; do
        if getMetrics ${version_ports[$version]} | grep -q $metric; then
            row+=" ✅ |"
        else
            row+=" ❌ |"
        fi
    done
    echo "$row" >> $FILE
done

docker-compose -f $dockerfile_path down -v --remove-orphans

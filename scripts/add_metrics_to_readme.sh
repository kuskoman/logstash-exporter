#! /usr/bin/env bash

function getMetrics() {
    curl -Ss http://localhost:9090/api/v1/targets/metadata | jq -c '.data[] | select(.metric | startswith("logstash"))' | while read -r line; do
        metric=$(echo "$line" | jq -r '.metric')
        type=$(echo "$line" | jq -r '.type')
        help=$(echo "$line" | jq -r '.help')
        echo "| $metric | $type | $help |"
    done
}

FILE=README.md
while IFS= read -r line; do LINES+=("$line"); done < $FILE

startLine=$(grep -n "^<!-- METRICS_TABLE_START -->" $FILE | awk -F: '{print $1}')
endLine=$(grep -n "^<!-- METRICS_TABLE_END -->" $FILE | awk -F: '{print $1}')

metricsTable=$(echo "| Name | Type | Description |
| ----------- | ----------- | ----------- |
$(getMetrics | sort --version-sort)")

for ((i=0; i<${#LINES[@]}; i++)); do
    if [ $i -eq $startLine ]; then
        echo -e "${LINES[i]}"
        echo "$metricsTable"
    elif [ $i -gt $startLine ] && [ $i -lt $((endLine-2)) ]; then # -2 because of the empty line before the end marker
        continue
    else
        echo -e "${LINES[i]}"
    fi
done > $FILE

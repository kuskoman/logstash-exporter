#! /usr/bin/env bash

function getMetrics() {
    curl -Ss http://localhost:9090/api/v1/targets/metadata | jq -c '.data[] | select(.metric | startswith("logstash"))' | while read -r line; do
        metric=$(echo "$line" | jq -r '.metric')
        type=$(echo "$line" | jq -r '.type')
        help=$(echo "$line" | jq -r '.help')
        echo "| $metric | $type | $help |"
    done
}

function failureConfigChange() {
	local logstashCID
	logstashCID=$( docker ps -a | grep 'logstash-exporter-logstash-1' | awk '{print $1}' )
	local logstashPID
	logstashPID=$( docker exec "$logstashCID" sh -c "echo \$(ps aux | grep logstash) | awk '{print \$2}'" )
	local logstashConf
	logstashConf='/usr/share/logstash/pipeline/logstash.conf'

	docker exec -it "$logstashCID" sh -c "echo 'Wrong Config' >> $logstashConf"
	# reload logstash
	docker exec -it "$logstashCID" sh -c "kill -1 $logstashPID"

	# bring back previous config 
	# walk around with 'cp' to avoid replacing config inside container
	docker exec "$logstashCID" sh -c "sed '\$d' $logstashConf > /tmp/prev_logstash.conf" 
	docker exec "$logstashCID" sh -c "cp /tmp/prev_logstash.conf $logstashConf"

	# reload logstash
	docker exec -it "$logstashCID" sh -c "kill -1 $logstashPID"

}

failureConfigChange

FILE=README.md
while IFS= read -r line; do LINES+=("$line"); done < $FILE

startLine=$(grep -n "^<!-- METRICS_TABLE_START -->" $FILE | awk -F: '{print $1}')
endLine=$(grep -n "^<!-- METRICS_TABLE_END -->" $FILE | awk -F: '{print $1}')

metricsTable="| Name | Type | Description |
| ----------- | ----------- | ----------- |
$(getMetrics | sort --version-sort)"

for ((i=0; i<${#LINES[@]}; i++)); do
    if [ $i -eq "$startLine" ]; then
        echo -e "${LINES[i]}"
        echo "$metricsTable"
    elif [ $i -gt "$startLine" ] && [ $i -lt $((endLine-2)) ]; then # -2 because of the empty line before the end marker
        continue
    else
        echo -e "${LINES[i]}"
    fi
done > $FILE

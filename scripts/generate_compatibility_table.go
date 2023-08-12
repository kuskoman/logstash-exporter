#!/bin/bash

compatibilityTable="| Logstash Version | Metric 1 | Metric 2 | Metric 3 |\n|------------------|----------|----------|----------|\n"

for service in $(docker service ls --format "{{.Name}}"); do
	if [[ $service == logstash_* ]]; then
		version=${service#logstash_}
		compatibilityTable+="| $version"

		metrics=$(curl -s "http://$service:9600/_node/stats")

		metric1Available=$(jq -r '.Metric1' <<< "$metrics")
		metric2Available=$(jq -r '.Metric2' <<< "$metrics")
		metric3Available=$(jq -r '.Metric3' <<< "$metrics")

		compatibilityTable+=" | $metric1Available | $metric2Available | $metric3Available |\n"
	fi
done

echo -e "$compatibilityTable" > COMPATIBILITY.md
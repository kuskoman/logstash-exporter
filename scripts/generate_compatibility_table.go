#!/bin/bash

compatibilityTable="| Logstash Version | Metric 1 | Metric 2 | Metric 3 |\n|------------------|----------|----------|----------|\n"

for service in $(docker service ls --format "{{.Name}}"); do
	if [[ $service == logstash_* ]]; then
		version=${service#logstash_}
		compatibilityTable+="| $version"

		metrics=$(curl -s "http://$service:9600/_node/stats" | jq -c '.data[] | select(.metric | startswith("logstash"))')

		for metric in $metrics; do
			metricName=$(echo "$metric" | jq -r '.metric')
			metricAvailable=$(jq -r ".$metricName" <<< "$metrics")
			compatibilityTable+=" | $metricAvailable"
		done

		compatibilityTable+=" |\n"
	fi
done

echo -e "$compatibilityTable" > COMPATIBILITY.md
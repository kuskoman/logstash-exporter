#!/bin/bash

compatibilityTable="| Logstash Version "
metrics=$(curl -s "http://localhost:9600/_node/stats" | jq -c '.data[] | select(.metric | startswith("logstash"))')
for metric in $metrics; do
	metricName=$(echo "$metric" | jq -r '.metric')
	compatibilityTable+="| $metricName "
done
compatibilityTable+="\n|------------------"
for metric in $metrics; do
	compatibilityTable+="|----------"
done
compatibilityTable+="\n"

for service in $(grep 'logstash_' docker-compose-compatibility.yml | cut -d ':' -f 1 | tr -d ' '); do
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
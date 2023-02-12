package config

import "github.com/kuskoman/logstash-exporter/helpers"

var (
	LogstashUrl = helpers.GetEnvWithDefault("LOGSTASH_URL", "http://localhost:9600")
)

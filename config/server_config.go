package config

import "github.com/kuskoman/logstash-exporter/helpers"

var (
	Port = helpers.GetEnvWithDefault("PORT", "9198")
)

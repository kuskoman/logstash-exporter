package config

var (
	LogstashUrl = getEnvWithDefault("LOGSTASH_URL", "http://localhost:9600")
)

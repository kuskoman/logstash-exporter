package config

var (
	// LogstashUrl is the URL of the Logstash instance to be monitored.
	// Defaults to http://localhost:9600
	// Can be overridden by setting the LOGSTASH_URL environment variable
	LogstashUrl = getEnvWithDefault("LOGSTASH_URL", "http://localhost:9600")
)

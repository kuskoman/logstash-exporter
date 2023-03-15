package config

var (
	Port = getEnvWithDefault("PORT", "9198")
	Host = getEnvWithDefault("HOST", "")
)

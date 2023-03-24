package config

var (
	// Port is the port the exporter will listen on.
	// Defaults to 9198
	// Can be overridden by setting the PORT environment variable
	Port = getEnvWithDefault("PORT", "9198")

	// Host is the host the exporter will listen on.
	// Defaults to an empty string, which will listen on all interfaces
	// Can be overridden by setting the HOST environment variable
	// For windows, use "localhost", because an empty string will not work
	// with the default windows firewall configuration.
	// Alternatively you can change the firewall configuration to allow
	// connections to the port from all interfaces.
	Host = getEnvWithDefault("HOST", "")
)

package config

var (
	// SSL determines if the exporter should use HTTPS instead of HTTP
	// Defaults to "FALSE"
	// Can be overridden by setting the SSL environment variable
	EnableSSL = getEnvWithDefault("ENABLE_SSL", "FALSE")

	// SSL_CERT_FILE_PATH specifies the file path to the SSL certificate file
	// Must be set if SSL is "TRUE"
	// Can be overridden by setting the SSL_CERT_FILE_PATH environment variable
	SSLCertFilePath = getEnvWithDefault("SSL_CERT_FILE_PATH","")

	// SSL_KEY_FILE_PATH specifies the file path to the SSL private key file
	// Must be set if SSL is "TRUE"
	// Can be overridden by setting the SSL_KEY_FILE_PATH environment variable
	SSLKeyFilePath = getEnvWithDefault("SSL_KEY_FILE_PATH","")

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

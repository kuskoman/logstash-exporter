# TLS Configuration Guide for Logstash Exporter

This document provides information about the TLS configuration capabilities in Logstash Exporter.

## Overview

Logstash Exporter supports TLS in two key areas:

1. **Server TLS**: Securing the HTTP server that Prometheus scrapes (issue #393)
2. **Client TLS**: Connection to Logstash instances with custom CA certificates (issue #359)

The configuration format is inspired by the Prometheus exporter-toolkit, but implemented independently.

## Server TLS Configuration

### Legacy Format (still supported)

The basic TLS configuration is available using these fields:

```yaml
server:
  enableSSL: true
  certFile: /path/to/cert.pem
  keyFile: /path/to/key.pem
```

### Advanced Format (recommended)

```yaml
server:
  tls_server_config:
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
    # Optional fields below
    min_version: TLS12
    client_auth_type: NoClientCert
    client_ca_file: /path/to/ca.pem
    prefer_server_cipher_suites: true
```

### Basic Authentication

You can secure the metrics endpoint with basic authentication:

```yaml
server:
  basic_auth:
    users:
      prometheus: secure_password
      admin: admin_password
    # Or use a users file
    users_file: /path/to/users_file
```

### Connection Limits and Timeouts

```yaml
server:
  read_timeout_seconds: 30
  write_timeout_seconds: 30
  max_connections: 512
```

## Client TLS Configuration (for Logstash connections)

To configure TLS for connections to Logstash instances:

```yaml
logstash:
  instances:
    - url: https://logstash:9600
      tls_config:
        # Path to custom CA certificate (solves issue #359)
        ca_file: /path/to/ca.pem
        # Override hostname verification
        server_name: logstash.internal
        # Disable certificate verification (not recommended for production)
        insecure_skip_verify: false
```

### Basic Authentication for Logstash

```yaml
logstash:
  instances:
    - url: https://logstash:9600
      basic_auth:
        username: logstash_user
        password: secure_password
        # Or use a password file
        password_file: /path/to/password_file
```

## SSL/TLS Version Support

The following TLS versions are supported:

- `TLS10` (not recommended)
- `TLS11` (not recommended)
- `TLS12` (recommended minimum)
- `TLS13` (recommended)

## Client Authentication Types

For the `client_auth_type` field in server configuration:

- `NoClientCert`: No client certificate is required (default)
- `RequestClientCert`: The server requests a client certificate but doesn't require it
- `RequireAnyClientCert`: The server requires a client certificate but doesn't verify it
- `VerifyClientCertIfGiven`: The server verifies a client certificate if provided
- `RequireAndVerifyClientCert`: The server requires and verifies a client certificate

## Relationship to Prometheus exporter-toolkit

Our configuration format is designed to be similar to the Prometheus exporter-toolkit, but this is an independent implementation. The exporter-toolkit is still considered experimental, so we've implemented our own solution to ensure stability.

In the future, once the exporter-toolkit is considered stable, we may consider adopting it directly.

## Security Recommendations

1. Always use TLS 1.2 or higher (`min_version: TLS12`)
2. Use strong certificates (2048+ bit RSA or ECDSA)
3. Keep certificate authority files in secure locations with proper permissions
4. Rotate certificates before they expire
5. Use password files rather than embedding passwords in the configuration when possible

## Troubleshooting

### Common Issues

1. **Certificate not found**: Ensure the paths to certificate files are correct and accessible
2. **Certificate verification failed**: Check if the CA certificate is correct or consider using `insecure_skip_verify: true` for testing. Make sure CA certificates are installed in the system trust store.
3. **TLS handshake errors**: Ensure compatible TLS versions between client and server

### Debugging

Enable debug logging to see more details about TLS connections:

```yaml
logging:
  level: debug
  format: text
```

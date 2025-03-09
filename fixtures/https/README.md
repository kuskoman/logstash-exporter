# TLS Certificate Generation for Testing

This directory contains sample certificates for HTTPS testing. **DO NOT USE THESE IN PRODUCTION.**

## Files in this Directory

- `server.crt`, `server.key` - TLS certificate and private key for the HTTP server
- `ca.crt`, `ca.key` - CA certificate and key used to sign the server certificate
- `server.csr`, `ca.srl` - Intermediate files created during certificate generation
- `server.cnf` - OpenSSL configuration used for certificate generation
- `config.yml` - Example configuration using TLS

At minimum, the exporter only needs `server.crt` and `server.key` for HTTPS operation.

## Generate Self-Signed Certificates

To generate your own certificates for testing:

### 1. Create a self-signed CA certificate:
```bash
# Generate CA private key and certificate
openssl req -x509 -newkey rsa:2048 -nodes -keyout ca.key -out ca.crt -days 365 -subj "/CN=Logstash Exporter Test CA"
```

### 2. Create a server certificate signed by your CA:
```bash
# Create server private key
openssl genrsa -out server.key 2048

# Create configuration file for certificate generation
cat > server.cnf << EOF
[req]
default_md = sha256
prompt = no
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]
CN = localhost

[v3_req]
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:localhost
EOF

# Create certificate signing request with the configuration
openssl req -new -key server.key -out server.csr -config server.cnf

# Create server certificate signed by CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extensions v3_req -extfile server.cnf
```

## Configuration

### Using TLS in the exporter server:
```yaml
server:
  tls_server_config:
    cert_file: "/path/to/server.crt"
    key_file: "/path/to/server.key"
```

### Using TLS when connecting to Logstash:
```yaml
logstash:
  instances:
    - url: "https://logstash:9600"
      tls_config:
        ca_file: "/path/to/ca.crt"    # Custom CA certificate (if needed)
        server_name: "logstash.internal"  # Override hostname verification
        insecure_skip_verify: false    # Verify certificates (default)
    - url: "https://other-logstash:9600"
      tls_config:
        insecure_skip_verify: true    # Skip certificate verification (for self-signed certs)
```

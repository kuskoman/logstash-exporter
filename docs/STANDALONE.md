# Logstash Exporter Standalone Deployment Guide

This guide provides instructions for deploying Logstash Exporter as a standalone application on a Linux system using systemd.

## Prerequisites

- A Linux server with systemd
- Access to download the Logstash Exporter binary
- Administrative privileges to create and manage systemd services

## Installation Steps

### 1. Download the Binary

First, download the appropriate binary for your system from the [GitHub releases page](https://github.com/kuskoman/logstash-exporter/releases).

```bash
# Set the version and OS variables
VERSION="2.0.0"  # Replace with the version you want to install
OS="linux"       # Options: linux, darwin (macOS), windows

# Download the binary and checksum
wget "https://github.com/kuskoman/logstash-exporter/releases/download/v${VERSION}/logstash-exporter-${OS}"
wget "https://github.com/kuskoman/logstash-exporter/releases/download/v${VERSION}/logstash-exporter-${OS}.sha256"

# Verify the checksum
sha256sum -c logstash-exporter-${OS}.sha256

# Make the binary executable
chmod +x logstash-exporter-${OS}

# Move to /usr/local/bin for system-wide availability
sudo mv logstash-exporter-${OS} /usr/local/bin/logstash-exporter
```

### 2. Create a Configuration File

Create a configuration file for the Logstash Exporter. The default location is `config.yml` in the current directory, but you can specify a different location with the `-config` flag.

```bash
# Create a directory for the configuration
sudo mkdir -p /etc/logstash-exporter

# Create a configuration file
sudo tee /etc/logstash-exporter/config.yml > /dev/null << 'EOF'
logstash:
  instances:
    - url: "http://localhost:9600"  # Replace with your Logstash URL
  timeout: 5s
server:
  host: "0.0.0.0"  # Listen on all interfaces
  port: 9198       # Default port
logging:
  level: "info"    # Options: debug, info, warn, error
EOF
```

### 3. Create a Systemd Service

Create a systemd service file to manage the Logstash Exporter:

```bash
sudo tee /etc/systemd/system/logstash-exporter.service > /dev/null << 'EOF'
[Unit]
Description=Logstash Exporter
Documentation=https://github.com/kuskoman/logstash-exporter
After=network.target

[Service]
Type=simple
User=logstash-exporter
Group=logstash-exporter
ExecStart=/usr/local/bin/logstash-exporter -config /etc/logstash-exporter/config.yml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536
NoNewPrivileges=true

# Hardening
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/var/log/logstash-exporter
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
```

### 4. Create a Dedicated User (Optional but Recommended)

For security reasons, it's best to run the Logstash Exporter as a non-privileged user:

```bash
# Create user and group
sudo useradd --system --shell /bin/false --home-dir /nonexistent logstash-exporter

# Create log directory if needed
sudo mkdir -p /var/log/logstash-exporter
sudo chown logstash-exporter:logstash-exporter /var/log/logstash-exporter
```

### 5. Start and Enable the Service

```bash
# Reload systemd to read the new service file
sudo systemctl daemon-reload

# Start the service
sudo systemctl start logstash-exporter

# Enable the service to start on boot
sudo systemctl enable logstash-exporter

# Check the status
sudo systemctl status logstash-exporter
```

### 6. Verify the Installation

You can verify that the Logstash Exporter is running correctly by checking its metrics endpoint:

```bash
curl http://localhost:9198/metrics
```

You should see Prometheus metrics in the response.

## Configuring Prometheus

Add the following configuration to your Prometheus config to scrape metrics from the Logstash Exporter:

```yaml
scrape_configs:
  - job_name: 'logstash'
    static_configs:
      - targets: ['localhost:9198']
```

## Troubleshooting

### Check Logs

If the exporter is not working correctly, check the systemd logs:

```bash
sudo journalctl -u logstash-exporter -f
```

### Verify Connectivity

Ensure that the exporter can reach your Logstash instances:

```bash
curl http://localhost:9600
```

### Check for Firewall Issues

Make sure that port 9198 is open on your firewall:

```bash
sudo ufw status
# If using ufw, allow the port if needed
sudo ufw allow 9198/tcp
```

## Advanced Configuration

### Multiple Logstash Instances

You can monitor multiple Logstash instances by adding them to your configuration file:

```yaml
logstash:
  instances:
    - url: "http://logstash1:9600"
      name: "logstash1"  # Optional custom name
    - url: "http://logstash2:9600"
      name: "logstash2"
  timeout: 5s
```

### Custom Ports

If you need to run the exporter on a different port:

```yaml
server:
  host: "0.0.0.0"
  port: 8080  # Custom port
```

Update your systemd service if you change the port to ensure the correct ports are used.

## Upgrading

To upgrade the Logstash Exporter:

1. Download the new version
2. Verify the checksum
3. Stop the service: `sudo systemctl stop logstash-exporter`
4. Replace the binary: `sudo mv logstash-exporter-${OS} /usr/local/bin/logstash-exporter`
5. Start the service: `sudo systemctl start logstash-exporter`

## Uninstallation

To completely remove the Logstash Exporter:

```bash
# Stop and disable the service
sudo systemctl stop logstash-exporter
sudo systemctl disable logstash-exporter

# Remove the service file
sudo rm /etc/systemd/system/logstash-exporter.service
sudo systemctl daemon-reload

# Remove the binary
sudo rm /usr/local/bin/logstash-exporter

# Remove the configuration
sudo rm -rf /etc/logstash-exporter

# Optionally remove the user
sudo userdel logstash-exporter
sudo rm -rf /var/log/logstash-exporter
```

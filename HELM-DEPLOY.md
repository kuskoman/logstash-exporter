# Logstash Exporter Helm Deployment Guide

This guide provides detailed instructions for deploying Logstash Exporter on Kubernetes using Helm.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- Access to a Logstash instance

## Installation

### Basic Installation

1. Clone the repository or download the chart:

```bash
git clone https://github.com/kuskoman/logstash-exporter.git
cd logstash-exporter
```

2. Install the chart with the release name "logstash-exporter":

```bash
helm install logstash-exporter ./chart \
  --set logstash.instances.url[0]="http://your-logstash-service:9600"
```

3. Verify the deployment:

```bash
kubectl get pods -l app.kubernetes.io/name=logstash-exporter
```

### Installation with Custom Values

Create a values.yaml file:

```yaml
logstash:
  instances:
    url:
      - "http://logstash-service:9600"
      - "http://logstash-service-2:9600"
  server:
    port: 9198
  logging:
    level: "info"

deployment:
  resources:
    limits:
      cpu: 100m
      memory: 128Mi
    requests:
      cpu: 50m
      memory: 64Mi

service:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9198"
```

Install with the custom values:

```bash
helm install logstash-exporter ./chart -f values.yaml
```

## Configuring Prometheus Integration

### Standard Prometheus Configuration

Add the following to your Prometheus configuration to scrape metrics from Logstash Exporter:

```yaml
scrape_configs:
  - job_name: 'logstash'
    kubernetes_sd_configs:
      - role: service
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        regex: logstash-exporter
        action: keep
```

### Prometheus Operator ServiceMonitor

If you're using the Prometheus Operator, create a ServiceMonitor resource:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: logstash-exporter
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: logstash-exporter
  endpoints:
  - port: http
    interval: 30s
  namespaceSelector:
    matchNames:
    - default  # Change to the namespace where logstash-exporter is installed
```

Apply the ServiceMonitor:

```bash
kubectl apply -f servicemonitor.yaml
```

## Advanced Configuration

### Monitoring Multiple Logstash Instances

For monitoring multiple Logstash instances across different environments:

```yaml
logstash:
  instances:
    url:
      - "http://logstash-prod:9600"
      - "http://logstash-staging:9600"
      - "http://logstash-dev:9600"
```

Each instance will be monitored independently, with metrics labeled accordingly.

## Upgrading

To upgrade your Logstash Exporter deployment:

```bash
helm upgrade logstash-exporter ./chart -f values.yaml
```

## Troubleshooting

### Common Issues

1. **Pod fails to start**: Check logs and events
   ```bash
   kubectl describe pod -l app.kubernetes.io/name=logstash-exporter
   kubectl logs -l app.kubernetes.io/name=logstash-exporter
   ```

2. **Cannot connect to Logstash**: Verify network connectivity
   ```bash
   # Get a shell in the exporter pod
   kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=logstash-exporter -o name | head -n 1) -- sh

   # Test connection to Logstash
   wget -O- http://logstash-service:9600 || echo "Connection failed"
   ```

3. **Prometheus not scraping metrics**: Check service discovery
   ```bash
   # Check if service is properly labeled
   kubectl get service -l app.kubernetes.io/name=logstash-exporter -o yaml

   # If using ServiceMonitor, check its status
   kubectl get servicemonitor logstash-exporter -n monitoring -o yaml
   ```

### Increasing Log Verbosity

To see more detailed logs, set the logging level to "debug":

```yaml
logstash:
  logging:
    level: "debug"
```

Apply the change:

```bash
helm upgrade logstash-exporter ./chart -f values.yaml
```

### Checking Metrics Manually

To check if metrics are being exposed correctly:

```bash
# Forward the service port to your local machine
kubectl port-forward service/logstash-exporter 9198:9198

# Check metrics endpoint
curl http://localhost:9198/metrics
```

## Reference

For the complete list of configuration parameters, see the [Helm Chart README](./chart/README.md), which contains the auto-generated documentation for all available values.

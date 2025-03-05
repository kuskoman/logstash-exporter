# Logstash Exporter Helm Chart

This Helm chart provides a Kubernetes deployment for the Logstash Exporter, enabling you to monitor your Logstash instances using Prometheus.

## Quick Start

### Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- Access to a Logstash instance

### Installation

1. Add the repository (if available) or use the chart directly:

```bash
# Clone the repository if using the chart directly
git clone https://github.com/kuskoman/logstash-exporter.git
cd logstash-exporter

# Install the chart with the release name "logstash-exporter"
helm install logstash-exporter ./chart \
  --set logstash.instances.url[0]="http://your-logstash-service:9600"
```

2. Verify the deployment:

```bash
kubectl get pods -l app.kubernetes.io/name=logstash-exporter
```

## Usage

By default, the Logstash Exporter will be deployed as a single pod that exposes metrics on port 9198. You can configure Prometheus to scrape these metrics.

### Example Prometheus ServiceMonitor (if using Prometheus Operator)

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

### Example values.yaml for custom configuration

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

## Configuration

### Logstash Configuration Parameters

#### Logstash Instances

| Name                     | Description  | Value                      |
| ------------------------ | ------------ | -------------------------- |
| `logstash.instances.url` | Logstash URL | `["http://logstash:9600"]` |

#### Logstash Exporter Server

| Name                   | Description                           | Value     |
| ---------------------- | ------------------------------------- | --------- |
| `logstash.server.host` | Host for the logstash exporter server | `0.0.0.0` |
| `logstash.server.port` | Port for the logstash exporter server | `9198`    |

#### Logging Configuration

| Name                     | Description                     | Value  |
| ------------------------ | ------------------------------- | ------ |
| `logstash.logging.level` | Logstash exporter logging level | `info` |

### Custom Configuration

If you need a completely custom configuration, you can override the default settings:

| Name                   | Description                 | Value                                                  |
| ---------------------- | --------------------------- | ------------------------------------------------------ |
| `customConfig.enabled` | Enable custom configuration | `false`                                                |
| `customConfig.config`  | Custom configuration        | `logstash:
  instances:
    - "http://logstash:9600"
` |

### Image Settings

| Name               | Description                                  | Value                        |
| ------------------ | -------------------------------------------- | ---------------------------- |
| `image.repository` | Image repository                             | `kuskoman/logstash-exporter` |
| `image.tag`        | Image tag, if not set the appVersion is used | `""`                         |
| `image.pullPolicy` | Image pull policy                            | `IfNotPresent`               |
| `fullnameOverride` | Override the fullname of the chart           | `""`                         |

### Deployment Settings

| Name                                  | Description                                                            | Value    |
| ------------------------------------- | ---------------------------------------------------------------------- | -------- |
| `deployment.replicas`                 | Number of replicas for the deployment                                  | `1`      |
| `deployment.restartPolicy`            | Restart policy for the deployment.                                     | `Always` |
| `deployment.annotations`              | Additional deployment annotations                                      | `{}`     |
| `deployment.labels`                   | Additional deployment labels                                           | `{}`     |
| `deployment.pullSecret`               | Kubernetes secret for pulling the image                                | `[]`     |
| `deployment.resources`                | Resource requests and limits                                           | `{}`     |
| `deployment.nodeSelector`             | Node selector for the deployment                                       | `{}`     |
| `deployment.tolerations`              | Tolerations for the deployment                                         | `[]`     |
| `deployment.podAnnotations`           | Additional pod annotations                                             | `{}`     |
| `deployment.podLabels`                | Additional pod labels                                                  | `{}`     |
| `deployment.affinity`                 | Affinity for the deployment                                            | `{}`     |
| `deployment.env`                      | Additional environment variables                                       | `{}`     |
| `deployment.envFrom`                  | Additional environment variables from config maps or secrets           | `[]`     |
| `deployment.priorityClassName`        | Priority class name for the deployment                                 | `""`     |
| `deployment.dnsConfig`                | DNS configuration for the deployment                                   | `{}`     |
| `deployment.securityContext`          | Security context for the deployment                                    | `{}`     |
| `deployment.podSecurityContext`       | Security context for the deployment that only applies to the pod       | `{}`     |
| `deployment.containerSecurityContext` | Security context for the deployment that only applies to the container | `{}`     |

### Health Checks

#### Liveness Probe

| Name                                           | Description                          | Value     |
| ---------------------------------------------- | ------------------------------------ | --------- |
| `deployment.livenessProbe.httpGet.path`        | Path for liveness probe              | `/health` |
| `deployment.livenessProbe.httpGet.port`        | Port for liveness probe              | `9198`    |
| `deployment.livenessProbe.initialDelaySeconds` | Initial delay for liveness probe     | `30`      |
| `deployment.livenessProbe.periodSeconds`       | Period for liveness probe            | `10`      |
| `deployment.livenessProbe.timeoutSeconds`      | Timeout for liveness probe           | `5`       |
| `deployment.livenessProbe.successThreshold`    | Success threshold for liveness probe | `1`       |
| `deployment.livenessProbe.failureThreshold`    | Failure threshold for liveness probe | `3`       |
| `deployment.readinessProbe`                    | Readiness probe configuration        | `{}`      |

#### Rolling Update

| Name                                      | Description                            | Value |
| ----------------------------------------- | -------------------------------------- | ----- |
| `deployment.rollingUpdate.maxSurge`       | Maximum surge for rolling update       | `1`   |
| `deployment.rollingUpdate.maxUnavailable` | Maximum unavailable for rolling update | `0`   |

### Service Settings

| Name                  | Description                    | Value       |
| --------------------- | ------------------------------ | ----------- |
| `service.type`        | Service type                   | `ClusterIP` |
| `service.port`        | Service port                   | `9198`      |
| `service.annotations` | Additional service annotations | `{}`        |
| `service.labels`      | Additional service labels      | `{}`        |

### ServiceAccount Settings

| Name                         | Description                            | Value   |
| ---------------------------- | -------------------------------------- | ------- |
| `serviceAccount.enabled`     | Enable service account creation        | `false` |
| `serviceAccount.create`      | Create service account                 | `false` |
| `serviceAccount.name`        | Service account name                   | `""`    |
| `serviceAccount.annotations` | Additional service account annotations | `{}`    |

## Advanced Usage

### Monitoring Multiple Logstash Instances

For monitoring multiple Logstash instances, set the URLs in your values.yaml:

```yaml
logstash:
  instances:
    url:
      - "http://logstash-1:9600"
      - "http://logstash-2:9600"
      - "http://logstash-3:9600"
```

### Security Best Practices

For production deployments, consider:

1. Setting resource limits
2. Using a dedicated service account with minimal permissions
3. Configuring security contexts to run as non-root
4. Implementing network policies to restrict access

```yaml
deployment:
  resources:
    limits:
      cpu: 200m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi
  
  podSecurityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000

serviceAccount:
  enabled: true
  create: true
```

## Troubleshooting

### Common Issues

1. **Cannot connect to Logstash**: Verify network connectivity between the exporter and Logstash instances
2. **No metrics available**: Check if the service is running and if Prometheus is configured to scrape it
3. **Pod crashes**: Check the logs with `kubectl logs -l app.kubernetes.io/name=logstash-exporter`

### Increasing Log Verbosity

To see more detailed logs, set the logging level to "debug":

```yaml
logstash:
  logging:
    level: "debug"
```

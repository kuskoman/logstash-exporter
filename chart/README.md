# Helm chart

## Parameters

### Logstash configuration

| Name           | Description           | Value                  |
| -------------- | --------------------- | ---------------------- |
| `logstash.url` | Logstash instance URL | `http://logstash:9600` |

### Image settings

| Name               | Description                                  | Value                        |
| ------------------ | -------------------------------------------- | ---------------------------- |
| `image.repository` | Image repository                             | `kuskoman/logstash-exporter` |
| `image.tag`        | Image tag, if not set the appVersion is used | `""`                         |
| `image.pullPolicy` | Image pull policy                            | `IfNotPresent`               |
| `fullnameOverride` | Override the fullname of the chart           | `""`                         |

### Deployment settings

| Name                           | Description                                                  | Value    |
| ------------------------------ | ------------------------------------------------------------ | -------- |
| `deployment.replicas`          | Number of replicas for the deployment                        | `1`      |
| `deployment.restartPolicy`     | Restart policy for the deployment.                           | `Always` |
| `deployment.annotations`       | Additional deployment annotations                            | `{}`     |
| `deployment.labels`            | Additional deployment labels                                 | `{}`     |
| `deployment.pullSecret`        | Kubernetes secret for pulling the image                      | `[]`     |
| `deployment.resources`         | Resource requests and limits                                 | `{}`     |
| `deployment.nodeSelector`      | Node selector for the deployment                             | `{}`     |
| `deployment.tolerations`       | Tolerations for the deployment                               | `[]`     |
| `deployment.podAnnotations`    | Additional pod annotations                                   | `{}`     |
| `deployment.podLabels`         | Additional pod labels                                        | `{}`     |
| `deployment.affinity`          | Affinity for the deployment                                  | `{}`     |
| `deployment.env`               | Additional environment variables                             | `{}`     |
| `deployment.envFrom`           | Additional environment variables from config maps or secrets | `[]`     |
| `deployment.priorityClassName` | Priority class name for the deployment                       | `""`     |
| `deployment.dnsConfig`         | DNS configuration for the deployment                         | `{}`     |
| `deployment.securityContext`   | Security context for the deployment                          | `{}`     |

### Liveness probe settings

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

### Rolling update settings

| Name                                      | Description                            | Value |
| ----------------------------------------- | -------------------------------------- | ----- |
| `deployment.rollingUpdate.maxSurge`       | Maximum surge for rolling update       | `1`   |
| `deployment.rollingUpdate.maxUnavailable` | Maximum unavailable for rolling update | `0`   |

### Service settings

| Name                  | Description                    | Value       |
| --------------------- | ------------------------------ | ----------- |
| `service.type`        | Service type                   | `ClusterIP` |
| `service.port`        | Service port                   | `9198`      |
| `service.annotations` | Additional service annotations | `{}`        |
| `service.labels`      | Additional service labels      | `{}`        |

### ServiceAccount settings

| Name                         | Description                            | Value   |
| ---------------------------- | -------------------------------------- | ------- |
| `serviceAccount.enabled`     | Enable service account creation        | `false` |
| `serviceAccount.create`      | Create service account                 | `false` |
| `serviceAccount.name`        | Service account name                   | `""`    |
| `serviceAccount.annotations` | Additional service account annotations | `{}`    |

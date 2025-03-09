# Helm chart

## Parameters

### Logstash configuration


### logstash.instances Logstash instances

| Name                        | Description                          | Value                   |
| --------------------------- | ------------------------------------ | ----------------------- |
| `logstash.instances[0].url` | Logstash URL for the first instance  | `http://logstash:9600`  |
| `logstash.instances[1].url` | Logstash URL for the second instance | `http://logstash2:9600` |

### Logstash exporter server configuration

| Name                   | Description                           | Value     |
| ---------------------- | ------------------------------------- | --------- |
| `logstash.server.host` | Host for the logstash exporter server | `0.0.0.0` |
| `logstash.server.port` | Port for the logstash exporter server | `9198`    |

### Logging configuration

| Name                     | Description                     | Value  |
| ------------------------ | ------------------------------- | ------ |
| `logstash.logging.level` | Logstash exporter logging level | `info` |

### Kubernetes controller configuration

| Name                             | Description                         | Value   |
| -------------------------------- | ----------------------------------- | ------- |
| `logstash.kubernetes.enabled`    | Enable Kubernetes controller        | `false` |
| `logstash.kubernetes.namespaces` | Namespaces to watch (empty for all) | `[]`    |

### Resource type monitoring configuration


### Pod monitoring configuration

| Name                                                  | Description                | Value                  |
| ----------------------------------------------------- | -------------------------- | ---------------------- |
| `logstash.kubernetes.resources.pods.enabled`          | Enable pod monitoring      | `true`                 |
| `logstash.kubernetes.resources.pods.annotationPrefix` | Prefix for pod annotations | `logstash-exporter.io` |

### Service monitoring configuration

| Name                                                      | Description                            | Value                           |
| --------------------------------------------------------- | -------------------------------------- | ------------------------------- |
| `logstash.kubernetes.resources.services.enabled`          | Enable service monitoring              | `false`                         |
| `logstash.kubernetes.resources.services.annotationPrefix` | Prefix for service annotations         | `logstash-exporter.io`          |
| `logstash.kubernetes.resyncPeriod`                        | Resync period for the controller cache | `10m`                           |
| `logstash.kubernetes.scrapeInterval`                      | Interval to scrape logstash instances  | `15s`                           |
| `logstash.kubernetes.logstashURLAnnotation`               | Annotation containing logstash URL     | `logstash-exporter.io/url`      |
| `logstash.kubernetes.logstashUsernameAnnotation`          | Annotation for logstash username       | `logstash-exporter.io/username` |
| `logstash.kubernetes.logstashPasswordAnnotation`          | Annotation for logstash password       | `logstash-exporter.io/password` |

### Custom logstash-exporter configuration

| Name                   | Description                 | Value                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| ---------------------- | --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `customConfig.enabled` | Enable custom configuration | `false`                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `customConfig.config`  | Custom configuration        | `logstash:
  instances:
    - "http://logstash:9600"
  server:
    host: "0.0.0.0"
    port: 9198
  logging:
    level: "info"
  kubernetes:
    enabled: false
    namespaces: []
    resources:
      pods:
        enabled: true
        annotationPrefix: "logstash-exporter.io"
      services:
        enabled: false
        annotationPrefix: "logstash-exporter.io"
    resyncPeriod: 10m
    scrapeInterval: 15s
    logstashURLAnnotation: "logstash-exporter.io/url"
` |

### Image settings

| Name                         | Description                                  | Value                        |
| ---------------------------- | -------------------------------------------- | ---------------------------- |
| `image.repository`           | Image repository                             | `kuskoman/logstash-exporter` |
| `image.tag`                  | Image tag, if not set the appVersion is used | `""`                         |
| `image.pullPolicy`           | Image pull policy                            | `IfNotPresent`               |
| `image.controllerRepository` | Image repository for the controller          | `""`                         |
| `fullnameOverride`           | Override the fullname of the chart           | `""`                         |

### Deployment settings

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

### RBAC settings for Kubernetes controller

| Name                      | Description                               | Value                    |
| ------------------------- | ----------------------------------------- | ------------------------ |
| `rbac.create`             | Create RBAC resources                     | `false`                  |
| `rbac.rules[0].apiGroups` | API groups the rule applies to            | `[""]`                   |
| `rbac.rules[0].resources` | Kubernetes resources the rule applies to  | `["pods","services"]`    |
| `rbac.rules[0].verbs`     | Allowed verbs for the specified resources | `["get","list","watch"]` |

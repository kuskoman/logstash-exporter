# logstash-exporter

![Version: 2.0.0](https://img.shields.io/badge/Version-2.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.0.0](https://img.shields.io/badge/AppVersion-2.0.0-informational?style=flat-square)

Prometheus exporter for Logstash written in Go

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| customConfig.config | string | `"logstash:\n  instances:\n    - \"http://logstash:9600\"\n  server:\n    host: \"0.0.0.0\"\n    port: 9198\n  logging:\n    level: \"info\"\n  kubernetes:\n    enabled: false\n    namespaces: []\n    resources:\n      pods:\n        enabled: true\n        annotationPrefix: \"logstash-exporter.io\"\n      services:\n        enabled: false\n        annotationPrefix: \"logstash-exporter.io\"\n    resyncPeriod: 10m\n    scrapeInterval: 15s\n    logstashURLAnnotation: \"logstash-exporter.io/url\"\n"` |  |
| customConfig.enabled | bool | `false` |  |
| deployment.affinity | object | `{}` |  |
| deployment.annotations | object | `{}` |  |
| deployment.containerSecurityContext | object | `{}` |  |
| deployment.dnsConfig | object | `{}` |  |
| deployment.env | object | `{}` |  |
| deployment.envFrom | list | `[]` |  |
| deployment.labels | object | `{}` |  |
| deployment.livenessProbe.failureThreshold | int | `3` |  |
| deployment.livenessProbe.httpGet.path | string | `"/health"` |  |
| deployment.livenessProbe.httpGet.port | int | `9198` |  |
| deployment.livenessProbe.initialDelaySeconds | int | `30` |  |
| deployment.livenessProbe.periodSeconds | int | `10` |  |
| deployment.livenessProbe.successThreshold | int | `1` |  |
| deployment.livenessProbe.timeoutSeconds | int | `5` |  |
| deployment.nodeSelector | object | `{}` |  |
| deployment.podAnnotations | object | `{}` |  |
| deployment.podLabels | object | `{}` |  |
| deployment.podSecurityContext | object | `{}` |  |
| deployment.priorityClassName | string | `""` |  |
| deployment.pullSecret | list | `[]` |  |
| deployment.readinessProbe | object | `{}` |  |
| deployment.replicas | int | `1` |  |
| deployment.resources | object | `{}` |  |
| deployment.restartPolicy | string | `"Always"` |  |
| deployment.rollingUpdate.maxSurge | int | `1` |  |
| deployment.rollingUpdate.maxUnavailable | int | `0` |  |
| deployment.securityContext | object | `{}` |  |
| deployment.tolerations | list | `[]` |  |
| fullnameOverride | string | `""` |  |
| image.controllerRepository | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"kuskoman/logstash-exporter"` |  |
| image.tag | string | `""` |  |
| logstash.instances[0].url | string | `"http://logstash:9600"` |  |
| logstash.instances[1].url | string | `"http://logstash2:9600"` |  |
| logstash.kubernetes.enabled | bool | `false` |  |
| logstash.kubernetes.logstashPasswordAnnotation | string | `"logstash-exporter.io/password"` |  |
| logstash.kubernetes.logstashURLAnnotation | string | `"logstash-exporter.io/url"` |  |
| logstash.kubernetes.logstashUsernameAnnotation | string | `"logstash-exporter.io/username"` |  |
| logstash.kubernetes.namespaces | list | `[]` |  |
| logstash.kubernetes.resources.pods.annotationPrefix | string | `"logstash-exporter.io"` |  |
| logstash.kubernetes.resources.pods.enabled | bool | `true` |  |
| logstash.kubernetes.resources.services.annotationPrefix | string | `"logstash-exporter.io"` |  |
| logstash.kubernetes.resources.services.enabled | bool | `false` |  |
| logstash.kubernetes.resyncPeriod | string | `"10m"` |  |
| logstash.kubernetes.scrapeInterval | string | `"15s"` |  |
| logstash.logging.level | string | `"info"` |  |
| logstash.server.host | string | `"0.0.0.0"` |  |
| logstash.server.port | int | `9198` |  |
| rbac.create | bool | `false` |  |
| rbac.rules[0].apiGroups[0] | string | `""` |  |
| rbac.rules[0].resources[0] | string | `"pods"` |  |
| rbac.rules[0].resources[1] | string | `"services"` |  |
| rbac.rules[0].verbs[0] | string | `"get"` |  |
| rbac.rules[0].verbs[1] | string | `"list"` |  |
| rbac.rules[0].verbs[2] | string | `"watch"` |  |
| service.annotations | object | `{}` |  |
| service.labels | object | `{}` |  |
| service.port | int | `9198` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `false` |  |
| serviceAccount.enabled | bool | `false` |  |
| serviceAccount.name | string | `""` |  |


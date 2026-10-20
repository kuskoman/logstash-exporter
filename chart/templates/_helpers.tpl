{{/*
Create a default fully qualified app name.
If release name contains chart name it will be used as a full name.
*/}}
{{- define "logstash-exporter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "logstash-exporter.name" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Service account name. Uses serviceAccount.name when set, otherwise the fullname
when the chart creates the service account, or "default" when it does not.
*/}}
{{- define "logstash-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "logstash-exporter.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
logstash-exporter.imageTag is a named template that returns the image tag.
It checks if .Values.image.tag is provided, and if not, it returns a tag with "v" prefix followed by the app version from the Chart.yaml.
*/}}
{{- define "logstash-exporter.imageTag" -}}
{{- if .Values.image.tag -}}
{{- .Values.image.tag -}}
{{- else -}}
{{- printf "v%s" .Chart.AppVersion -}}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "logstash-exporter.labels" -}}
app: {{ include "logstash-exporter.name" . }}
release: {{ .Release.Name }}
{{- end -}}

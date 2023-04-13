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
By default we want service account name to be the same as fullname
but this may change in the future, so for easier usage this is extracted
to a separate template.
*/}}
{{- define "logstash-exporter.serviceAccountName" -}}
{{- include "logstash-exporter.fullname" . }}
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

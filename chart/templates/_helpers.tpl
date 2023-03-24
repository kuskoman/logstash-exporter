{{- define "logstash-exporter.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "logstash-exporter.name" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "logstash-exporter.serviceAccountName" -}}
{{/*
By default we want service account name to be the same as fullname
but this may change in the future, so for easier usage this is extracted
to a separate template.
*/}}
{{- include "logstash-exporter.fullname" . }}
{{- end -}}

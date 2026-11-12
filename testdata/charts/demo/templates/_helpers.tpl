{{/*
Expand the name of the chart.
*/}}
{{- define "demo.name" -}}
{{- default .Chart.Name .Values.nameOverride|trunc 63|trimSuffix "-" }}
{{- end }}

{{- define "demo.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride|trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name:=default .Chart.Name .Values.nameOverride }}
{{- printf "%s-%s" .Release.Name $name|trunc 63|trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "demo.labels" -}}
app.kubernetes.io/name: {{include "demo.name" .}}
app.kubernetes.io/instance: {{ .Release.Name}}
{{- end }}

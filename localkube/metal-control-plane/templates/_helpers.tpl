{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "metal-api.fullname" -}}
{{- printf "%s-%s" .Release.Name "api" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "rethinkdb.fullname" -}}
{{- printf "%s-%s" .Release.Name "rethinkdb" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "rethinkdb-data-volume.fullname" -}}
{{- printf "%s-%s" .Release.Name "rethinkdb-data-volume" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "nsq.fullname" -}}
{{- printf "%s-%s" .Release.Name "nsq" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "nsqd.fullname" -}}
{{- printf "%s-%s" .Release.Name "nsqd" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "nsq-lookupd.fullname" -}}
{{- printf "%s-%s" .Release.Name "nsq-lookupd" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "nsq-lookupd-headless.fullname" -}}
{{- printf "%s-%s" .Release.Name "nsq-lookupd-headless" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "nsq-admin.fullname" -}}
{{- printf "%s-%s" .Release.Name "nsq-admin" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "nsq-data-volume.fullname" -}}
{{- printf "%s-%s" .Release.Name "nsq-data-volume" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox-api-proxy.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox-api-proxy" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox-postgres.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox-postgres" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox-postgres-data-volume.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox-postgres-data-volume" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox-nginx.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox-nginx" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox-nginx-static-content.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox-nginx-static-content" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "netbox-nginx-config.fullname" -}}
{{- printf "%s-%s" .Release.Name "netbox-nginx-config" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "swagger-ui.fullname" -}}
{{- printf "%s-%s" .Release.Name "swagger-ui" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "metal-api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

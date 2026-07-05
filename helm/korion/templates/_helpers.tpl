{{/*
Copyright 2026 The Korion Authors. Licensed under the Apache License, Version 2.0.
Shared template helpers: names, labels, and derived image references.
*/}}

{{/* Base name, overridable via nameOverride. */}}
{{- define "korion.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified app name. Follows the standard Helm pattern: if the release
name already contains the chart name, don't repeat it. Overridable via
fullnameOverride.
*/}}
{{- define "korion.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "korion.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Labels shared by every object in the chart. */}}
{{- define "korion.labels" -}}
helm.sh/chart: {{ include "korion.chart" . }}
{{ include "korion.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: korion
{{- end -}}

{{- define "korion.selectorLabels" -}}
app.kubernetes.io/name: {{ include "korion.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Controller component names and labels. */}}
{{- define "korion.controller.fullname" -}}
{{- printf "%s-controller" (include "korion.fullname" .) -}}
{{- end -}}

{{- define "korion.controller.selectorLabels" -}}
{{ include "korion.selectorLabels" . }}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "korion.controller.labels" -}}
{{ include "korion.labels" . }}
app.kubernetes.io/component: controller
{{- end -}}

{{/* UI component names and labels. */}}
{{- define "korion.ui.fullname" -}}
{{- printf "%s-ui" (include "korion.fullname" .) -}}
{{- end -}}

{{- define "korion.ui.selectorLabels" -}}
{{ include "korion.selectorLabels" . }}
app.kubernetes.io/component: ui
{{- end -}}

{{- define "korion.ui.labels" -}}
{{ include "korion.labels" . }}
app.kubernetes.io/component: ui
{{- end -}}

{{/* ServiceAccount name used by the controller. */}}
{{- define "korion.serviceAccountName" -}}
{{- include "korion.controller.fullname" . -}}
{{- end -}}

{{/* Resolved image refs: image.tag falls back to the chart appVersion. */}}
{{- define "korion.controller.image" -}}
{{- printf "%s:%s" .Values.controller.image.repository (default .Chart.AppVersion .Values.controller.image.tag) -}}
{{- end -}}

{{- define "korion.ui.image" -}}
{{- printf "%s:%s" .Values.ui.image.repository (default .Chart.AppVersion .Values.ui.image.tag) -}}
{{- end -}}

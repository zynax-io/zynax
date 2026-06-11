{{/* SPDX-License-Identifier: Apache-2.0 */}}

{{/*
Fullname — preserves the Service name consumers already use in their DSNs.
The previous wrapper chart rendered "<release-name>-postgresql" (e.g.
"zynax-postgresql" for release "zynax"), so the thin chart keeps the exact
same default (ADR-026: consumer DSNs unchanged). Override via
fullnameOverride only if you also update every consumer DSN.
*/}}
{{- define "zynax-postgres.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-postgresql" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/* Common labels */}}
{{- define "zynax-postgres.labels" -}}
app.kubernetes.io/name: postgresql
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{/* Selector labels (immutable on StatefulSets — do not change) */}}
{{- define "zynax-postgres.selectorLabels" -}}
app.kubernetes.io/name: postgresql
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Name of the Secret carrying the admin + application-user passwords.
Uses auth.existingSecret when set (production); otherwise the chart-managed
Secret (dev/e2e convenience).
*/}}
{{- define "zynax-postgres.secretName" -}}
{{- if .Values.postgresql.auth.existingSecret -}}
{{- .Values.postgresql.auth.existingSecret -}}
{{- else -}}
{{- include "zynax-postgres.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "zynax.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
If .Release.Name already contains the chart name, use .Release.Name alone to
avoid duplication (e.g. "zynax-api-gateway" instead of "zynax-zynax-api-gateway").
Truncated at 63 chars (Kubernetes DNS label limit).
*/}}
{{- define "zynax.fullname" -}}
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

{{/*
Chart label value — used in helm.sh/chart annotation.
*/}}
{{- define "zynax.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels — applied to every resource's metadata.labels.
Includes app.kubernetes.io/part-of: zynax so NetworkPolicy selectors can
match across all service charts without listing each chart individually.
*/}}
{{- define "zynax.labels" -}}
helm.sh/chart: {{ include "zynax.chart" . }}
{{ include "zynax.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: zynax
{{- end }}

{{/*
Selector labels — used in spec.selector.matchLabels and spec.template.metadata.labels.
Must be stable across upgrades; do not add mutable fields here.
*/}}
{{- define "zynax.selectorLabels" -}}
app.kubernetes.io/name: {{ include "zynax.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
Respects .Values.serviceAccount.name override; falls back to fullname when
serviceAccount.create is true, or "default" when create is false.
*/}}
{{- define "zynax.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "zynax.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Pod-level security context (spec.securityContext).
Applied to every Zynax service Deployment pod spec.
Enforces: non-root UID 1001, shared fsGroup 1001, RuntimeDefault seccomp profile.
*/}}
{{- define "zynax.podSecurityContext" -}}
runAsNonRoot: true
runAsUser: 1001
fsGroup: 1001
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{/*
Container-level security context (containers[].securityContext).
Applied to every Zynax service container.
Enforces: no privilege escalation, read-only root filesystem, drop all Linux capabilities.
*/}}
{{- define "zynax.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop: ["ALL"]
{{- end }}

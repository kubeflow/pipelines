{{/*
Expand the name of the chart.
*/}}
{{- define "kubeflow-pipelines.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kubeflow-pipelines.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "kubeflow-pipelines.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubeflow-pipelines.labels" -}}
helm.sh/chart: {{ include "kubeflow-pipelines.chart" . }}
{{ include "kubeflow-pipelines.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: kubeflow
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubeflow-pipelines.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubeflow-pipelines.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{/*
ServiceAccount annotations (for IRSA and custom annotations)
*/}}
{{- define "kubeflow-pipelines.serviceaccount-annotations" -}}
{{- with .Values.serviceAccount.annotations }}
{{- toYaml . | nindent 0 }}
{{- end }}
{{- end }}

{{/*
Namespace helper
*/}}
{{- define "kubeflow-pipelines.namespace" -}}
{{- .Values.namespace.name | default "kubeflow" }}
{{- end }}

{{/*
Compute checksum for config resources
*/}}
{{- define "kubeflow-pipelines.config-checksum" -}}
checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
checksum/secret: {{ include (print $.Template.BasePath "/externalsecret.yaml") . | sha256sum }}
{{- end }}

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
Chart label value.
*/}}
{{- define "kubeflow-pipelines.chart" -}}
{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubeflow-pipelines.labels" -}}
helm.sh/chart: {{ include "kubeflow-pipelines.chart" . }}
{{ include "kubeflow-pipelines.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubeflow-pipelines.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubeflow-pipelines.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Database secret name
*/}}
{{- define "kubeflow-pipelines.dbSecretName" -}}
{{- if .Values.dbSecret.existingSecret -}}
{{ .Values.dbSecret.existingSecret }}
{{- else if eq .Values.database.type "postgresql" -}}
postgres-secret-extended
{{- else -}}
mysql-secret
{{- end -}}
{{- end }}

{{/*
Database secret username key
*/}}
{{- define "kubeflow-pipelines.dbSecretUsernameKey" -}}
{{ .Values.dbSecret.usernameKey }}
{{- end }}

{{/*
Database secret password key
*/}}
{{- define "kubeflow-pipelines.dbSecretPasswordKey" -}}
{{ .Values.dbSecret.passwordKey }}
{{- end }}

{{/*
Object store secret name
*/}}
{{- define "kubeflow-pipelines.objectStoreSecretName" -}}
{{- if .Values.objectStoreSecret.existingSecret -}}
{{ .Values.objectStoreSecret.existingSecret }}
{{- else -}}
mlpipeline-minio-artifact
{{- end -}}
{{- end }}

{{/*
DB host
*/}}
{{- define "kubeflow-pipelines.dbHost" -}}
{{- if eq .Values.database.type "postgresql" -}}
postgres
{{- else -}}
mysql
{{- end -}}
{{- end }}

{{/*
DB port
*/}}
{{- define "kubeflow-pipelines.dbPort" -}}
{{- if eq .Values.database.type "postgresql" -}}
5432
{{- else -}}
3306
{{- end -}}
{{- end }}

{{/*
DB type value for config
*/}}
{{- define "kubeflow-pipelines.dbType" -}}
{{- if eq .Values.database.type "postgresql" -}}
postgres
{{- else -}}
mysql
{{- end -}}
{{- end }}

{{/*
Namespace to watch: current namespace for single-user, empty string for multi-user
*/}}
{{- define "kubeflow-pipelines.namespaceToWatch" -}}
{{- if .Values.multiUser.enabled -}}
""
{{- else -}}
{{ .Release.Namespace }}
{{- end -}}
{{- end }}

{{/*
Cache DB driver
*/}}
{{- define "kubeflow-pipelines.cacheDbDriver" -}}
{{- if eq .Values.database.type "postgresql" -}}
pgx
{{- else -}}
mysql
{{- end -}}
{{- end }}

{{/*
Object store endpoint host
*/}}
{{- define "kubeflow-pipelines.objectStoreHost" -}}
minio-service.{{ .Release.Namespace }}
{{- end }}

{{/*
Object store endpoint port
*/}}
{{- define "kubeflow-pipelines.objectStorePort" -}}
9000
{{- end }}

{{/*
imagePullSecrets
*/}}
{{- define "kubeflow-pipelines.imagePullSecrets" -}}
{{- with .Values.global.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubeflow-pipelines.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
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

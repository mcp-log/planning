{{/*
Expand the name of the chart.
*/}}
{{- define "planning.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "planning.fullname" -}}
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
{{- define "planning.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "planning.labels" -}}
helm.sh/chart: {{ include "planning.chart" . }}
{{ include "planning.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "planning.selectorLabels" -}}
app.kubernetes.io/name: {{ include "planning.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "planning.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "planning.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Database connection string
*/}}
{{- define "planning.databaseUrl" -}}
{{- $host := .Values.database.host }}
{{- $port := .Values.database.port }}
{{- $name := .Values.database.name }}
{{- $user := .Values.database.user }}
{{- $sslMode := .Values.database.sslMode }}
{{- printf "postgres://%s:$(DATABASE_PASSWORD)@%s:%d/%s?sslmode=%s" $user $host (int $port) $name $sslMode }}
{{- end }}

{{/*
Kafka brokers
*/}}
{{- define "planning.kafkaBrokers" -}}
{{- .Values.kafka.brokers }}
{{- end }}

{{/*
Database secret name
*/}}
{{- define "planning.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "planning.fullname" . }}-db
{{- end }}
{{- end }}

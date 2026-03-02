{{/*
Chart name, truncated to 63 chars (K8s label limit).
*/}}
{{- define "storm-data.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified release name.
*/}}
{{- define "storm-data.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "storm-data.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "storm-data.name" . }}
{{- end }}

{{/*
Per-component labels. Call with (dict "ctx" $ "component" "collector").
*/}}
{{- define "storm-data.componentLabels" -}}
{{ include "storm-data.labels" .ctx }}
app.kubernetes.io/name: {{ .component }}
app.kubernetes.io/instance: {{ .ctx.Release.Name }}
{{- end }}

{{/*
Selector labels for a component. Minimal set for matchLabels.
*/}}
{{- define "storm-data.selectorLabels" -}}
app: {{ .component }}
{{- end }}

{{/*
Database connection string.
*/}}
{{- define "storm-data.databaseUrl" -}}
postgres://{{ .Values.postgres.credentials.user }}:{{ .Values.postgres.credentials.password }}@postgres:5432/{{ .Values.postgres.credentials.database }}?sslmode=disable
{{- end }}

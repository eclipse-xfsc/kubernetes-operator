{{- define "resource-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "resource-operator.fullname" -}}
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

{{- define "resource-operator.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "resource-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "resource-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "resource-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "resource-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "resource-operator.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "resource-operator.webhookServiceName" -}}
{{- printf "%s-webhook" (include "resource-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "resource-operator.certName" -}}
{{- printf "%s-serving-cert" (include "resource-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "resource-operator.certSecretName" -}}
{{- default (printf "%s-webhook-server-cert" (include "resource-operator.fullname" .) | trunc 63 | trimSuffix "-") .Values.webhook.tls.secretName -}}
{{- end -}}


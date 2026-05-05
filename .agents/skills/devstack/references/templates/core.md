# Core Helm Chart Templates

Templates for the mandatory files every devstack helm chart must have.

## Chart.yaml

```yaml
apiVersion: v2
name: <service-name>
description: <service-name> helmchart
type: application
version: 0.1.0
appVersion: 1.16.0
```

---

## values.yaml

```yaml
# Core Application Settings
app_env: dev
namespace: <service-name>
name: <service-name>
bu: platform

# Image Settings
image_base: c.rzp.io/razorpay/<service-name>
image_pull_policy: IfNotPresent

# Resource Configuration (Required)
web_requests_cpu: 50m
web_requests_memory: 200Mi
web_limits_memory: 500Mi
# NOTE: CPU limits are intentionally omitted to prevent throttling

# Deployment Settings
replicas: 1
service_port: 80
container_port: 9400

# Node Placement
node_selector: node.kubernetes.io/worker-generic
dns_policy: ClusterFirst

# Secrets
secret_name: <service-name>

# Base Pod (for persistent deployment)
base:
  replicas: 2
  node_selector: node.kubernetes.io/worker-generic-base

# Ephemeral Resource Flags
# Set both the top-level flag AND the configurator flag when enabling SQS/SNS
ephemeral_db: false
ephemeral_cache: false
ephemeral_sqs: false   # also set configurator.sqs: true
ephemeral_sns: false   # also set configurator.sns: true

# Secret cloning — when true, a label-specific secret (<secret_name>-<devstack_label>) is
# cloned from the base secret and injected with ephemeral resource credentials.
# Set to false only if you want to use the shared base secret as-is (no cloning).
# Default: true
secret_cloner_enabled: true
```

---

## deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    janitor/ttl: "{{ .Values.ttl }}"
  labels:
    bu: {{ .Values.bu }}
    name: {{ .Values.name }}-{{ .Values.devstack_label }}
    devstack_label: {{ .Values.devstack_label }}
    {{ if eq .Values.devstack_label "base" }}
    velero.io/include-in-backup: "true"
    protected: "true"
    {{ end }}
  name: {{ .Values.name }}-{{ .Values.devstack_label }}
  namespace: {{ .Values.namespace }}
spec:
  progressDeadlineSeconds: 600
  {{ if eq .Values.devstack_label "base" }}
  replicas: {{ .Values.base.replicas }}
  {{ else }}
  replicas: {{ .Values.replicas }}
  {{ end }}
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: {{ .Values.name }}-{{ .Values.devstack_label }}
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
    type: RollingUpdate
  template:
    metadata:
      annotations:
        prometheus.io/path: /metrics
        prometheus.io/port: "{{ .Values.container_port }}"
        prometheus.io/scrape: "true"
      labels:
        bu: {{ .Values.bu }}
        name: {{ .Values.name }}-{{ .Values.devstack_label }}
        devstack_label: {{ .Values.devstack_label }}
      name: {{ .Values.name }}-{{ .Values.devstack_label }}
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: name
                      operator: In
                      values:
                        - {{ .Values.name }}-{{ .Values.devstack_label }}
                topologyKey: kubernetes.io/hostname
              weight: 100
      automountServiceAccountToken: true
      containers:
        - name: web
          image: {{ .Values.image_base }}:{{ .Values.image }}
          imagePullPolicy: {{ .Values.image_pull_policy }}
          ports:
            - containerPort: {{ .Values.container_port }}
          env:
            - name: APP_ENV
              value: {{ .Values.app_env }}
            - name: JAEGER_HOSTNAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
          envFrom:
            - secretRef:
                # Use base secret when: devstack_label is "base" OR secret_cloner_enabled is false
                # Otherwise use the label-specific cloned secret
                {{- if or (eq .Values.devstack_label "base") (not .Values.secret_cloner_enabled) }}
                name: {{ .Values.secret_name }}
                {{- else }}
                name: {{ .Values.secret_name }}-{{ .Values.devstack_label }}
                {{- end }}
                optional: false
          livenessProbe:
            httpGet:
              path: /health
              port: {{ .Values.container_port }}
            initialDelaySeconds: 30
            failureThreshold: 3
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          readinessProbe:
            httpGet:
              path: /health
              port: {{ .Values.container_port }}
            initialDelaySeconds: 10
            failureThreshold: 3
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          resources:
            requests:
              cpu: {{ .Values.web_requests_cpu }}
              memory: {{ .Values.web_requests_memory }}
            limits:
              memory: {{ .Values.web_limits_memory }}
              # NOTE: CPU limits intentionally omitted to prevent throttling
      dnsPolicy: {{ .Values.dns_policy }}
      nodeSelector:
        {{ if eq .Values.devstack_label "base" }}
          {{ .Values.base.node_selector }}: ""
        {{ else }}
          {{ .Values.node_selector }}: ""
        {{ end }}
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 60
```

---

## svc.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ .Values.name }}-{{ .Values.devstack_label }}
  namespace: {{ .Values.namespace }}
  annotations:
    janitor/ttl: "{{ .Values.ttl }}"
  labels:
    devstack_label: {{ .Values.devstack_label }}
    {{ if eq .Values.devstack_label "base" }}
    velero.io/include-in-backup: "true"
    protected: "true"
    {{ end }}
spec:
  ports:
    - port: {{ .Values.service_port }}
      protocol: TCP
      targetPort: {{ .Values.container_port }}
  selector:
    name: {{ .Values.name }}-{{ .Values.devstack_label }}
  sessionAffinity: None
  type: ClusterIP
```

---

## preview-url.yaml

```yaml
{{- if ne .Values.devstack_label "base" }}
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  labels:
    devstack_label: {{ .Values.devstack_label }}
  annotations:
    janitor/ttl: "{{ .Values.ttl }}"
  name: injectheader-{{ .Values.devstack_label }}
  namespace: {{ .Values.namespace }}
spec:
  headers:
    customRequestHeaders:
      rzpctx-dev-serve-user: {{ .Values.devstack_label }}
{{- end }}
---
kind: IngressRoute
apiVersion: traefik.containo.us/v1alpha1
metadata:
  labels:
    devstack_label: {{ .Values.devstack_label }}
    {{ if eq .Values.devstack_label "base" }}
    velero.io/include-in-backup: "true"
    protected: "true"
    {{ end }}
  annotations:
    kubernetes.io/ingress.class: traefik-concierge
    janitor/ttl: "{{ .Values.ttl }}"
  name: {{ .Values.name }}-{{ .Values.devstack_label }}
  namespace: {{ .Values.namespace }}
spec:
  entryPoints:
    - http
  routes:
    - kind: Rule
      match: Host(`{{ .Values.name }}.dev.razorpay.in`) && Headers(`rzpctx-dev-serve-user`,`{{ .Values.devstack_label }}`)
      services:
        - name: '{{ .Values.name }}-{{ .Values.devstack_label }}'
          port: {{ .Values.service_port }}
    {{- if ne .Values.devstack_label "base" }}
    - kind: Rule
      match: Host(`{{ .Values.name }}-{{ .Values.devstack_label }}.dev.razorpay.in`)
      services:
        - name: '{{ .Values.name }}-{{ .Values.devstack_label }}'
          port: {{ .Values.service_port }}
      middlewares:
        - name: injectheader-{{ .Values.devstack_label }}
    {{- end }}
```

---

## NOTES.txt

```txt
*****************HURRRAAAYYYYY******************
Thank you for installing {{ .Chart.Name }}.

This installation of yours can be accessed on
URL :  https://{{ .Values.ingress }}
Header : "rzpctx-dev-serve-user": "{{ .Values.devstack_label }}"
OR

URL : https://{{ .Values.name }}-{{ .Values.devstack_label }}.dev.razorpay.in

For serving through your local code from this installation, please follow the devspace doc
PS: Also remember to run helmfile delete once you are done.
************************************************
```

---

## Helmfile Entry

```yaml
- name: <service-name>-{{ .Values.devstack_label }}
  namespace: <service-name>
  chart: ./charts/<service-name>
  values:
    - image: <commit-hash>
    - devstack_label: {{ .Values.devstack_label }}
    - ttl: {{ .Values.ttl }}
    - namespace: <service-name>
    - secret_name: <service-name>
```

---

## Connection String Reference

| Resource | Format |
|---|---|
| Service (internal) | `<service-name>-<devstack-label>.<namespace>.svc.cluster.local` |

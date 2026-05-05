# Helm Chart Templates Reference

> Templates are split into focused files. Load only what is relevant to the task.

| File | When to use |
|---|---|
| [templates/core.md](templates/core.md) | Always — Chart.yaml, values.yaml, deployment.yaml, svc.yaml, preview-url.yaml, NOTES.txt, helmfile entry |
| [templates/ephemeral-db.md](templates/ephemeral-db.md) | When `ephemeral_db: true` — db-configmap.yaml, db-configurator.yaml |
| [templates/ephemeral-cache.md](templates/ephemeral-cache.md) | When `ephemeral_cache: true` — cache-configmap.yaml, cache-configurator.yaml |
| [templates/ephemeral-sqs.md](templates/ephemeral-sqs.md) | When `ephemeral_sqs: true` — sqs-configmap.yaml, sqs-configurator.yaml |
| [templates/ephemeral-sns.md](templates/ephemeral-sns.md) | When `ephemeral_sns: true` — sns-configmap.yaml, sns-configurator.yaml |
| [templates/secret-management.md](templates/secret-management.md) | Whenever any ephemeral resource is enabled — secret-cloner.yaml, sec-updater-cm.yaml, sec-updater.yaml |

## CRITICAL: Chart Location

**ALWAYS create/update helm charts in `<kube-manifests-repo>/helmfile/charts/<service-name>/` ONLY.**

This is the ONLY valid location for helm charts in the devstack ecosystem.

## Directory Structure

```
<kube-manifests-repo>/helmfile/charts/<service-name>/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── NOTES.txt
    ├── deployment.yaml
    ├── svc.yaml
    ├── preview-url.yaml
    ├── db-configmap.yaml (optional)
    ├── db-configurator.yaml (optional)
    ├── cache-configmap.yaml (optional)
    ├── cache-configurator.yaml (optional)
    ├── sqs-configmap.yaml (optional)
    ├── sqs-configurator.yaml (optional)
    ├── sns-configmap.yaml (optional)
    ├── sns-configurator.yaml (optional)
    ├── maxwell-cdc.yaml (optional - for CDC events from ephemeral MySQL DB)
    ├── debezium-cdc.yaml (optional - for CDC events from ephemeral PostgreSQL DB)
    ├── secret-cloner.yaml (optional)
    ├── sec-updater-cm.yaml (optional)
    └── sec-updater.yaml (optional)
```

**Full Path Example:**
```
/path/to/kube-manifests/helmfile/charts/payment-service/
```

## Core Templates

### Chart.yaml

```yaml
apiVersion: v2
name: <service-name>
description: <service-name> helmchart
type: application
version: 0.1.0
appVersion: 1.16.0
```

### values.yaml (Minimal Configuration)

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

# Secret cloning — when true, clones <secret_name> into <secret_name>-<devstack_label>
# and injects ephemeral resource credentials. Default: true
secret_cloner_enabled: true

# Base Pod (for persistent deployment)
base:
  replicas: 2
  node_selector: node.kubernetes.io/worker-generic-base

# Ephemeral Resources
ephemeral_db: false
ephemeral_cache: false
ephemeral_sqs: false
ephemeral_sns: false
```

### values.yaml (With Ephemeral Database)

```yaml
# ... (all above fields)

ephemeral_db: true

database:
  type: mysql
  name: <service-name>
  namespace: <service-name>
  username: <service-name>
  password: <generated-password>
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database
  version: ""
  attach_volume: false
  volume_size: ""
```

### values.yaml (With Ephemeral Cache)

```yaml
# ... (all above fields)

ephemeral_cache: true

cache:
  namespace: <service-name>
  type: redis
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database-graviton
  version: "6.0"
```

## Kubernetes Resource Templates

### deployment.yaml (Complete Reference)

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
                name: {{ if eq .Values.devstack_label "base" }}{{ .Values.secret_name }}{{ else }}{{ .Values.secret_name }}-{{ .Values.devstack_label }}{{ end }}
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

### svc.yaml

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

### preview-url.yaml (Concierge IngressRoute)

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  labels:
    devstack_label: {{ .Values.devstack_label }}
    {{ if eq .Values.devstack_label "base" }}
    velero.io/include-in-backup: "true"
    protected: "true"
    {{ end }}
  annotations:
    janitor/ttl: "{{ .Values.ttl }}"
  name: injectheader-{{ .Values.devstack_label }}
  namespace: {{ .Values.namespace }}
spec:
  headers:
    customRequestHeaders:
      rzpctx-dev-serve-user: {{ .Values.devstack_label }}
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
    - kind: Rule
      match: Host(`{{ .Values.name }}-{{ .Values.devstack_label }}.dev.razorpay.in`)
      services:
        - name: '{{ .Values.name }}-{{ .Values.devstack_label }}'
          port: {{ .Values.service_port }}
      middlewares:
        - name: injectheader-{{ .Values.devstack_label }}
```

### NOTES.txt

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

## Ephemeral Resource Templates

### db-configmap.yaml

```yaml
{{- if .Values.ephemeral_db }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    name: {{ .Values.database.type }}-{{ .Values.devstack_label }}
    type: {{ .Values.database.type }}
    imageTag: {{ .Values.database.version | quote }}
    namespace: {{ .Values.database.namespace }}
    ttl: {{ .Values.ttl }}
    requestsCpu: {{ .Values.database.requests_cpu }}
    requestsMemory: {{ .Values.database.requests_memory }}
    dnsPolicy: {{ .Values.database.dns_policy }}
    nodeSelector: {{ .Values.database.node_selector }}
    rootPassword: {{ randAlphaNum 12 | lower }}
    attachVolume: {{ .Values.database.attach_volume | default false }}
    volumeSize: {{ .Values.database.volume_size | default "" }}
    databases:
      - dbName: {{ .Values.database.name }}
        username: {{ .Values.database.username }}
        password: {{ .Values.database.password }}
        seeding: false
        snapshotPath: ""
        configKey: db
metadata:
  labels:
    app: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
  name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
  namespace: db-configurator
{{- end }}
```

### db-configurator.yaml

```yaml
{{- if .Values.ephemeral_db }}
apiVersion: batch/v1
kind: Job
metadata:
  name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "3"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: db-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      annotations:
        iam.amazonaws.com/role: dev-serve-api
      labels:
        name: dbc
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:dbc'
          imagePullPolicy: Always
          name: dbc
          resources:
            limits:
              cpu: 200m
              memory: 500Mi
            requests:
              cpu: 100m
              memory: 150Mi
          volumeMounts:
            - name: config-volume
              mountPath: /src/config
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      volumes:
        - name: config-volume
          configMap:
            name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

### secret-cloner.yaml

```yaml
{{- if .Values.secret_cloner_enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sec-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "1"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: secret-cloner
spec:
  backoffLimit: 0
  template:
    metadata:
      labels:
        name: sec
    spec:
      containers:
        - env:
            - name: ACTION
              value: clone
            - name: NAMESPACE
              value: '{{ .Values.namespace }}'
            - name: SECRETNAME
              value:  '{{ .Values.secret_name }}'
            - name: SECRETSUFFIX
              value: '{{ .Values.devstack_label }}'
          image: 'c.rzp.io/razorpay/kube-manifests:sec'
          imagePullPolicy: IfNotPresent
          name: sec
          resources:
            limits:
              cpu: 50m
              memory: 50Mi
            requests:
              cpu: 50m
              memory: 50Mi
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      restartPolicy: OnFailure
{{- end }}
```

### sec-updater-cm.yaml

> ⚠️ **Verify secret key names with the user** before applying. The `key:` values below are defaults — ask the user what environment variable names their application actually reads and update accordingly.

```yaml
{{- if .Values.secret_cloner_enabled }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    updateEntries:
{{- if .Values.ephemeral_db }}
      s1:
        key: DB_HOST
        value: {{ .Values.database.type }}-{{ .Values.devstack_label }}.{{ .Values.database.namespace }}.svc.cluster.local
      s2:
        key: DB_NAME
        value: {{ .Values.database.name }}
      s3:
        key: DB_USERNAME
        value: {{ .Values.database.username }}
      s4:
        key: DB_PASSWORD
        value: {{ .Values.database.password }}
      s5:
        key: DB_URL
        value: {{ .Values.database.type }}-{{ .Values.devstack_label }}.{{ .Values.database.namespace }}.svc.cluster.local
{{- end }}
{{- if or .Values.ephemeral_sqs .Values.ephemeral_sns }}
      aws1:
        key: AWS_REGION
        value: ap-south-1
      aws2:
        key: AWS_ACCESS_KEY_ID
        value: test
      aws3:
        key: AWS_SECRET_ACCESS_KEY
        value: test
{{- end }}
    action: update
    secretName: {{ .Values.secret_name }}-{{ .Values.devstack_label }}
    namespace: {{ .Values.namespace }}
metadata:
  labels:
    app: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "4"
  namespace: secret-cloner
{{- end }}
```

### sec-updater.yaml

```yaml
{{- if .Values.secret_cloner_enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "5"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: secret-cloner
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        name: sec-updater
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:sec'
          imagePullPolicy: Always
          name: sec
          resources:
            limits:
              cpu: 50m
              memory: 50Mi
            requests:
              cpu: 50m
              memory: 50Mi
          volumeMounts:
          - name: config-volume
            mountPath: /src/config
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      volumes:
        - name: config-volume
          configMap:
            name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: OnFailure
{{- end }}
```

### sqs-configmap.yaml

```yaml
{{- if .Values.configurator.sqs }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    queue:
      {{- range $idx, $queue := $.Values.queues }}
      q{{ add $idx 1 }}:
        name: {{ $queue.name }}-{{ $.Values.devstack_label }}
        secretKey: {{ $queue.secretKey }}
      {{- end }}
    updateSecret: true
    kubeSecret: {{ .Values.name }}-{{ .Values.devstack_label }}
    namespace: {{ .Values.namespace }}
    provider: localstack
    enableEndpointPrefix: false
metadata:
  labels:
    app: sqs-{{ .Values.name }}-{{ $.Values.devstack_label }}
  name: sqs-{{ .Values.name }}-{{ $.Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: sqs-configurator
{{- end }}
```

### sqs-configurator.yaml

```yaml
{{- if .Values.configurator.sqs }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sqs-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "3"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: sqs-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        name: irc
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:sqsc'
          imagePullPolicy: IfNotPresent
          name: irc
          resources:
            limits:
              cpu: 100m
              memory: 100Mi
            requests:
              cpu: 100m
              memory: 100Mi
          volumeMounts:
          - name: config-volume
            mountPath: /src/config
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      volumes:
        - name: config-volume
          configMap:
            name: sqs-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

### sns-configmap.yaml

```yaml
{{- if .Values.configurator.sns }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    provider: localstack
    debug: true
    deleteExistingTopic: true  # if you set this then it will delete a topic which is existing with all its subscriptions
    multipleTopics:
      {{- range $pidx, $topic := $.Values.topics }}
      - name: {{ $topic.prefix }}-{{ $.Values.devstack_label }}   # Note, this is the SNS topic name
        subscriptions:
          {{- range $sidx, $subscription := $topic.subscriptions }}
          - protocol: sqs
            endpoint: http://localstack.localstack.svc.cluster.local:4566/000000000000/{{ $subscription }}-{{ $.Values.devstack_label }}
            dlqEndpoint: arn:aws:sqs:ap-south-1:000000000000:{{ $subscription }}-base
          {{- end }}
      {{- end }}
metadata:
  labels:
    app: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
  name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
  namespace: sns-configurator
{{- end }}
```

### sns-configurator.yaml

```yaml
{{- if .Values.configurator.sns }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
  annotations:
    "helm.sh/hook": post-install,post-upgrade
    "helm.sh/hook-weight": "4"
    janitor/ttl: "{{ $.Values.ttl }}"
  namespace: sns-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 0
  template:
    metadata:
      labels:
        name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:snsc'
          imagePullPolicy: IfNotPresent
          name: snsc
          resources:
            limits:
              cpu: 50m
              memory: 50Mi
            requests:
              cpu: 50m
              memory: 50Mi
          volumeMounts:
          - name: config-volume
            mountPath: /src/config
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      volumes:
        - name: config-volume
          configMap:
            name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

### values.yaml (With SNS Topics)

```yaml
# ... (all above fields)

# Enable SNS configurator
configurator:
  sns: true
  sqs: false

# Define SNS topics and their SQS subscriptions
topics:
  - prefix: devstack-my-service-event-processed
    secret_name: SNS_TOPICS_EVENT_PROCESSED_NAME
    subscriptions:
      - devstack-consumer-service-process-event
      - devstack-analytics-service-track-event
  - prefix: devstack-my-service-notification-sent
    secret_name: SNS_TOPICS_NOTIFICATION_SENT_NAME
    subscriptions:
      - devstack-audit-service-log-notification
```

### values.yaml (With Both SQS and SNS)

```yaml
# ... (all above fields)

# Enable both configurators
configurator:
  sns: true
  sqs: true

# SQS queues configuration
queues:
  - name: devstack-my-service-jobs
    secretKey: JOBS_QUEUE_URL
  - name: devstack-my-service-tasks
    secretKey: TASKS_QUEUE_URL

# SNS topics configuration
topics:
  - prefix: devstack-my-service-event-processed
    secret_name: SNS_TOPICS_EVENT_PROCESSED_NAME
    subscriptions:
      - devstack-consumer-service-process-event
```

## Maxwell CDC Template (Ephemeral CDC Events)

When your service uses an ephemeral database and you need CDC (Change Data Capture) events flowing to Kafka, add a Maxwell CDC template. This creates a `MaxwellInstance` CR that the maxwell-operator picks up to deploy a full Maxwell CDC pipeline.

**Prerequisites:**
- `MaxwellInstance` CRD installed on dev-serve (already done)
- Maxwell operator running in `maxwell-microservices` namespace (already done)
- Service deploys with `ephemeral_db: true`

### values.yaml (With Maxwell CDC)

```yaml
# ... (all above fields)

# Maxwell CDC (auto-created when ephemeral_db=true and devstack_label != base)
maxwell:
  image: "c.rzp.io/razorpay/maxwell-microservice:maxwell-c85bb0f1b196a61ea1a5699bd1e7e6b65306b54c"
  dbName: <your_db_name>                    # e.g., pg_router_orders
  metaFilePath: <your_meta_file_path>       # e.g., db_metadata/maxwell/stage/meta_pg_router_order.json
  maxwellType: legacy_microservice          # or "microservice"
  kafkaTopic: ""                            # leave empty for default template
```

### maxwell-cdc.yaml

```yaml
{{- if and .Values.ephemeral_db (ne .Values.devstack_label "base") }}
apiVersion: cdc.razorpay.com/v1alpha1
kind: MaxwellInstance
metadata:
  name: <service>-{{ .Values.devstack_label }}
  namespace: maxwell-microservices
  labels:
    devstack_label: {{ .Values.devstack_label }}
  annotations:
    janitor/ttl: "{{ .Values.ttl }}"
spec:
  devstackLabel: {{ .Values.devstack_label }}
  dbName: {{ .Values.maxwell.dbName | default "<your_db_name>" }}
  replication:
    host: mysql-{{ .Values.devstack_label }}.{{ .Values.namespace }}.svc.cluster.local
    user: {{ .Values.database.db_username | default "<db_user>" }}
    password: {{ .Values.database.db_password | default .Values.secret }}
  metaFilePath: {{ .Values.maxwell.metaFilePath | default "<your_meta_file_path>" }}
  {{- if .Values.maxwell.kafkaTopic }}
  kafkaTopic: {{ .Values.maxwell.kafkaTopic }}
  {{- end }}
  maxwellType: {{ .Values.maxwell.maxwellType | default "legacy_microservice" }}
  filterFilePath: {{ .Values.maxwell.filterFilePath | default "filter_enhanced.js" }}
  image: {{ .Values.maxwell.image | default "c.rzp.io/razorpay/maxwell-microservice:maxwell-c85bb0f1b196a61ea1a5699bd1e7e6b65306b54c" }}
{{- end }}
```

**How it works:**
- Only rendered when `ephemeral_db=true` AND `devstack_label != base`
- Creates a `MaxwellInstance` CR in `maxwell-microservices` namespace
- The maxwell-operator watches this CR and creates a full Maxwell Deployment with:
  - Dev-serve MSK bootstrap patching
  - Kafka headers: `DEVSTACK_LABEL=<your_label>` and `Async-Target-Consumer=<your_label>`
  - JMX exporter sidecar, probes, rds-meta secret
- On helmfile destroy, the CR is deleted and operator cleans up the Deployment

**Reference implementation:** See `helmfile/charts/pg-router/templates/maxwell-cdc.yaml`

### MaxwellInstance CR Field Reference

| Field | Required | Default | Description |
|---|---|---|---|
| `spec.devstackLabel` | Yes | - | Your devstack label (must NOT be `base`) |
| `spec.dbName` | Yes | - | Maxwell DB_NAME |
| `spec.replication.host` | Yes | - | MySQL host (ephemeral DB service) |
| `spec.replication.user` | Yes | - | MySQL replication user |
| `spec.replication.password` | No | - | MySQL password |
| `spec.metaFilePath` | Yes | - | Metadata JSON path in Maxwell image |
| `spec.kafkaTopic` | No | template | Kafka topic override |
| `spec.maxwellType` | No | `legacy_microservice` | `microservice` or `legacy_microservice` |
| `spec.image` | No | latest | Maxwell docker image override |

### Maxwell CDC Connection

```
Bootstrap Servers: devserve-kafka-msk.np.razorpay.vpc:9094
Kafka Headers: DEVSTACK_LABEL=<devstack_label>, Async-Target-Consumer=<devstack_label>
Topic Pattern: mysql_cdc_events_<database>_<table>
```

## Debezium CDC Template (Ephemeral PostgreSQL CDC Events)

When your service uses an ephemeral PostgreSQL database and you need CDC (Change Data Capture) events flowing to Kafka, add a Debezium CDC template. This creates a `DebeziumInstance` CR that the debezium-operator picks up to register a Debezium connector on Kafka Connect.

**Prerequisites:**
- `DebeziumInstance` CRD installed on dev-serve (already done)
- Debezium operator running in `debezium` namespace (already done)
- Kafka Connect running in `debezium` namespace (already done)
- Service deploys with `ephemeral_db: true` and `ephemeral_debezium: true`

### values.yaml (With Debezium CDC)

```yaml
# ... (all above fields)

# Debezium CDC (auto-created when ephemeral_db=true, ephemeral_debezium=true, devstack_label != base)
ephemeral_debezium: false
# debezium_topicPrefix: ""        # optional: override topic prefix (default: {dbName}-{devstackLabel})
# debezium_tablesInclude: ""      # optional: comma-separated table include filter
# debezium_tablesExclude: ""      # optional: comma-separated table exclude filter
```

### debezium-cdc.yaml

```yaml
{{- if and .Values.ephemeral_db .Values.ephemeral_debezium (ne .Values.devstack_label "base") }}
apiVersion: cdc.razorpay.com/v1alpha1
kind: DebeziumInstance
metadata:
  name: {{ .Values.name }}-{{ .Values.devstack_label }}
  namespace: debezium
  labels:
    devstack_label: {{ .Values.devstack_label }}
  annotations:
    janitor/ttl: "{{ .Values.ttl }}"
spec:
  devstackLabel: {{ .Values.devstack_label }}
  dbName: {{ .Values.database.db1_name }}
  replication:
    host: postgres-{{ .Values.devstack_label }}.{{ .Values.database.namespace }}.svc.cluster.local
    port: 5432
    user: {{ .Values.database.db1_username }}
    password: {{ .Values.database.db1_secret }}
  {{- if .Values.debezium_topicPrefix }}
  topicPrefix: {{ .Values.debezium_topicPrefix }}
  {{- end }}
  {{- if .Values.debezium_tablesInclude }}
  tablesInclude: {{ .Values.debezium_tablesInclude | quote }}
  {{- end }}
  {{- if .Values.debezium_tablesExclude }}
  tablesExclude: {{ .Values.debezium_tablesExclude | quote }}
  {{- end }}
{{- end }}
```

**How it works:**
- Only rendered when `ephemeral_db=true`, `ephemeral_debezium=true`, AND `devstack_label != base`
- Creates a `DebeziumInstance` CR in `debezium` namespace
- The debezium-operator watches this CR and registers a Debezium PostgreSQL connector on Kafka Connect with:
  - Automatic topic prefix and replication slot naming
  - Kafka headers: `DEVSTACK_LABEL=<your_label>` and `Async-Target-Consumer=<your_label>`
  - Finalizer-based cleanup on CR deletion
- On helmfile destroy, the CR is deleted and operator cleans up the connector

**Reference implementation:** See `helmfile/charts/optimizer-core/templates/debezium-cdc.yaml`

### DebeziumInstance CR Field Reference

| Field | Required | Default | Description |
|---|---|---|---|
| `spec.devstackLabel` | Yes | - | Your devstack label (must NOT be `base`) |
| `spec.dbName` | Yes | - | PostgreSQL database name |
| `spec.replication.host` | Yes | - | PostgreSQL host (ephemeral DB service) |
| `spec.replication.port` | No | `5432` | PostgreSQL port |
| `spec.replication.user` | Yes | - | PostgreSQL replication user |
| `spec.replication.password` | Yes | - | PostgreSQL password |
| `spec.topicPrefix` | No | `{dbName}-{devstackLabel}` | Kafka topic prefix |
| `spec.slotName` | No | `dbz_{dbName}_{devstackLabel}` | Replication slot name |
| `spec.pluginName` | No | `pgoutput` | Logical decoding plugin |
| `spec.tablesInclude` | No | all tables | Table include filter |
| `spec.tablesExclude` | No | none | Table exclude filter |

### Debezium CDC Connection

```
Bootstrap Servers: devserve-kafka-msk.np.razorpay.vpc:9094
Kafka Headers: DEVSTACK_LABEL=<devstack_label>, Async-Target-Consumer=<devstack_label>
Topic Pattern: <topicPrefix>.<schema>.<table>
Connector Name: dbz-<dbName>-<devstackLabel>
```

## Template Variable Reference

### Common Template Variables

| Variable | Usage | Example |
|----------|-------|---------|
| `.Values.name` | Service name | `payment-service` |
| `.Values.namespace` | Kubernetes namespace | `payment-service` |
| `.Values.devstack_label` | Devstack label/user | `parag`, `base` |
| `.Values.ttl` | Time to live | `1h`, `8h`, `forever` |
| `.Values.image` | Image tag/commit hash | `abc123def` |
| `.Values.container_port` | Application port | `9400` |
| `.Values.service_port` | Service port | `80` |
| `.Chart.Name` | Chart name | `payment-service` |

### Conditional Logic

**Base vs Ephemeral**:
```yaml
{{ if eq .Values.devstack_label "base" }}
  # Base-specific configuration
{{ else }}
  # Ephemeral-specific configuration
{{ end }}
```

**Conditional Resource Creation**:
```yaml
{{- if .Values.ephemeral_db }}
# Database resources only if enabled
{{- end }}
```

## Helmfile Entry Template

```yaml
- name: <service-name>-{{ .Values.devstack_label }}
  namespace: <service-name>
  chart: ./charts/<service-name>
  values:
    - image: <commit-hash>
    - devstack_label: {{ .Values.devstack_label }}
    - ttl: {{ .Values.ttl }}
    - namespace: <service-name>
    - secret: {{ .Values.secret }}
```

## Connection String Formats

### Database Connection
```
Host: <database-type>-<devstack-label>.<namespace>.svc.cluster.local
Port: 3306 (MySQL) or 5432 (Postgres)
Database: <database-name>
Username: <database-username>
Password: <database-password>
```

### Redis Connection
```
Host: redis-<devstack-label>.<namespace>.svc.cluster.local
Port: 6379
```

### SQS Queue URL
```
http://localstack.localstack.svc.cluster.local:4566/000000000000/<queue-name>-<devstack-label>
```

### SNS Topic ARN
```
arn:aws:sns:ap-south-1:000000000000:<topic-prefix>-<devstack-label>
```

### SNS Subscription Endpoint (SQS)
```
http://localstack.localstack.svc.cluster.local:4566/000000000000/<subscription-queue>-<devstack-label>
```

### Kafka Connection
```
Bootstrap Servers: devserve-kafka-msk.np.razorpay.vpc:9094
Topic: <topic-name>-<devstack-label>
```

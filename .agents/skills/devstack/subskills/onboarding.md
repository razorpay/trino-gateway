# Application Onboarding Subskill

Comprehensive workflow for onboarding new applications to the devstack ecosystem.

## ⚠️ CRITICAL: Use Own Knowledge Base Only

**When onboarding a new service, use ONLY the templates and patterns defined in this skill.** Do NOT:
- Copy or infer chart structure from other existing services in kube-manifests
- Assume a service uses the same image tag format, resource limits, or template layout as another service
- Reference another chart to guess values

Every service is different. Use this skill's templates as the starting point, then adapt based on what you discover about THIS service's actual configuration (GitHub workflows, runtime requirements stated by the user).

## ⚠️ CRITICAL: Helm Chart Location

**ALWAYS create/update helm charts in `helmfile/charts/<application-name>` within the kube-manifests repository ONLY.**

- ✅ CORRECT: `<kube-manifests-repo>/helmfile/charts/<service-name>/`
- ❌ WRONG: `charts/<service-name>/`
- ❌ WRONG: `<any-other-repo>/charts/<service-name>/`
- ❌ WRONG: Any path outside `helmfile/charts/` in kube-manifests repo

**This is the ONLY valid location for helm charts in the devstack ecosystem.**

## Purpose

Guide developers through the complete process of onboarding a new application to devstack, from helm chart creation to deployment and monitoring.

## When to Use

- Onboarding a new microservice to devstack
- Setting up ephemeral environments for a new application
- Creating helm charts for kubernetes deployment
- Configuring databases, caches, and queues for an application
- Setting up secrets and monitoring for a new service

## Prerequisites

- **kube-manifests repository** — auto-cloned if not found locally (requires `git` and SSH access to GitHub)
- Access to kube-manifests repository (SSH key configured)
- kubectl configured with devstack cluster access
- Understanding of the application's runtime requirements (DB, cache, queues, etc.)
- Application container image available in `c.rzp.io/razorpay/<service-name>`
- Helmfile directory configured (see [../SKILL.md#configuration](../SKILL.md#configuration))

### Clone kube-manifests Repository

The skill will automatically clone the repository if it is not found locally (see Phase 0 below). No manual action needed.

## Onboarding Workflow

### Phase 0: Path Detection and Repository Check

**Before starting**, locate or clone the kube-manifests repository and confirm the helmfile directory path.

#### Step 1: Check git availability

```bash
git --version
```

If this fails → **stop and inform the user**: git is required to clone kube-manifests. Ask them to install git before continuing.

#### Step 2: Search for kube-manifests locally

Check in priority order — **stop at the first match**:

```bash
# 1. Path from config.json (highest priority — user explicitly configured this)
#    Read helmfile_directory from agent-skills/infrastructure/skills/devstack/config.json
#    If set and helmfile.yaml exists there → use it

# 2. Current working directory
ls "$(pwd)/kube-manifests/helmfile/helmfile.yaml" 2>/dev/null

# 3. One level up (common when working inside a sibling repo)
ls "$(pwd)/../kube-manifests/helmfile/helmfile.yaml" 2>/dev/null

# 4. Home directory subtree (covers ~/razorpay/, ~/work/, ~/code/, etc.)
find ~ -maxdepth 5 -name "helmfile.yaml" -path "*/kube-manifests/helmfile/*" 2>/dev/null | head -1
```

If any of the above resolves → set `HELMFILE_DIR` to that path and **skip Step 3**.

#### Step 3: Clone if not found locally

Only reached if all local searches in Step 2 returned nothing.

```bash
# Clone into the current working directory
git clone --depth 1 --single-branch git@github.com:razorpay/kube-manifests.git
```

If the clone fails (SSH key not configured, no network access, etc.) → **stop immediately** and show the user the exact git error. Do not proceed.

On success, set `HELMFILE_DIR="$(pwd)/kube-manifests/helmfile"`.

#### Step 4: Confirm and report helmfile directory path

```bash
echo "Using helmfile directory: $HELMFILE_DIR"
ls "$HELMFILE_DIR/helmfile.yaml"   # final sanity check
```

Inform the user which path is being used. For all examples below, `<HELMFILE_DIR>` refers to this value.

### Phase 1: Repository Setup

1. **Navigate to helmfile directory in kube-manifests repo**
   ```bash
   # Use the path from config.json
   cd <HELMFILE_DIR>

   # Verify you're in the right place
   ls helmfile.yaml

   # Create a branch for onboarding
   git checkout -b onboard-<service-name>
   ```

2. **Create helm chart directory structure in helmfile/charts/<service-name>**
   ```bash
   # CRITICAL: Must be inside helmfile/charts/ directory
   cd helmfile/charts
   mkdir <service-name>
   cd <service-name>
   mkdir templates
   touch Chart.yaml values.yaml
   cd templates
   touch NOTES.txt deployment.yaml svc.yaml preview-url.yaml
   ```

   **Final directory structure:**
   ```
   <kube-manifests-repo>/helmfile/charts/<service-name>/
   ├── Chart.yaml
   ├── values.yaml
   └── templates/
       ├── NOTES.txt
       ├── deployment.yaml
       ├── svc.yaml
       └── preview-url.yaml
   ```

### Phase 1.5: Discover Image Tag Pattern (MANDATORY before chart configuration)

**Before writing any chart, discover how the service builds and tags its Docker images.** Do NOT guess or infer from other services' charts. Always use the service's own CI configuration as the authoritative source.

#### Step 1: Check GitHub Actions workflows in the service's repo

```bash
gh api repos/razorpay/<service-name>/contents/.github/workflows --jq '.[].name' 2>/dev/null
```

Look for workflows that contain `docker`, `build`, `image`, or `push` in their name.

#### Step 2: Read the relevant workflow file

```bash
gh api repos/razorpay/<service-name>/contents/.github/workflows/<workflow-file>.yml --jq '.content' | base64 -d 2>/dev/null
```

#### Step 3: Identify which action is used

**Case A — Razorpay composite action** (either `razorpay/actions/docker-image-build-push` or `razorpay/docker-image-build-push` — treat both identically):

If you need to inspect the action definition, always fetch from the `master` branch:
```bash
gh api "repos/razorpay/actions/contents/docker-image-build-push/action.yml?ref=master" --jq '.content' | base64 -d 2>/dev/null
```

Read the action inputs in the workflow step to find the tag pattern. Key inputs to look for: `image-name`, `tags`, `tag-suffix`, `tag-prefix`. Common Razorpay patterns:
- `api-<commit>` (API/web image)
- `worker-<commit>` (worker image)
- `api_<commit>` (underscore separator variant)

**Case B — Open source action** (`docker/build-push-action`):

Look for `tags:` in the workflow step. The image repository itself may encode the image type:

```yaml
# Tag on a single image — commit hash is the tag
tags: c.rzp.io/razorpay/<service>:${{ github.sha }}

# Tag with prefix separator
tags: c.rzp.io/razorpay/<service>:api-${{ github.sha }}

# Separate image repository per image type (worker image in its own repo)
tags: c.rzp.io/razorpay/<service>-worker:${{ github.sha }}
tags: c.rzp.io/razorpay/<service>-api:${{ github.sha }}
```

> In the last pattern, `image_base` in values.yaml must reflect the correct repo (e.g. `c.rzp.io/razorpay/<service>-worker`) and the tag in deployment.yaml would be just `{{ .Values.image }}` with no prefix.

#### Step 4: Set the correct image tag in deployment.yaml

Once you know the tag pattern, configure `deployment.yaml` accordingly:

```yaml
# Tag is `api-<commit>` (prefix before commit, same repo)
image: "{{ .Values.image_base }}:api-{{ .Values.image }}"

# Tag is `<commit>` only (no prefix, same repo)
image: "{{ .Values.image_base }}:{{ .Values.image }}"

# Tag is `worker-<commit>` for a worker deployment (same repo)
image: "{{ .Values.image_base }}:worker-{{ .Values.image }}"

# Image is in a separate repo per type, e.g. c.rzp.io/razorpay/<service>-worker:<commit>
# → set image_base: c.rzp.io/razorpay/<service>-worker in values.yaml
# → tag in template: {{ .Values.image_base }}:{{ .Values.image }}
```

> **Recognized patterns — proceed without asking the user:**
> - `c.rzp.io/razorpay/<service>:<commit>` (bare commit hash)
> - `c.rzp.io/razorpay/<service>:api-<commit>` (api- prefix)
> - `c.rzp.io/razorpay/<service>:worker-<commit>` (worker- prefix)
> - `c.rzp.io/razorpay/<service>:api_<commit>` (underscore variant)
> - `c.rzp.io/razorpay/<service>-worker:<commit>` (separate repo per image type)
>
> **If the pattern does NOT match any of the above — STOP and ask the user:**
> "What does a typical image tag look like for this service? For example: `api-abc123def`, `abc123def`, or is there a separate image repo like `c.rzp.io/razorpay/<service>-worker`? Or share the GitHub Actions workflow file."

**NEVER copy the image tag pattern from another service's chart.** Each service has its own build configuration.

### Phase 2: Chart Configuration

#### Chart.yaml

Create basic chart metadata:

```yaml
apiVersion: v2
name: <service-name>
description: <service-name> helmchart
type: application
version: 0.1.0
appVersion: 1.16.0
```

#### values.yaml

Configure service parameters. **Critical fields**:

```yaml
# Application Configuration
app_env: dev
namespace: <service-name>
name: <service-name>
bu: platform  # Business unit

# Image Configuration
image_base: c.rzp.io/razorpay/<service-name>
image_pull_policy: IfNotPresent
# Note: `image` in helmfile.yaml is always the raw commit hash only (e.g. abc123def).
# The chart template is responsible for constructing the full tag including any prefix.
# Example: if the built image tag is `api_abc123def`, the deployment template should be:
#   image: "{{ .Values.image_base }}:api_{{ .Values.image }}"
# Never put the tag prefix in helmfile.yaml — that belongs in the chart template.

# Resource Limits (REQUIRED)
web_requests_cpu: 50m
web_requests_memory: 200Mi
web_limits_memory: 500Mi

# Deployment Configuration
replicas: 1
service_port: 80
container_port: 9400  # Application server port

# Node Placement
node_selector: node.kubernetes.io/worker-generic
dns_policy: ClusterFirst

# Secrets
secret_name: <service-name>

# Secret cloning — when true, clones <secret_name> into <secret_name>-<devstack_label>
# and injects ephemeral resource credentials into the clone.
# Set false only if you want all deployments to use the shared base secret as-is.
secret_cloner_enabled: true

# Base Pod Configuration (for persistent base deployment)
base:
  replicas: 2
  node_selector: node.kubernetes.io/worker-generic-base

# Ephemeral Resources (Configure as needed)
ephemeral_db: true
ephemeral_cache: false
ephemeral_sqs: false
ephemeral_sns: false
```

**Optional Database Configuration** (if ephemeral_db: true):

```yaml
database:
  type: mysql  # or postgres
  name: <service-name>
  namespace: <service-name>
  username: <service-name>
  password: <generated-password>
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database
  version: ""  # Defaults to latest
  attach_volume: false
  volume_size: ""
```

**Optional Cache Configuration** (if ephemeral_cache: true):

```yaml
cache:
  namespace: <service-name>
  type: redis
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database-graviton
  version: "6.0"
```

#### NOTES.txt

Template for post-deployment access information:

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

### Phase 3: Create Kubernetes Resources

#### deployment.yaml

**Critical Requirements**:
- Suffix deployment name with `{{ .Values.devstack_label }}`
- Include mandatory annotations: `janitor/ttl`
- Include mandatory labels: `bu`, `name`, `devstack_label`
- Configure resource limits (CPU + Memory)
- Add readiness and liveness probes
- Mount secrets properly

**Minimum Template**:

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
  replicas: {{ .Values.replicas }}
  selector:
    matchLabels:
      name: {{ .Values.name }}-{{ .Values.devstack_label }}
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
    spec:
      containers:
        - name: web
          image: {{ .Values.image_base }}:{{ .Values.image }}
          imagePullPolicy: {{ .Values.image_pull_policy }}
          ports:
            - containerPort: {{ .Values.container_port }}
          env:
            - name: APP_ENV
              value: {{ .Values.app_env }}
          envFrom:
            - secretRef:
                # Use base secret when: devstack_label is "base" OR secret_cloner_enabled is false
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
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health
              port: {{ .Values.container_port }}
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          resources:
            requests:
              cpu: {{ .Values.web_requests_cpu }}
              memory: {{ .Values.web_requests_memory }}
            limits:
              memory: {{ .Values.web_limits_memory }}
      dnsPolicy: {{ .Values.dns_policy }}
      nodeSelector:
        {{ if eq .Values.devstack_label "base" }}
          {{ .Values.base.node_selector }}: ""
        {{ else }}
          {{ .Values.node_selector }}: ""
        {{ end }}
```

#### svc.yaml

Create ClusterIP service:

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
  type: ClusterIP
```

#### service-account.yaml (if the service needs an IAM role)

**CRITICAL**: ServiceAccounts are namespace-scoped and **shared across all deployments** of a service. Only the base deployment should create it — personal deployments (label: `parag`, `alice`, etc.) reuse it without recreating it.

**Always gate ServiceAccount creation behind `devstack_label == "base"`**:

```yaml
{{ if eq .Values.devstack_label "base" }}
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: {{ .Values.serviceAccountRoleArn }}
  name: {{ .Values.serviceaccount_name }}
  namespace: {{ .Values.namespace }}
imagePullSecrets:
  - name: registry
{{ end }}
```

**Why this matters**:
- If the SA is created unconditionally, every personal deployment tries to own it via Helm — causing a hard failure: `ServiceAccount exists and cannot be imported into the current release: invalid ownership metadata`
- Gating on `devstack_label == "base"` means the SA is created once (by Spinnaker when deploying the base pod) and all personal deployments simply use it without conflict

**Values to add** (`values.yaml`):
```yaml
serviceaccount_name: <service-name>
serviceAccountRoleArn: arn:aws:iam::<account-id>:role/<role-name>
```

**If the service doesn't need an IAM role**: omit this file entirely. Kubernetes provides a default SA automatically.

#### preview-url.yaml

Create IngressRoute for external access. Supports two access patterns:
1. `service-name.dev.razorpay.in` + header `rzpctx-dev-serve-user: <label>`
2. `service-name-<label>.dev.razorpay.in`

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
    {{ if eq .Values.devstack_label "base" }}
    - kind: Rule
      match: Host(`{{ .Values.name }}.dev.razorpay.in`)
      services:
        - name: '{{ .Values.name }}-{{ .Values.devstack_label }}'
          port: {{ .Values.service_port }}
    {{ end }}
```

**Why the base deployment gets an extra rule without middleware:**

Personal deployments use `injectheader` middleware on their direct URL to inject `rzpctx-dev-serve-user: <label>` into requests. The base deployment instead gets a catch-all rule on `<service>.dev.razorpay.in` (no label suffix, no middleware) — this makes it the default handler for any request to the shared URL that doesn't match a personal deployment's header. Adding the middleware would inject `rzpctx-dev-serve-user: base` which is redundant and could interfere with routing logic.

### Phase 4: Configure Ephemeral Resources

#### Ephemeral Database (Optional)

If `ephemeral_db: true`, create these files:

**db-configmap.yaml**:

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

**db-configurator.yaml**:

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
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      volumes:
        - name: config-volume
          configMap:
            name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

**Database Connection Details**:
- Host: `<database-type>-<label>.<namespace>.svc.cluster.local`
- Port: 3306 (MySQL) or 5432 (Postgres)
- Database: As configured in configmap
- Username/Password: As configured in configmap

#### Ephemeral Cache (Optional)

Similar structure to database. Create `cache-configmap.yaml` and `cache-configurator.yaml`.

**Cache Connection**: `redis-<label>.<namespace>.svc.cluster.local:6379`

#### Ephemeral SQS/Queues (Optional)

For applications requiring async queues:

**sqs-configmap.yaml**: Define queue names and secret keys
**sqs-configurator.yaml**: Job to provision queues on localstack

**Queue URL Format**: `http://localstack.localstack.svc.cluster.local:4566/000000000000/<queue-name>-<label>`

**Example values.yaml configuration**:
```yaml
# Both flags must be set when enabling SQS:
# - ephemeral_sqs: activates secret-cloner and sec-updater
# - configurator.sqs: activates the SQS provisioner job
ephemeral_sqs: true

configurator:
  sqs: true

queues:
  - name: devstack-my-service-jobs
    secretKey: JOBS_QUEUE_URL
  - name: devstack-my-service-tasks
    secretKey: TASKS_QUEUE_URL
```

See [ephemeral-sqs.md](../references/templates/ephemeral-sqs.md) for complete templates.

#### Ephemeral SNS/Topics (Optional)

For applications requiring pub/sub messaging with SNS topics:

**sns-configmap.yaml**: Define SNS topics and their SQS subscriptions
**sns-configurator.yaml**: Job to provision topics and subscriptions on localstack

**Topic ARN Format**: `arn:aws:sns:ap-south-1:000000000000:<topic-prefix>-<label>`

**Subscription Endpoint Format**: `http://localstack.localstack.svc.cluster.local:4566/000000000000/<subscription-queue>-<label>`

**Example values.yaml configuration**:
```yaml
# Both flags must be set when enabling SNS:
# - ephemeral_sns: activates secret-cloner and sec-updater
# - configurator.sns: activates the SNS provisioner job
ephemeral_sns: true

configurator:
  sns: true

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

**Key Features**:
- **Multiple topics**: Define multiple SNS topics with different subscriptions
- **SQS subscriptions**: Each topic can have multiple SQS queue subscriptions
- **Auto-cleanup**: Topics are deleted and recreated on each sync (controlled by `deleteExistingTopic`)
- **Debug mode**: Enabled by default for better troubleshooting
- **Dead Letter Queue**: Supports DLQ endpoints for failed messages

**Hook Execution Order**:
1. `pre-install,pre-upgrade` with weight `2` - Creates SNS ConfigMap
2. `post-install,post-upgrade` with weight `4` - Executes SNS configurator job

See [ephemeral-sns.md](../references/templates/ephemeral-sns.md) for complete templates.

#### Using Both SQS and SNS (Optional)

You can enable both configurators for applications that need both queues and topics:

```yaml
configurator:
  sqs: true
  sns: true

# SQS queues
queues:
  - name: devstack-my-service-jobs
    secretKey: JOBS_QUEUE_URL

# SNS topics that publish to SQS queues
topics:
  - prefix: devstack-my-service-events
    secret_name: SNS_TOPICS_EVENTS_NAME
    subscriptions:
      - devstack-my-service-jobs  # Can subscribe to your own SQS queues
      - devstack-other-service-consumer
```

**Common Use Cases**:
- **Event-driven architecture**: SNS topics for events, SQS for processing
- **Fan-out pattern**: One SNS topic publishes to multiple SQS consumers
- **Microservices communication**: SNS for pub/sub, SQS for work queues
- **Cross-service integration**: SNS topics consumed by multiple services

### Phase 5: Secrets Management

#### Base Secrets

1. **Add secrets to credstash**:
   - URL: https://credstash-ui.concierge.stage.razorpay.in/dist/
   - Table: `kubestash-dev-serve`
   - Key format: `<namespace>/<secret-name>/<secret-key>`
   - Example: `pg-router/pg-router-live/DB_HOST`

2. **Secrets sync automatically** within 5 minutes via kubestash job

3. **Verify secret creation**:
   ```bash
   kubectl --context dev-serve get secret -n <namespace> <secret-name>
   ```

#### Ephemeral Secrets

For label-specific overrides (e.g., ephemeral DB credentials):

**secret-cloner.yaml**: Clones base secret

> **Required whenever `secret_cloner_enabled: true` (default) and any ephemeral resource is enabled.** Both conditions must hold — the flag controls secret cloner and sec-updater together.

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
    spec:
      containers:
        - env:
            - name: ACTION
              value: clone
            - name: NAMESPACE
              value: '{{ .Values.namespace }}'
            - name: SECRETNAME
              value: '{{ .Values.secret_name }}'
            - name: SECRETSUFFIX
              value: '{{ .Values.devstack_label }}'
          image: 'c.rzp.io/razorpay/kube-manifests:sec'
          name: sec
      restartPolicy: OnFailure
{{- end }}
```

**sec-updater-cm.yaml**: Define keys to override

> **Required whenever `secret_cloner_enabled: true` (default) and any ephemeral resource is enabled.** Uses the same condition as secret-cloner. Inject DB credentials when `ephemeral_db` is true. Inject localstack AWS credentials when `ephemeral_sqs` or `ephemeral_sns` is true.

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
  name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "4"
  namespace: secret-cloner
{{- end }}
```

> ⚠️ **Action required — verify secret key names with the user**
>
> The `key:` values above (`DB_HOST`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`, `DB_URL`, `AWS_REGION`, etc.) are defaults. **Ask the user to confirm the exact environment variable names their application reads from the secret**, and update the `key:` fields accordingly before applying.
>
> Example prompt to ask:
> _"What environment variable names does `<service-name>` use for the database host, name, username, password, and URL? (e.g. `DATABASE_URL`, `MYSQL_HOST`, `DB_PASS`) I'll update the secret keys to match."_

**sec-updater.yaml**: Job to update secrets

### Phase 6: Add Service to Helmfile

1. **Edit helmfile.yaml in kube-manifests repo**:
   ```bash
   cd <kube-manifests-repo>/helmfile
   vim helmfile.yaml
   ```

2. **Add service entry**:
   ```yaml
   ## My Service (descriptive comment using ## not #)
   # - name: <service-name>-{{ .Values.devstack_label }}
   #   namespace: <service-name>
   #   chart: ./charts/<service-name>
   #   values:
   #     - image: <commit-hash>
   #     - devstack_label: {{ .Values.devstack_label }}
   #     - ttl: {{ .Values.ttl }}
   #     - namespace: <service-name>
   #     - secret: {{ .Values.secret }}
   ```

   **CRITICAL — Comment convention in the `releases:` section**:

   The CI `validate-helmfile` step strips **one leading `#`** from all commented lines in the `releases:` block to validate the full helmfile. This means:

   | What you write | After CI strips one `#` | Result |
   |---|---|---|
   | `# - name: service-...` | `- name: service-...` | ✅ valid YAML |
   | `# My Service description` | `My Service description` | ❌ invalid YAML — bare text breaks parsing |
   | `## My Service description` | `# My Service description` | ✅ valid comment |

   **Rule**: Any descriptive/freeform comment inside the `releases:` section MUST use `##` (double hash), not `#`. Single `#` is only for commented-out YAML entries (`# - name:`, `#   namespace:`, etc.).

### Phase 7: Create Pull Request

After all helm chart files and helmfile.yaml entries are created/updated, commit and raise a PR to kube-manifests. This applies to any change made via this skill — new service onboarding, adding ephemeral DB/cache/SQS/SNS, or any other chart modification.

#### Step 1: Verify git is available

```bash
git --version
```

If this fails → skip all remaining steps in Phase 7 (no commit, no push, no PR) and jump to **Option 4** below.

#### Step 2: Determine branch name and PR metadata

The branch name and PR title depend on what was changed. Choose based on the operation:

| Operation | Branch name | PR title |
|---|---|---|
| New service onboarding | `onboard/<service-name>` | `feat(<service-name>): onboard to devstack` |
| Add ephemeral DB | `feat/<service-name>/add-ephemeral-db` | `feat(<service-name>): add ephemeral database` |
| Add ephemeral cache | `feat/<service-name>/add-ephemeral-cache` | `feat(<service-name>): add ephemeral cache` |
| Add ephemeral SQS | `feat/<service-name>/add-sqs` | `feat(<service-name>): add ephemeral SQS queues` |
| Add ephemeral SNS | `feat/<service-name>/add-sns` | `feat(<service-name>): add ephemeral SNS topics` |
| Add SQS + SNS | `feat/<service-name>/add-sqs-sns` | `feat(<service-name>): add ephemeral SQS and SNS` |
| Other chart changes | `feat/<service-name>/<short-description>` | `feat(<service-name>): <short-description>` |

Use `<BRANCH_NAME>` and `<PR_TITLE>` as placeholders for the chosen values in the steps below.

#### Step 3: Check active branch and prepare

```bash
cd <kube-manifests-root>

# Always check the active branch explicitly
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo "Active branch: $CURRENT_BRANCH"

if [ "$CURRENT_BRANCH" = "master" ]; then
  # On master — check if target branch already exists before creating
  if git branch --list "<BRANCH_NAME>" | grep -q "<BRANCH_NAME>"; then
    # Branch exists locally — check out and use it
    git checkout <BRANCH_NAME>
    echo "Using existing local branch: <BRANCH_NAME>"
  else
    # Create new branch from latest master
    git pull origin master
    git checkout -b <BRANCH_NAME>
    echo "Created new branch: <BRANCH_NAME>"
  fi
else
  # Already on a feature branch — use it as-is
  echo "Using existing branch: $CURRENT_BRANCH"
  # <BRANCH_NAME> for push/PR = $CURRENT_BRANCH
fi

# Stage all changed files under helmfile/
git add helmfile/charts/<service-name>/ helmfile/helmfile.yaml

# Commit
git commit -m "<PR_TITLE>

<bullet list of what was added or changed>"

# Push
git push -u origin HEAD
```

#### Step 4: PR Creation — Try in Order

**Option 1: GitHub MCP** *(preferred — try first)*

Check whether a GitHub MCP tool is available (e.g. `mcp__github__create_pull_request` or similar). If it is, use it:

```
Create a PR in razorpay/kube-manifests:
  title: "<PR_TITLE>"
  body:  "## Summary\n<bullet list of changes>\n\n## Test plan\n- [ ] helmfile template renders without errors\n- [ ] Deploy with devstack label and verify expected resources start"
  base:  master
  head:  <BRANCH_NAME>
```

**Option 2: `gh` CLI** *(fallback if GitHub MCP not available)*

```bash
gh auth status  # verify authenticated

gh pr create \
  --repo razorpay/kube-manifests \
  --title "<PR_TITLE>" \
  --body "## Summary
<bullet list of changes>

## Test plan
- [ ] helmfile template renders without errors
- [ ] Deploy with devstack label and verify expected resources start" \
  --base master
```

**Option 3: Git only** *(fallback if neither GitHub MCP nor `gh` CLI is available)*

Branch already pushed in Step 3. Inform the user:

```
✅ Branch pushed: <BRANCH_NAME>
⚠️  Could not create PR automatically (no GitHub MCP or gh CLI available).
👉 Open PR manually at:
   https://github.com/razorpay/kube-manifests/compare/<BRANCH_NAME>?expand=1
```

Show the **Recommendation** message below.

**Option 4: git not available — skip commit, push, and PR entirely**

```
⚠️  Skipping PR creation — git is not available in this environment.
    Helm chart files have been created/updated locally. To raise a PR manually:
    1. Install git, then run: git add + git commit + git push
    2. Open a PR at: https://github.com/razorpay/kube-manifests/compare
```

Show the **Recommendation** message below.

#### Recommendation (show whenever PR creation is not fully automatic)

```
💡 To enable automatic PR creation in future, install one of:
   • GitHub MCP server — add to your MCP config and set GITHUB_PERSONAL_ACCESS_TOKEN:
     {"type": "stdio", "command": "npx", "args": ["-y", "@modelcontextprotocol/server-github"]}
   • gh CLI — https://cli.github.com/ (run `gh auth login` after install)
```

#### Outcome Report

```
## 📬 Pull Request
✅ PR created: https://github.com/razorpay/kube-manifests/pull/<N>
   Branch: <BRANCH_NAME> → master
```

or, if PR creation was partial/skipped:

```
## 📬 Pull Request
⚠️ <reason — no GitHub MCP / no gh CLI / git not available>
   <Branch pushed / files created locally>
   <Manual PR link or manual git instructions>
💡 <Recommendation>
```

---

### Phase 8: Validation & Deployment

1. **Validate chart syntax in helmfile/charts/**:
   ```bash
   cd <kube-manifests-repo>/helmfile
   helm lint charts/<service-name>/
   ```

2. **Validate template rendering**:
   ```bash
   helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service-name>-<label> template
   ```

3. **Deploy**:
   ```bash
   helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service-name>-<label> sync
   ```

4. **Verify deployment**:
   ```bash
   kubectl --context dev-serve get pods -n <service-name> -l devstack_label=<label>
   kubectl --context dev-serve get svc -n <service-name> -l devstack_label=<label>
   ```

### Phase 9: Setup Monitoring & Logging

#### Logging

Logs automatically pushed to Coralogix if printed to stdout.

**Access logs**: https://razorpay-non-prod.app.coralogix.in/#/query-new/archive-logs

#### Monitoring

Ensure prometheus annotations in deployment:

```yaml
template:
  metadata:
    annotations:
      prometheus.io/path: /metrics
      prometheus.io/port: "<container-port>"
      prometheus.io/scrape: "true"
```

### Phase 10: Deploy Base Pod via Spinnaker (Optional)

For persistent base deployments, use Spinnaker V3 pipeline:

1. **Create pipeline config** in spinnacode repo
2. **Configure variables**:
   - `namespace`: Service namespace
   - `helm_chart_path_prefix`: S3 location
   - `default_overrides`: `devstack_label=base,ttl=forever`
   - `helm_release_name_override`: `<service-name>-base`

3. **Execute deployment** at deploy.razorpay.com

## Validation Checklist

Before deployment, verify:

- [ ] Chart.yaml has correct name and version
- [ ] values.yaml has all required fields
- [ ] Resource limits configured (CPU + Memory)
- [ ] Mandatory annotations present: `janitor/ttl`
- [ ] Mandatory labels present: `bu`, `name`, `devstack_label`
- [ ] Probes configured (liveness + readiness)
- [ ] Secrets mounted correctly
- [ ] Service targetPort matches container port
- [ ] IngressRoute configured for external access
- [ ] Ephemeral resources configured if needed
- [ ] Base secrets added to credstash
- [ ] Service added to helmfile.yaml
- [ ] Template renders without errors
- [ ] Monitoring annotations present
- [ ] Branch created (non-master) and changes committed
- [ ] PR raised to kube-manifests (via GitHub MCP, gh CLI, or manually)

## Common Issues

**Issue**: Template rendering fails
**Fix**: Run `helm lint charts/<service-name>/` to identify syntax errors

**Issue**: Secrets not found
**Fix**: Verify secret exists: `kubectl --context dev-serve get secret -n <namespace> <secret-name>`

**Issue**: Pods fail to start (ImagePullBackOff)
**Fix**: Verify image exists in registry and imagePullPolicy is correct

**Issue**: Database connection fails
**Fix**: Check ephemeral DB is running: `kubectl --context dev-serve get pods -n <namespace> -l app=<database-type>-<label>`

**Issue**: Service not accessible
**Fix**: Verify IngressRoute and Service are created: `kubectl --context dev-serve get ingressroute,svc -n <namespace>`

## Related Documentation

- [Deployment Subskill](deployment.md) - Deploy onboarded applications
- [Debugging Subskill](debugging.md) - Debug deployment issues
- [Validation Subskill](validation.md) - Validate configurations
- [Config Checklist](../references/config-checklist.md) - Complete configuration reference

## Automation Tools

**Go Foundation V2**: Auto-generates manifests for Golang applications
- Documentation: https://idocs.razorpay.com/platform/dev-productivity/go-foundation/v2/#devstack
- Use for Golang services to skip manual chart creation

## Adding Ephemeral Resources to Existing Applications

If an application is already onboarded to devstack but needs ephemeral resources (database, cache, queues) added:

### Step 1: Identify Existing Chart

1. **Locate the chart in helmfile/charts/ directory**:
   ```bash
   # Navigate to kube-manifests repo
   cd <kube-manifests-repo>

   # Verify chart exists in correct location
   ls helmfile/charts/<service-name>/
   ```

2. **Read current configuration**:
   ```bash
   cat helmfile/charts/<service-name>/values.yaml
   cat helmfile/charts/<service-name>/templates/deployment.yaml
   ```

### Step 2: Update values.yaml

Add ephemeral resource flags and configuration:

```yaml
# Add to existing values.yaml

# Enable ephemeral database
ephemeral_db: true

database:
  type: mysql  # or postgres
  name: <service-name>
  namespace: <service-name>
  username: <service-name>
  password: <auto-generated-password>
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database
  version: ""

# Or enable ephemeral cache
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

### Step 3: Create New Template Files

Add these files to `helmfile/charts/<service-name>/templates/`:

**For Ephemeral Database**:
- `db-configmap.yaml` (see [ephemeral-db.md](../references/templates/ephemeral-db.md))
- `db-configurator.yaml` (see [ephemeral-db.md](../references/templates/ephemeral-db.md))

**For Ephemeral Cache**:
- `cache-configmap.yaml` (similar to db-configmap.yaml)
- `cache-configurator.yaml` (similar to db-configurator.yaml)

**For Secret Management**:
- `secret-cloner.yaml` (if not already present)
- `sec-updater-cm.yaml` (update with DB/cache credentials)
- `sec-updater.yaml` (if not already present)

### Step 4: Update Existing deployment.yaml

Modify the secret mounting section to include ephemeral secrets:

**Find this section**:
```yaml
envFrom:
  - secretRef:
      name: {{ .Values.name }}
      optional: false
```

**Replace with**:
```yaml
envFrom:
  - secretRef:
      # Use base secret when: devstack_label is "base" OR secret_cloner_enabled is false
      {{- if or (eq .Values.devstack_label "base") (not .Values.secret_cloner_enabled) }}
      name: {{ .Values.secret_name }}
      {{- else }}
      name: {{ .Values.secret_name }}-{{ .Values.devstack_label }}
      {{- end }}
      optional: false
```

### Step 5: Configure Secret Updates

Update `sec-updater-cm.yaml` to include credentials for all enabled ephemeral resources. The condition must be `{{- if .Values.secret_cloner_enabled }}` — same flag controls both secret-cloner and sec-updater.

**If `ephemeral_db: true`** — add DB credentials:

```yaml
updateEntries:
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
```

**If `ephemeral_sqs: true` or `ephemeral_sns: true`** — add localstack AWS credentials:

```yaml
  aws1:
    key: AWS_REGION
    value: ap-south-1
  aws2:
    key: AWS_ACCESS_KEY_ID
    value: test
  aws3:
    key: AWS_SECRET_ACCESS_KEY
    value: test
```

> ⚠️ **Action required — verify secret key names with the user**
>
> The `key:` values above are defaults. **Ask the user to confirm the exact environment variable names their application reads**, and update the `key:` fields to match before applying.
>
> Example prompt:
> _"What environment variable names does `<service-name>` use for the database host, name, username, password, and URL? (e.g. `DATABASE_URL`, `MYSQL_HOST`, `DB_PASS`) I'll update the secret keys to match."_

Both blocks can coexist in the same `updateEntries` map if multiple resources are enabled.

### Step 6: Add Base Secrets

1. **Add database credentials to credstash**:
   - URL: https://credstash-ui.concierge.stage.razorpay.in/dist/
   - Table: `kubestash-dev-serve`
   - Keys to add:
     - `<namespace>/<secret-name>/DB_HOST` (will be overridden for ephemeral)
     - `<namespace>/<secret-name>/DB_NAME` (will be overridden for ephemeral)
     - `<namespace>/<secret-name>/DB_USERNAME` (will be overridden for ephemeral)
     - `<namespace>/<secret-name>/DB_PASSWORD` (will be overridden for ephemeral)
     - `<namespace>/<secret-name>/DB_PORT` (will be overridden for ephemeral)

2. **Wait for sync** (5 minutes) or verify:
   ```bash
   kubectl --context dev-serve get secret -n <namespace> <secret-name>
   ```

### Step 7: Test Changes

1. **Validate template rendering**:
   ```bash
   cd <kube-manifests-repo>/helmfile
   helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service>-<label> template
   ```

2. **Deploy and verify**:
   ```bash
   helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service>-<label> sync
   kubectl --context dev-serve get pods -n <namespace> -l devstack_label=<label>
   kubectl --context dev-serve get pods -n <namespace> -l app=<database-type>-<label>  # Check DB pod
   ```

3. **Check application logs**:
   ```bash
   kubectl --context dev-serve logs -f <app-pod-name> -n <namespace>
   # Should show successful database connection
   ```

### Common Modifications

#### Adding Only Database

Files to create/modify:
- ✅ `values.yaml` - Add `ephemeral_db: true` and `database:` config
- ✅ `templates/db-configmap.yaml` - NEW
- ✅ `templates/db-configurator.yaml` - NEW
- ✅ `templates/deployment.yaml` - Modify `envFrom` section
- ✅ `templates/secret-cloner.yaml` - Create if missing
- ✅ `templates/sec-updater-cm.yaml` - Create/update with DB credentials
- ✅ `templates/sec-updater.yaml` - Create if missing

#### Adding Only Cache

Files to create/modify:
- ✅ `values.yaml` - Add `ephemeral_cache: true` and `cache:` config
- ✅ `templates/cache-configmap.yaml` - NEW
- ✅ `templates/cache-configurator.yaml` - NEW
- ✅ `templates/deployment.yaml` - Modify `envFrom` section
- ✅ `templates/secret-cloner.yaml` - Create if missing
- ✅ `templates/sec-updater-cm.yaml` - Create/update with Redis host
- ✅ `templates/sec-updater.yaml` - Create if missing

#### Adding SQS Queues

Files to create/modify:
- ✅ `values.yaml` - Add `configurator.sqs: true` and `queues:` config
- ✅ `templates/sqs-configmap.yaml` - NEW
- ✅ `templates/sqs-configurator.yaml` - NEW
- ✅ `templates/secret-cloner.yaml` - Create if missing (update condition to cover `ephemeral_sqs`)
- ✅ `templates/sec-updater-cm.yaml` - Create/update with localstack AWS credentials (`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- ✅ `templates/sec-updater.yaml` - Create if missing

**Note**: SQS configurator automatically updates the secret with queue URLs using the `secretKey` field. The localstack AWS credentials must also be injected so the application can connect to localstack.

#### Adding SNS Topics

Files to create/modify:
- ✅ `values.yaml` - Add `configurator.sns: true` and `topics:` config
- ✅ `templates/sns-configmap.yaml` - NEW
- ✅ `templates/sns-configurator.yaml` - NEW
- ✅ `templates/secret-cloner.yaml` - Create if missing (update condition to cover `ephemeral_sns`)
- ✅ `templates/sec-updater-cm.yaml` - Create/update with localstack AWS credentials (`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- ✅ `templates/sec-updater.yaml` - Create if missing

**Important Considerations**:
- SNS topics publish to SQS queues, so ensure subscription queues exist
- Hook weight is `4` (runs after SQS configurator which has weight `3`)
- Each topic can have multiple SQS subscriptions (fan-out pattern)
- Topic ARNs are stored in secrets using the `secret_name` field

**Hook Execution Order for SNS**:
1. `pre-install` weight `2` - SNS ConfigMap created
2. `post-install` weight `4` - SNS configurator runs (after SQS)

#### Adding Multiple Resources

When adding multiple resources (database + cache + SQS + SNS):
- Create all resource-specific templates
- `secret-cloner.yaml` and `sec-updater-cm.yaml` outer condition: `{{- if .Values.secret_cloner_enabled }}`
- Inside `sec-updater-cm.yaml` `updateEntries`, use inner guards:
  - `{{- if .Values.ephemeral_db }}` → DB credentials (`DB_HOST`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`, `DB_URL`)
  - `{{- if or .Values.ephemeral_sqs .Values.ephemeral_sns }}` → AWS credentials (`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- `deployment.yaml` envFrom secret name: `{{- if or (eq .Values.devstack_label "base") (not .Values.secret_cloner_enabled) }}` → `{{ .Values.secret_name }}` `{{- else }}` → `{{ .Values.secret_name }}-{{ .Values.devstack_label }}` `{{- end }}`
- For SNS+SQS: Create SQS queues first (subscription endpoints), then SNS topics
- Consider hook weights to ensure proper order: DB/Cache/SQS (weight 3) → SNS (weight 4)

### Troubleshooting

**Issue**: Ephemeral database pod not starting

**Debug**:
```bash
kubectl --context dev-serve get pods -n db-configurator -l app=dbc-<service>-<label>
kubectl --context dev-serve logs -n db-configurator dbc-<service>-<label>-xxxxx
kubectl --context dev-serve describe job -n db-configurator dbc-<service>-<label>
```

**Issue**: Application can't connect to database

**Fix**: Check secret was updated correctly:
```bash
kubectl --context dev-serve get secret -n <namespace> <service>-<label> -o yaml
# Verify DB_HOST, DB_NAME, DB_USERNAME, DB_PASSWORD are present
```

**Issue**: Helm hook failures

**Fix**: Check hook weights are correct:
```
secret-cloner: hook-weight: "1"
db-configmap: hook-weight: "2"
db-configurator: hook-weight: "3"
sqs-configmap: hook-weight: "2"
sqs-configurator: hook-weight: "3"
sns-configmap: hook-weight: "2"
sns-configurator: hook-weight: "4"
sec-updater-cm: hook-weight: "4"
sec-updater: hook-weight: "5"
```

**Issue**: SNS configurator fails with subscription errors

**Debug**:
```bash
kubectl --context dev-serve get pods -n sns-configurator -l name=sns-<service>-<label>
kubectl --context dev-serve logs -n sns-configurator sns-<service>-<label>-xxxxx
```

**Common Causes**:
- SQS queue (subscription endpoint) doesn't exist
- Queue name mismatch in subscription endpoint
- SNS configurator ran before SQS configurator (check hook weights)

**Fix**:
1. Verify SQS queues exist:
   ```bash
   aws --endpoint-url=http://localstack.localstack.svc.cluster.local:4566 sqs list-queues
   ```
2. Check subscription queue names match in both configs
3. Ensure SNS hook weight (4) is greater than SQS hook weight (3)

**Issue**: SNS topics not receiving messages

**Debug**:
```bash
# Check topic exists
aws --endpoint-url=http://localstack.localstack.svc.cluster.local:4566 sns list-topics

# Check subscriptions
aws --endpoint-url=http://localstack.localstack.svc.cluster.local:4566 sns list-subscriptions-by-topic --topic-arn <arn>

# Check if messages are going to DLQ
aws --endpoint-url=http://localstack.localstack.svc.cluster.local:4566 sqs receive-message --queue-url <dlq-url>
```

**Fix**:
- Verify subscription endpoints are correct in sns-configmap.yaml
- Check protocol is set to `sqs`
- Ensure application is publishing to correct topic ARN

### Quick Checklist

When adding ephemeral resources to existing app:

- [ ] Update `values.yaml` with ephemeral config
- [ ] Create resource configmap (db/cache/sqs/sns)
- [ ] Create resource configurator job
- [ ] `deployment.yaml` envFrom: uses `{{ .Values.secret_name }}` when base label or `secret_cloner_enabled: false`; else `{{ .Values.secret_name }}-{{ .Values.devstack_label }}`
- [ ] Create/update `secret-cloner.yaml` — condition must cover all four ephemeral flags
- [ ] Create/update `sec-updater-cm.yaml`:
  - DB credentials (`DB_HOST`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`, `DB_URL`) if `ephemeral_db`
  - Localstack AWS credentials (`AWS_REGION=ap-south-1`, `AWS_ACCESS_KEY_ID=test`, `AWS_SECRET_ACCESS_KEY=test`) if `ephemeral_sqs` or `ephemeral_sns`
- [ ] Create/update `sec-updater.yaml`
- [ ] For SNS: Ensure subscription SQS queues exist before creating topics
- [ ] Add base secrets to credstash (if needed)
- [ ] Validate template rendering
- [ ] Deploy and test connectivity

## Best Practices

1. **Start minimal**: Begin with basic deployment + service, add ephemeral resources as needed
2. **Use ephemeral resources for development**: Keep RDS/Redis for base/prod only
3. **Set appropriate TTLs**: `1h` for quick testing, `8h` for active development, `forever` for base
4. **Resource limits**: Start conservative (50m CPU, 200Mi memory), scale as needed
5. **Probes**: Use simple `/health` endpoints, avoid complex dependency checks in liveness
6. **Secrets**: Never hardcode secrets in charts, always use credstash
7. **Node selectors**: Use `worker-generic` for compute, `worker-database` for databases
8. **Naming**: Keep deployment names consistent with production (suffix with label only)

## Quick Reference Commands

```bash
# Navigate to kube-manifests repo
cd <kube-manifests-repo>/helmfile

# Create chart structure (ALWAYS in helmfile/charts/)
mkdir -p charts/<service>/templates

# Validate chart
helm lint charts/<service>/

# Test template rendering
helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service>-<label> template

# Deploy
helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service>-<label> sync

# Check status
kubectl --context dev-serve get pods,svc,ingressroute -n <namespace> -l devstack_label=<label>

# View logs
kubectl --context dev-serve logs -f <pod-name> -n <namespace>

# Delete deployment
helmfile --kube-context dev-serve -f helmfile.yaml -l name=<service>-<label> delete
```

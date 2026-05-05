# Ephemeral Maxwell CDC - SOP

> Bring up Maxwell to read CDC events from your ephemeral database

**Cluster:** dev-serve | **Namespace:** maxwell-microservices | **Last updated:** March 2026

---

## Overview

When you deploy a service with an ephemeral database on dev-serve (e.g., pg-router with `ephemeral_db=true`), your database changes are isolated to your ephemeral MySQL instance. To get CDC (Change Data Capture) events flowing to Kafka from your ephemeral database, you need an ephemeral Maxwell instance.

### What You Get

Every ephemeral Maxwell instance automatically:
- Connects to your ephemeral MySQL via binlog replication
- Produces CDC events to dev-serve MSK (`devserve-kafka-msk.np.razorpay.vpc:9094`)
- Adds Kafka headers: `DEVSTACK_LABEL=<your_label>` and `Async-Target-Consumer=<your_label>`
- Gets cleaned up automatically when you tear down your deployment

### Prerequisites

- MaxwellInstance CRD installed on dev-serve (already done)
- Maxwell operator running in `maxwell-microservices` namespace (already done)
- Your ephemeral MySQL is up and accessible within the cluster
- `kubectl` access to dev-serve cluster

---

## Setup: Add Maxwell CDC Template to Your Helm Chart

To enable automatic Maxwell creation for your service, you need to add a `MaxwellInstance` CR template to your service's helm chart. This is a **one-time setup** per service.

### Step 1: Add maxwell defaults to `values.yaml`

Add the following to your chart's `values.yaml`:

```yaml
# Maxwell CDC (auto-created when ephemeral_db=true and devstack_label != base)
maxwell:
  image: "c.rzp.io/razorpay/maxwell-microservice:maxwell-c85bb0f1b196a61ea1a5699bd1e7e6b65306b54c"
  dbName: <your_db_name>                    # e.g., pg_router_orders
  metaFilePath: <your_meta_file_path>       # e.g., db_metadata/maxwell/stage/meta_pg_router_order.json
  maxwellType: legacy_microservice          # or "microservice"
  kafkaTopic: ""                            # leave empty for default template
```

### Step 2: Create `templates/maxwell-cdc.yaml`

Create a new template file in your chart's `templates/` directory:

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

Replace `<service>`, `<your_db_name>`, `<db_user>`, and `<your_meta_file_path>` with your service-specific values.

**Reference implementation:** See [pg-router maxwell-cdc.yaml](https://github.com/razorpay/kube-manifests/blob/master/helmfile/charts/pg-router/templates/maxwell-cdc.yaml)

### How the template works

| Condition | Result |
|---|---|
| `ephemeral_db=false` | Template not rendered, no Maxwell created |
| `devstack_label=base` | Template not rendered (base uses static deployments) |
| `ephemeral_db=true` + `devstack_label=aditya` | Creates `MaxwellInstance` CR, operator creates Maxwell Deployment |

The key fields are auto-derived:
- **`devstackLabel`** → from `{{ .Values.devstack_label }}` (same as your service)
- **`replication.host`** → `mysql-{{ .Values.devstack_label }}.{{ .Values.namespace }}.svc.cluster.local` (your ephemeral MySQL service)
- **`replication.user/password`** → from your database config in values

---

## Deploy: Bring Up Ephemeral Maxwell

Once the helm chart template is added (one-time setup above), Maxwell is created automatically every time you deploy with `ephemeral_db=true`.

### Step 1: Set helmfile values

In your helmfile, ensure these values are set:

```yaml
- name: pg-router-{{ .Values.devstack_label }}
  namespace: pg-router
  chart: ./charts/pg-router
  values:
    - image: <commit_sha>
    - devstack_label: {{ .Values.devstack_label }}
    - ttl: {{ .Values.ttl }}
    - ephemeral_db: true
    - secret: {{ .Values.secret }}
```

### Step 2: Deploy via helmfile

```bash
helmfile -e default \
  --state-values-set devstack_label=aditya,ttl=8h,secret=<secret> \
  sync --selector name=pg-router-aditya
```

### Step 3: Verify

```bash
# Check CR was created
kubectl -n maxwell-microservices get maxwellinstances

# Expected output:
# NAME                  LABEL    DB                HOST                                         STATUS   AGE
# pg-router-aditya      aditya   pg_router_orders  mysql-aditya.pg-router.svc.cluster.local              10s

# Check Maxwell deployment was created by operator
kubectl -n maxwell-microservices get deployment maxwell-pg-router-aditya

# Check pod is running
kubectl -n maxwell-microservices get pods -l name=maxwell-pg-router-aditya

# Check Kafka headers are configured
kubectl -n maxwell-microservices logs -l name=maxwell-pg-router-aditya -c maxwell | grep 'Static headers'
# Expected: Static headers configured: [DEVSTACK_LABEL, Async-Target-Consumer]
```

### Step 4: Cleanup

When you tear down your service via `helmfile destroy`, the MaxwellInstance CR is deleted, and the operator automatically cleans up the Maxwell Deployment.

Or manually:
```bash
kubectl -n maxwell-microservices delete maxwellinstance pg-router-aditya
```

---

## MaxwellInstance CR Reference

| Field | Required | Default | Description |
|---|---|---|---|
| `spec.devstackLabel` | Yes | - | Your devstack label (must NOT be `base`) |
| `spec.dbName` | Yes | - | Maxwell DB_NAME (e.g., `pg_router_orders`) |
| `spec.replication.host` | Yes | - | MySQL host (your ephemeral DB service) |
| `spec.replication.user` | Yes | - | MySQL replication user |
| `spec.replication.password` | No | - | MySQL password (plain text) |
| `spec.metaFilePath` | Yes | - | Metadata JSON path in Maxwell image |
| `spec.kafkaTopic` | No | template | Kafka topic override |
| `spec.maxwellType` | No | `legacy_microservice` | `microservice` or `legacy_microservice` |
| `spec.filterFilePath` | No | `filter_enhanced.js` | JS filter file |
| `spec.image` | No | latest fixed image | Maxwell docker image override |

---

## Kafka Headers

Every message produced by ephemeral Maxwell includes these Kafka record headers:

| Header Key | Value | Purpose |
|---|---|---|
| `DEVSTACK_LABEL` | `<your_label>` | Identifies which devstack instance produced the event |
| `Async-Target-Consumer` | `<your_label>` | Routes async consumers to the correct devstack |

Consumers can filter messages by header to only process events from their devstack instance.

### Verifying Headers

Kafdrop may not display headers. Use the Java consumer from inside the Maxwell pod:

```bash
kubectl -n maxwell-microservices exec <pod> -c maxwell -- bash
# Then compile and run CheckHeaders.java to verify headers
```

---

## Troubleshooting

### Maxwell pod not created after deploying

```bash
# Check operator logs
kubectl -n maxwell-microservices logs -l app=maxwell-operator --tail=20

# Check if operator is running
kubectl -n maxwell-microservices get pods -l app=maxwell-operator
```

### Maxwell pod in CrashLoopBackOff

```bash
kubectl -n maxwell-microservices logs <pod> -c maxwell --tail=30
```

Common causes:
- **MySQL connection refused** — ephemeral DB not ready or wrong host
- **TopicAuthorizationException** — Kafka topic ACL missing on dev-serve MSK
- **Missing metadata JSON** — wrong `metaFilePath` or image doesn't contain the file

### CR rejected by Kyverno

Ensure your CR has `devstack_label` in `metadata.labels` and `janitor/ttl` in annotations.

### DEVSTACK_LABEL not set error

The Maxwell pod requires `DEVSTACK_LABEL` env var. If you see this error, check the operator logs and restart it:

```bash
kubectl -n maxwell-microservices rollout restart deployment maxwell-operator
```

---

## Important Notes

> **`devstackLabel: base` is reserved.** The `base` label is used by the 55 static Maxwell deployments that replicate stage RDS. The operator will reject any CR with `devstackLabel: base`. Use your own label (e.g., your name or ticket ID).

- Ephemeral Maxwell deployments have `janitor/ttl` and will be auto-cleaned after expiry
- Both base and ephemeral Maxwell write to the same Kafka topics — use `DEVSTACK_LABEL` header to filter
- The operator runs in `maxwell-microservices` namespace and watches `MaxwellInstance` CRs only in that namespace
- Deleting the CR automatically cleans up the Maxwell Deployment (via ownerReference)

---

## Support

- Slack: `#platform-devstack`
- Maxwell microservice PR (header fix): [PR #80](https://github.com/razorpay/maxwell-microservice/pull/80)
- Kube-manifests Maxwell chart: [PR #24759](https://github.com/razorpay/kube-manifests/pull/24759)
- pg-router integration: [PR #24735](https://github.com/razorpay/kube-manifests/pull/24735)

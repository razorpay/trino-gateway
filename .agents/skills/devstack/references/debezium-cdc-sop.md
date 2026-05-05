# Ephemeral Debezium CDC - SOP

> Bring up Debezium to read CDC events from your ephemeral PostgreSQL database

**Cluster:** dev-serve | **Namespace:** debezium | **Last updated:** April 2026

---

## Overview

When you deploy a service with an ephemeral PostgreSQL database on dev-serve (e.g., optimizer-core with `ephemeral_db=true`), your database changes are isolated to your ephemeral Postgres instance. To get CDC (Change Data Capture) events flowing to Kafka from your ephemeral database, you need an ephemeral Debezium connector.

### What You Get

Every ephemeral Debezium connector automatically:
- Connects to your ephemeral PostgreSQL via logical replication
- Produces CDC events to dev-serve MSK (`devserve-kafka-msk.np.razorpay.vpc:9094`)
- Adds Kafka headers: `DEVSTACK_LABEL=<your_label>` and `Async-Target-Consumer=<your_label>`
- Gets cleaned up automatically when you tear down your deployment (finalizer-based)

### Prerequisites

- DebeziumInstance CRD installed on dev-serve (already done)
- Debezium operator running in `debezium` namespace (already done)
- Kafka Connect running in `debezium` namespace (already done)
- Your ephemeral PostgreSQL is up and accessible within the cluster
- PostgreSQL has logical replication enabled and `debezium_publication` created (handled by db-configurator)
- `kubectl` access to dev-serve cluster

---

## Setup: Add Debezium CDC Template to Your Helm Chart

To enable automatic Debezium connector creation for your service, you need to add a `DebeziumInstance` CR template to your service's helm chart. This is a **one-time setup** per service.

### Step 1: Add debezium defaults to `values.yaml`

Add the following to your chart's `values.yaml`:

```yaml
# Debezium CDC (auto-created when ephemeral_db=true, ephemeral_debezium=true, devstack_label != base)
ephemeral_debezium: false
# debezium_topicPrefix: ""        # optional: override topic prefix (default: {dbName}-{devstackLabel})
# debezium_tablesInclude: ""      # optional: comma-separated table include filter
# debezium_tablesExclude: ""      # optional: comma-separated table exclude filter
```

### Step 2: Create `templates/debezium-cdc.yaml`

Create a new template file in your chart's `templates/` directory:

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

Replace `database.db1_name`, `database.namespace`, `database.db1_username`, and `database.db1_secret` with your service-specific values references.

**Reference implementation:** See [optimizer-core debezium-cdc.yaml](https://github.com/razorpay/kube-manifests/blob/master/helmfile/charts/optimizer-core/templates/debezium-cdc.yaml)

### How the template works

| Condition | Result |
|---|---|
| `ephemeral_db=false` | Template not rendered, no connector created |
| `ephemeral_debezium=false` | Template not rendered |
| `devstack_label=base` | Template not rendered (base uses static connectors) |
| `ephemeral_db=true` + `ephemeral_debezium=true` + `devstack_label=aditya` | Creates `DebeziumInstance` CR, operator registers connector on Kafka Connect |

The key fields are auto-derived:
- **`devstackLabel`** → from `{{ .Values.devstack_label }}` (same as your service)
- **`replication.host`** → `postgres-{{ .Values.devstack_label }}.{{ .Values.database.namespace }}.svc.cluster.local` (your ephemeral PostgreSQL service)
- **`topicPrefix`** → defaults to `{dbName}-{devstackLabel}` (auto-generated)
- **`slotName`** → defaults to `dbz_{dbName}_{devstackLabel}` (auto-generated)

---

## Deploy: Bring Up Ephemeral Debezium

Once the helm chart template is added (one-time setup above), the Debezium connector is created automatically every time you deploy with `ephemeral_db=true` and `ephemeral_debezium=true`.

### Step 1: Set helmfile values

In your helmfile, ensure these values are set:

```yaml
- name: optimizer-core-{{ .Values.devstack_label }}
  namespace: optimizer-core
  chart: ./charts/optimizer-core
  values:
  - image: <commit_sha>
  - devstack_label: {{ .Values.devstack_label }}
  - ttl: {{ .Values.ttl }}
  - ephemeral_db: true
  - ephemeral_debezium: true
  - secret: {{ .Values.secret }}
```

### Step 2: Deploy via helmfile

```bash
helmfile -e default \
  --state-values-set devstack_label=aditya,ttl=8h,secret=<secret> \
  sync --selector name=optimizer-core-aditya
```

### Step 3: Verify

```bash
# Check CR was created
kubectl --context dev-serve -n debezium get debeziuminstances

# Expected output:
# NAME                      LABEL    DB               HOST                                                       STATUS    AGE
# optimizer-core-aditya     aditya   optimizer_core   postgres-aditya.optimizer-core.svc.cluster.local            Running   30s

# Check CR status details
kubectl --context dev-serve -n debezium describe debeziuminstance optimizer-core-aditya

# Check connector on Kafka Connect (via port-forward)
kubectl --context dev-serve port-forward -n debezium svc/kafka-connect 18083:8083 &
curl -s http://localhost:18083/connectors/dbz-optimizer_core-aditya/status | python3 -m json.tool
# Expected: connector.state = RUNNING, tasks[0].state = RUNNING
```

### Step 4: Cleanup

When you tear down your service via `helmfile destroy`, the DebeziumInstance CR is deleted, and the operator automatically cleans up the connector via finalizer.

Or manually:
```bash
kubectl --context dev-serve -n debezium delete debeziuminstance optimizer-core-aditya
```

---

## DebeziumInstance CR Reference

| Field | Required | Default | Description |
|---|---|---|---|
| `spec.devstackLabel` | Yes | - | Your devstack label (must NOT be `base`) |
| `spec.dbName` | Yes | - | PostgreSQL database name to capture changes from |
| `spec.replication.host` | Yes | - | PostgreSQL host (your ephemeral DB service) |
| `spec.replication.port` | No | `5432` | PostgreSQL port |
| `spec.replication.user` | Yes | - | PostgreSQL user with REPLICATION privilege |
| `spec.replication.password` | Yes | - | PostgreSQL password |
| `spec.topicPrefix` | No | `{dbName}-{devstackLabel}` | Kafka topic prefix for CDC events |
| `spec.slotName` | No | `dbz_{dbName}_{devstackLabel}` | PostgreSQL replication slot name |
| `spec.pluginName` | No | `pgoutput` | Logical decoding plugin |
| `spec.kafkaConnectHost` | No | `kafka-connect.debezium.svc.cluster.local:8083` | Kafka Connect REST API endpoint |
| `spec.tablesInclude` | No | all tables | Comma-separated list of tables to include |
| `spec.tablesExclude` | No | none | Comma-separated list of tables to exclude |

---

## Kafka Headers

Every CDC event produced by an ephemeral Debezium connector includes these Kafka record headers:

| Header Key | Value | Purpose |
|---|---|---|
| `DEVSTACK_LABEL` | `<your_label>` | Identifies which devstack instance produced the event |
| `Async-Target-Consumer` | `<your_label>` | Routes async consumers to the correct devstack |

Consumers can filter messages by header to only process events from their devstack instance.

### Debezium CDC Connection

```
Bootstrap Servers: devserve-kafka-msk.np.razorpay.vpc:9094
Kafka Headers: DEVSTACK_LABEL=<devstack_label>, Async-Target-Consumer=<devstack_label>
Topic Pattern: <topicPrefix>.<schema>.<table>
Connector Name: dbz-<dbName>-<devstackLabel>
```

---

## Troubleshooting

### Connector not created after deploying

```bash
# Check operator logs
kubectl --context dev-serve -n debezium logs deploy/debezium-operator --tail=20

# Check if operator is running
kubectl --context dev-serve -n debezium get pods -l app=debezium-operator
```

### CR stuck in Pending or Failed

```bash
kubectl --context dev-serve -n debezium describe debeziuminstance <name>
# Check .status.message for error details
```

Common causes:
- **Kafka Connect unreachable** — check `kafka-connect` pod: `kubectl -n debezium get pods | grep kafka-connect`
- **`publication does not exist`** — run `CREATE PUBLICATION debezium_publication FOR ALL TABLES;` as superuser on the PG database (normally handled by db-configurator)
- **`TopicAuthorizationException`** — Kafka topic ACL missing on dev-serve MSK
- **Slot name conflict** — replication slots use `dbz_{dbName}_{label}`, ensure unique devstack labels
- **PostgreSQL connection refused** — ephemeral DB not ready, or wrong host in CR

### CR rejected by Kyverno

Ensure your CR has `devstack_label` in `metadata.labels` and `janitor/ttl` in annotations.

### Check connector status directly

```bash
kubectl --context dev-serve port-forward -n debezium svc/kafka-connect 18083:8083 &

# List all connectors
curl -s http://localhost:18083/connectors | python3 -m json.tool

# Check specific connector
curl -s http://localhost:18083/connectors/dbz-optimizer_core-aditya/status | python3 -m json.tool

# Restart failed connector
curl -s -X POST http://localhost:18083/connectors/dbz-optimizer_core-aditya/restart
```

---

## Important Notes

> **`devstackLabel: base` is reserved.** The `base` label is used by the 12 static Debezium connectors that replicate stage databases. The operator will not create ephemeral connectors with `devstackLabel: base`. Use your own label (e.g., your name or ticket ID).

- Ephemeral Debezium connectors have `janitor/ttl` and will be auto-cleaned after expiry
- Both base and ephemeral connectors write to the same Kafka topics — use `DEVSTACK_LABEL` header to filter
- The operator runs in `debezium` namespace and watches `DebeziumInstance` CRs only in that namespace
- Deleting the CR automatically cleans up the connector on Kafka Connect (via finalizer)
- Unlike Maxwell (MySQL binlog), Debezium uses PostgreSQL logical replication which requires the `debezium_publication` to be pre-created

---

## Support

- Slack: `#platform-devstack`
- Debezium Operator SOP: [README.md](https://github.com/razorpay/kube-manifests/blob/master/tools/hooks/debezium_operator/README.md)
- Kube-manifests Debezium operator: [PR #25031](https://github.com/razorpay/kube-manifests/pull/25031)
- Kube-manifests Debezium bugfixes: [PR #25168](https://github.com/razorpay/kube-manifests/pull/25168)

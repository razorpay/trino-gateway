---
name: baseline-alerting
description: >-
  Use when creating, updating, or reviewing baseline Prometheus alert rules
  for a Razorpay microservice. Triggered by requests like 'add baseline alerts
  for X service', 'set up monitoring alerts', 'create baseline alerting for my
  service'.
---

# Baseline Alerting Skill

Generate standardized Prometheus alert rules for a Razorpay microservice. This skill produces a complete set of baseline alerts covering pod health, ingress/egress traffic, latency SLAs, Kafka consumer lag, AWS resource monitoring, and more — then places them in the `alert-rules` repository as a PR.

**Skill directory** is referred to as `${CLAUDE_SKILL_DIR}` throughout. It is the directory containing this `SKILL.md` file.

## Prerequisites

- Access to the `alert-rules` repository (https://github.com/razorpay/alert-rules). Ask the user for the correct URL if unsure.
- Python 3.8+ (for rendering templates)
- `promtool` — install if not available:
  - macOS: `brew install prometheus`
  - Linux: download binary from https://github.com/prometheus/prometheus/releases (extract, add to PATH)
- `git` and `gh` CLI (for PR creation)

---

## Workflow Overview

Execute in 6 sequential phases. **Each phase runs as a sub-agent** to keep context focused. The parent agent coordinates, passing outputs between phases.

| Phase | Name | Purpose |
|-------|------|---------|
| 1 | Reconnaissance | Find existing alerts, determine CREATE vs UPDATE |
| 2 | Service Analysis | Detect surfaces from service code |
| 3 | Input Generation | Build or update `{app_name}.input.yaml` |
| 4 | Render + Dedup | Render templates, remove duplicate alerts |
| 5 | Place + Validate | Copy to final path, validate YAML/promtool |
| 6 | PR Creation | Branch, commit, push, open PR |

---

## Phase 1: Reconnaissance

**Goal:** Inventory existing alerts for the target service. Determine if this is a CREATE or UPDATE flow.

Spawn a sub-agent that:

1. Clones or accesses the `alert-rules` repository.
2. Searches `rules/prod-rules/` for existing alert rules matching the target service. Use these search patterns:
   - Files with the service name in the filename
   - YAML files containing `service: {app_name}` in labels
   - YAML files containing `namespace="{namespace}"` in PromQL expressions. Namespace is usually the app_name or some version of it.
   - YAML files with `[{BU}][{appName}]` in alert names
3. Builds a **dedup inventory** — for each found alert, record:
   - Alert name (the `alert:` field value)
   - Core PromQL metric (e.g., `container_memory_working_set_bytes`, `http_requests_total`, `aws_rds_cpuutilization_maximum`)
   - Namespace or service filter used in the expression
   - Threshold value and comparison operator
4. Checks if `baseline-monitoring-inputs/{app_name}.input.yaml` exists in the alert-rules repo:
   - **Exists** → this is an UPDATE flow
   - **Does not exist** → this is a CREATE flow

**Output:** Dedup inventory (list of existing alerts) + CREATE/UPDATE determination.

---

## Phase 2: Service Analysis

**Goal:** Detect which alert surfaces apply to this service.

Spawn a sub-agent that:

1. Analyzes the service repository or PR provided by the user.
2. Makes a **best-effort** detection of surfaces from the service codebase. Do not try too hard — if a signal isn't obvious, skip it and ask the user instead.

**Auto-detectable surfaces** (look for obvious signals in the service repo):

| Surface | Detection Signals |
|---------|-------------------|
| **REST** | HTTP route definitions, handler registrations, `http.HandleFunc`, Express routes, Spring `@RequestMapping`, `gin.Engine`, `chi.Router` |
| **gRPC** | `.proto` files, gRPC server initialization, `grpc.NewServer()`, `RegisterXxxServer()` |
| **Kafka** | Kafka consumer/producer config, topic names in config files, `kafka.ConsumerGroup`, `@KafkaListener`. **Note:** you can detect that Kafka is used, but exact topic names may need user confirmation. |
| **Outbox** | Outbox pattern imports (Go: `outbox` package usage), outbox worker config |
| **Anomaly** | Suggest for all REST services (week-over-week comparison) |

**Always ask the user**

| Surface | What to ask for |
|---------|-----------------|
| **Traefik** | Traefik service names (IngressRoute manifests live in infra repos) |
| **Edge/Kong** | Kong edge service name (edge config lives in separate repo) |
| **ASG/Node** | ASG names and k8s cluster names (terraform/infra config) |
| **AWS RDS** | RDS instance identifiers (terraform-managed, not in app code) |
| **AWS ElastiCache** | ElastiCache cluster IDs (terraform-managed) |
| **AWS SQS** | SQS queue names (terraform-managed) |
| **StatusCake** | StatusCake test ID (configured externally) |

The core philosophy being don't guess anything. If unsure, ask the user. Apply this to other unknown values too


3. Presents detection results **and asks for missing info** from the user:

```
Detected surfaces for {app_name}:
- REST: YES (found HTTP handlers in cmd/server.go)
- gRPC: NO
- Kafka: LIKELY (found kafka consumer config — confirm topic names below)
- Outbox: YES (outbox package imported in internal/worker/)
- Anomaly: SUGGESTED (REST service — week-over-week comparison recommended)

I need the following from you (resource names live in infra/terraform, not in service code):
- Traefik service names: (or "none")
- Edge/Kong service name: (or "none")
- ASG names: (or "none")
- K8s cluster names: (clusters your service runs on, or "none")
- RDS instance identifiers: (or "none")
- ElastiCache cluster IDs: (or "none")
- SQS queue names: (or "none")
- Kafka topic names: (confirm detected topics, or provide exact list)
- StatusCake test ID: (or "none")

Also provide:
- BU (Business Unit): e.g., Platform, Payments, RazorpayX, Capital
- Pod/team name: e.g., platform-engg
- Slack channel: e.g., #my-team-alerts
- Slack handle: e.g., <!subteam^SXXXXXX>
- Runbook URL
- Grafana/Vajra dashboard base URL
- Metric prefix: e.g., myapp_ (must match your instrumentation code)
- K8s namespace
- Container names
```

4. **Wait for user confirmation before proceeding.** Do not guess missing values — especially traefik and edge details.

**Output:** Confirmed surface list + team metadata.

---

## Phase 3: Input Generation

**Goal:** Produce a complete `{app_name}.input.yaml` config file.

Spawn a sub-agent that:

### CREATE Flow (no existing input file)

1. Copies `${CLAUDE_SKILL_DIR}/inputs.template.yaml` as the starting point.
2. Fills in all confirmed values from Phase 2.
3. **Removes entire sections** for surfaces that do not apply. For example, if gRPC is not used, delete the `application.interface.grpc` field. If no Kafka topics, delete the `infra.kafka` block entirely. This keeps the config clean.
4. Ensures all REQUIRED fields (marked in the template comments) are populated.

### UPDATE Flow (existing input file found)

1. Reads the existing `baseline-monitoring-inputs/{app_name}.input.yaml`.
2. Compares its schema against `${CLAUDE_SKILL_DIR}/inputs.template.yaml` (source of truth).
3. If schema has changed:
   - Adds new fields. Ask the user for values you don't know
   - Preserves all existing user-customized values
   - Removes deprecated fields no longer in the template
4. Presents a migration diff to the user showing what changed.
5. Applies the user's new changes (e.g., adding a newly detected surface) on top.

### Both Flows

1. Presents the final input YAML to the user for review. If you're running in a cloud environment, like vyom, then attach this yaml file to the conversation as a file(usually slack) 
2. **Wait for user approval before saving.**
3. Saves to `baseline-monitoring-inputs/{app_name}.input.yaml` in the alert-rules repo.

**Output:** Path to saved input YAML.

---

## Phase 4: Render + Dedup

**Goal:** Render alert rules from templates, then remove alerts that already exist.

Spawn a sub-agent that:

1. Installs dependencies and runs the render script:
   ```bash
   pip install -r ${CLAUDE_SKILL_DIR}/scripts/requirements.txt
   python ${CLAUDE_SKILL_DIR}/scripts/render.py \
     --inputs {alert_rules_repo}/baseline-monitoring-inputs/{app_name}.input.yaml \
     --output /tmp/baseline-{app_name}.yaml
   ```

2. Reads the rendered output file.

3. Compares **each** rendered alert against the dedup inventory from Phase 1. Two alerts are duplicates if ALL of these match:
   - Same core metric in the PromQL `expr` (e.g., both use `container_memory_working_set_bytes`)
   - Same namespace/service filter (e.g., both filter on `namespace="my-service"`)
   - Same or stricter threshold (existing alert threshold is equal or tighter)

4. For each duplicate found, marks it for removal.

5. Presents a dedup report to the user:
   ```
   Baseline alerts generated: 25
   Duplicates found and removed: 3

   Removed (already exist in alert-rules repo):
   - "Pod memory usage > 75%" -- exists in rules/prod-rules/myapp_rules.yaml:15
   - "DB CPU > 80%" -- exists in rules/prod-rules/myapp_rds.yaml:8
   - "Consumer lag > 1000" -- exists in rules/prod-rules/myapp_kafka.yaml:22

   Final alert count: 22
   ```

6. **When in doubt about whether two alerts overlap, ask the user** rather than silently removing.

7. Writes the deduplicated output to `/tmp/baseline-{app_name}.yaml`.

**Output:** Deduplicated alert rules YAML + dedup report.

---

## Phase 5: Place + Validate

**Goal:** Place final YAML in the correct location and validate it.

Instruct the agent to:

1. Copy `/tmp/baseline-{app_name}.yaml` to `rules/prod-rules/baseline-{app_name}.yaml` in the alert-rules repo.

2. Install `promtool` if not already available:
   ```bash
   which promtool || brew install prometheus  # macOS
   # Linux: download from https://github.com/prometheus/prometheus/releases
   ```

3. Run validation:
   ```bash
   promtool check rules rules/prod-rules/baseline-{app_name}.yaml
   ```

4. If validation fails:
   - Show the specific error and which rule caused it
   - Diagnose the issue (common: bad indentation, invalid PromQL syntax, missing quotes)
   - Fix and re-validate
   - Repeat until clean

5. Present the final file to the user for review before proceeding to PR.

**Output:** Validated alert rules file at its final path.

---

## Phase 6: PR Creation

**Goal:** Create a pull request in the alert-rules repo.


1. Create a new branch:
   ```bash
   git checkout -b baseline-alerting/{app_name}-$(openssl rand -hex 3)
   ```

2. Stage both files:
   ```bash
   git add baseline-monitoring-inputs/{app_name}.input.yaml
   git add rules/prod-rules/baseline-{app_name}.yaml
   ```

3. Commit:
   ```bash
   git commit -m "feat(baseline-alerting): {app_name} <meaningful commit message for the change>"
   ```

4. Push and create PR:
   ```bash
   git push -u origin baseline-alerting/{app_name}
   gh pr create --title "feat(baseline-alerting): {add/modify} baseline alerts for {app_name}" --body "$(cat <<'EOF'
   ## Baseline Alerts for {app_name}

   ### Surfaces covered
   - [x/blank] Pod (CPU, memory, OOM, restarts, state, count)
   - [x/blank] REST Ingress (RPS high/low, 5xx rate, 4xx rate, non-2xx rate)
   - [x/blank] gRPC Ingress (RPS high/low, Internal error rate, InvalidArgument rate)
   - [x/blank] Egress (RPS per host, 5xx/4xx/non-2xx rates, P99/P95/P90 latency)
   - [x/blank] REST Latency (P99, P95, P90)
   - [x/blank] gRPC Latency (P99, P95, P90)
   - [x/blank] Traefik (request drop detection)
   - [x/blank] Edge/Kong (request drop, throttling, auth rejections)
   - [x/blank] Kafka (consumer lag, offset lag per topic)
   - [x/blank] Outbox (job SLA, oldest pending job age)
   - [x/blank] Uptime / StatusCake (downtime, liveliness P99)
   - [x/blank] Anomaly (RPS drop vs last week, latency increase vs last week)
   - [x/blank] AWS RDS (CPU, connections, write/read latency, replication lag, free space)
   - [x/blank] AWS ElastiCache (CPU, connections, memory usage, free memory)
   - [x/blank] AWS SQS (visible messages, message age)
   - [x/blank] AWS Node (CPU, memory, min/max node count)

   ### Alerts added: N
   ### Alerts skipped (dedup): M
   {list each skipped alert with the file:line where the existing alert lives}

   ### Input config
   Stored at: `baseline-monitoring-inputs/{app_name}.input.yaml`

   Generated by baseline-alerting skill v1.0
   EOF
   )"
   ```

   Fill in the checklist based on actual surfaces. Replace `[x/blank]` with `[x]` for enabled surfaces and `[ ]` for those not applicable. Replace `N`, `M`, and the skipped list with actual values from Phase 4.

**Output:** PR URL.

---

## Alert Modules Reference

The following Jinja2 templates in `${CLAUDE_SKILL_DIR}/templates/jinja2/modules/` generate alerts. Each is conditionally included based on the input config.

### Always Included
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `pod.yaml.j2` | Memory >75%, OOM, CPU >75%, restarts, non-ready pods, pod count min/max, memory limit failures | `container_memory_working_set_bytes`, `container_oom_events_total`, `container_cpu_usage_seconds_total`, `kube_pod_container_status_restarts_total`, `kube_pod_status_phase`, `kube_pod_container_status_running`, `container_memory_failcnt` |
| `egress.yaml.j2` | Egress RPS high/low per host, 5xx/4xx/non-2xx rates, P99/P95/P90 latency | `httpclient_http_requests_count`, `httpclient_http_request_duration_ms_hist_bucket` |
| `uptime.yaml.j2` | StatusCake downtime, liveliness P99 latency | `statuscake_test_up`, `{prefix}http_durations_ms_histogram_bucket` |

### Conditional on `application.interface.rest`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `ingress.yaml.j2` (REST section) | Max/Min RPS, 5xx >0.5%, 4xx >0.5%, non-2xx >1% | `{prefix}http_requests_total`, `{prefix}http_responses_total` |
| `latency.yaml.j2` (REST section) | P99 >100ms, P95 >200ms, P90 >300ms | `{prefix}http_durations_ms_histogram_bucket` |

### Conditional on `application.interface.grpc`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `ingress.yaml.j2` (gRPC section) | Max/Min RPS, Internal error >0.5%, InvalidArgument >0.5% | `{prefix}server_requests_total`, `{prefix}grpc_server_handled_total` |
| `latency.yaml.j2` (gRPC section) | P99 >100ms, P95 >200ms, P90 >300ms | `{prefix}grpc_server_handling_seconds_bucket` |

### Conditional on `infra.traefik`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `traefik_request_drop.yaml.j2` | Requests dropped between Traefik and app | `traefik_service_requests_total` vs `{prefix}http_requests_total` |

### Conditional on `infra.edgeService`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `edge.yaml.j2` | Request drop at edge, requests terminated at edge, throttling >50 RPS, AuthN/AuthZ rejections >10 RPS | `kong_request_total` |

### Conditional on `metrics.anomalyOffsets`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `anomaly.yaml.j2` | RPS drop >50% vs offset, latency increase >200ms vs offset | `{prefix}http_requests_total` (with `offset`), `{prefix}http_durations_ms_histogram_bucket` (with `offset`) |

### Conditional on `application.utils.hasOutboxWorkers` (GOLANG only)
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `outbox.yaml.j2` | Job SLA P99 >20s, oldest pending job >10s | `outbox_job_process_duration_seconds_bucket`, `outbox_age_of_oldest_pending_job_seconds` |

### Conditional on `infra.kafka.topics` (per topic)
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `kafka.yaml.j2` | Consumer lag >1000, offset lag per partition >15 | `kafka_consumergroup_lag`, `kafka_cluster_partition_laststableoffsetlag` |

### Conditional on `infra.aws.rds` (per instance)
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `aws/rds.yaml.j2` | CPU >80%, connections >300, write latency >1s, read latency >1s, replication lag >10s, replica not running, free space <50GB | `aws_rds_cpuutilization_maximum`, `aws_rds_database_connections_maximum`, `aws_rds_write_latency_maximum`, `aws_rds_read_latency_maximum`, `aws_rds_replica_lag_maximum`, `aws_rds_free_storage_space_maximum` |

### Conditional on `infra.aws.elasticCache` (per cluster)
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `aws/elastic_cache.yaml.j2` | CPU >75%, connections >100, memory >75%, free memory <5GB | `aws_elasticache_cpuutilization_maximum`, `aws_elasticache_curr_connections_average`, `aws_elasticache_database_memory_usage_percentage_average`, `aws_elasticache_freeable_memory_average` |

### Conditional on `infra.aws.sqs` (per queue)
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `aws/sqs.yaml.j2` | Visible messages >25, oldest message age >60s | `aws_sqs_approximate_number_of_messages_visible_sum`, `aws_sqs_approximate_age_of_oldest_message_sum` |

### Conditional on `infra.aws.asg`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `aws/node.yaml.j2` | Node CPU >75%, node memory >75% | `node_cpu_seconds_total`, `node_memory_MemAvailable_bytes` |

### Conditional on `infra.kubernetes.clusters`
| Module | Alerts Generated | Key Metrics |
|--------|-----------------|-------------|
| `aws/node.yaml.j2` (cluster section) | Min node count <2, max node count >5 | `cluster_autoscaler_node_groups_count` |

---

## Input Schema Reference

The source-of-truth schema lives at `${CLAUDE_SKILL_DIR}/inputs.template.yaml`. Key sections:

| Section | Required | Description |
|---------|----------|-------------|
| `team.bu` | YES | Business unit (Platform, Payments, RazorpayX, Capital, etc.) |
| `team.pod` | YES | Pod/team name |
| `team.slack.channel` | YES | Slack channel for alert routing |
| `team.slack.handle` | YES | Slack mention handle |
| `team.runbook` | YES | Runbook URL |
| `team.dashboardLink` | YES | Vajra/Grafana base dashboard URL (templates append panel IDs) |
| `cmd.appName` | YES | Service/application name |
| `metrics.prefix` | YES | Metric name prefix (must match instrumentation) |
| `metrics.statusCake.testId` | NO | StatusCake test ID |
| `metrics.anomalyOffsets` | NO | List of Prometheus offset durations (e.g., `["1w"]`) |
| `application.language` | NO | `GOLANG`, `JAVA`, etc. (needed for outbox alerts) |
| `application.interface.rest` | NO | `true` if REST endpoints exposed |
| `application.interface.grpc` | NO | `true` if gRPC endpoints exposed |
| `application.utils.hasOutboxWorkers` | NO | `true` if outbox workers used (GOLANG only) |
| `infra.kubernetes.namespace` | YES | K8s namespace |
| `infra.kubernetes.containers` | YES | List of container names |
| `infra.kubernetes.clusters` | NO | List of cluster names (for node count alerts) |
| `infra.traefik.services` | NO | List of Traefik service names |
| `infra.edgeService` | NO | Kong edge service name |
| `infra.kafka.topics` | NO | List of Kafka topic names |
| `infra.aws.rds` | NO | List of RDS instance identifiers |
| `infra.aws.elasticCache` | NO | List of ElastiCache cluster IDs |
| `infra.aws.sqs` | NO | List of SQS queue names |
| `infra.aws.asg` | NO | List of Auto Scaling Group names |

---

## Critical Rules

### Duplicate Detection is Non-Negotiable

Before adding ANY baseline alert, you MUST verify no semantically equivalent alert already exists. Two alerts are duplicates if:

1. They monitor the **same metric** (e.g., `container_memory_working_set_bytes`)
2. They filter on the **same namespace/service**
3. They have the **same or similar threshold**

When in doubt, **ask the user** whether an existing alert covers the same concern. Never silently add a duplicate.

### User Interaction Checkpoints

- **Phase 2:** Confirm detected surfaces before generating inputs
- **Phase 3:** Show final input YAML for approval before rendering
- **Phase 4:** Present dedup report before placing final file
- **Phase 5:** Show validated file for review before PR creation
- **Always:** If any required input is missing, ask the user. Never guess.

### Error Recovery

| Error | Action |
|-------|--------|
| `render.py` fails | Show full error output. Help user fix the input YAML. Re-run. |
| `promtool check rules` fails | Show the specific rule and line that failed. Fix the syntax. Re-validate. |
| YAML parse error | Show the parse error with line number. Fix indentation or quoting. Re-validate. |
| Git push fails | Show error. Check branch naming conflicts. Suggest manual steps if needed. |
| Missing required field in input | Do NOT proceed. Ask the user for the value. |

### Schema Versioning

`${CLAUDE_SKILL_DIR}/inputs.template.yaml` is the **source of truth** for the input schema. When this skill is updated with new alert types or surfaces, the UPDATE flow (Phase 3) handles migrating existing input files forward.

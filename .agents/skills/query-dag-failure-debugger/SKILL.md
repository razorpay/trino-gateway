---
name: query-dag-failure-debugger
description: Triages and investigates Airflow DAG, Spark/EMR, and Presto/Trino failures — classifies them into failure buckets, explains likely causes, and when runtime context is available, reads DAG config, queries AWS systems, fetches logs, and debugs a specific run. Use when given any DAG/task error, log snippet, failure description, or request to debug a DAG run.
user-invocable: true
category: data
tags: [airflow, spark, emr, presto, trino, debugging, triage]
version: "1.4.0"
author: "razorpay"
metadata:
  mcp_servers:
    - coralogix-mcp
    - friday-aws-mcp
    - friday-kubernetes-mcp-server
    - grafana-mcp
---

# Query DAG Failure Debugger

Failure-triage and live-investigation assistant for Airflow DAG and query job failures in the airflow-dags monorepo (~3000 DAGs, AWS ap-south-1). It classifies failures into one of 9 buckets, explains likely causes, then — on user confirmation — moves into active investigation mode: reads DAG definitions, queries AWS/EMR, and pulls the most relevant logs.

Does NOT auto-remediate or cover Databricks, Pinot, or Prometheus failures (planned for v2).

## Prerequisites

This skill requires the following MCP tools to be available:
- **Coralogix MCP** (`query_logs`) — Airflow task logs, task timelines, scheduler warnings, EMR cluster IDs from task output. DataPrime syntax. See [coralogix-airflow-queries.md](references/coralogix-airflow-queries.md).
- **Friday AWS MCP** (`mcp__aws-api__call_aws`) — EMR cluster/step inspection, S3 log fetch, Athena metadata checks. Auth is automatic — no profile or credentials needed.
- **Friday K8s MCP** (`kubectl`) — Airflow worker pod health, OOMKilled/eviction detection, pod logs, cluster events. Namespace: `airflow`, cluster: `prod-de-white`.
- **Grafana MCP** — correlated infra metrics (node/pod CPU/memory, Airflow worker saturation, EMR cluster metrics) for the same time window as the failure.

If any tool is unavailable, proceed with all other sources and note the gap in the `[Data Gaps]` section at the end. Never mention tool unavailability upfront.

## When to Use

- Someone pastes an error message, log snippet, or task failure context
- Someone asks to debug a specific DAG or task run end-to-end
- Spark/EMR failure (OOM, executor lost, bootstrap failure, step failure)
- Presto/Trino query failure (syntax error, table not found, query expired)
- Airflow sensor timeout or upstream dependency failure
- Permission or access error (IAM, S3, network)
- Unknown or unclear failure needing classification

## When NOT to Use

- Already know the failure type and just need the fix
- Failure is in Databricks, Pinot, or Prometheus DAGs (not covered in v1)
- Need deep query plan or cost analysis (use `data/skills/trino-analyzer/`)
- Need to understand DAG config patterns or repo structure

## Output Format (Vyom / Slack)

Output is rendered in Slack via Vyom. Format rules:
- Use emoji section headers (🔍 🔎 💡 🛠️ 🔗 ⚠️) — they render in Slack and make the triage block scannable
- Use Unicode box dividers (─────) for table borders
- Wrap IDs, cluster names, step IDs, query IDs, and error strings in backticks
- Keep lines under ~100 characters to avoid Slack truncation
- Use plain text for prose — no `##` markdown headers
- Split long responses with "↓ Response continues below..." at ~4000 chars
- Put the most important info (bucket, confidence, root cause) in the first ~20 lines

## Two-Phase Interaction Model

Every session follows two sequential phases. Do not skip Phase 1 to go straight to investigation.

### Phase 1 — Classify

Run the triage workflow on whatever context the user provided. Produce the full triage output block. Then ask:

> **"Does this classification look right? Want me to investigate further?"**
> If yes, provide:
> - For Spark/EMR: the DAG name and execution date (AWS access is handled automatically)
> - For Trino/Presto: the DAG name and, if available, the query ID and Airflow task log
> - For Airflow-only: the DAG name and execution date

If the user only asked for classification and says no to investigation, stop here.

### Phase 2 — Investigate

On user confirmation, enter active investigation mode. Start in **Fast mode** by default: read the DAG, collect the minimum artifacts for that backend, and do the single highest-signal confirming check. Only switch to **Deep mode** when Fast mode is inconclusive or the user explicitly wants full runtime debugging.

| Classified bucket | Investigation path |
|---|---|
| SPARK_RESOURCE / SPARK_RUNTIME | Coralogix task timeline → extract cluster ID → Friday AWS MCP for step logs; Grafana for node/pod memory metrics if OOM suspected; kubectl if worker pod crash suspected |
| SQL_FAILURE / QUERY_ENGINE_TRANSIENT / PERMISSION_ACCESS | Coralogix task log for exact error + query ID → Friday AWS MCP Athena for table metadata checks; see [trino-query-investigation.md](references/trino-query-investigation.md) |
| AIRFLOW_ORCHESTRATION / UPSTREAM_DATA | Coralogix task timeline + scheduler warnings → DAG file → [airflow-investigation.md](references/airflow-investigation.md); kubectl for worker pod health if tasks are stuck |
| INFRASTRUCTURE | Coralogix errors across multiple DAGs in same window → kubectl pod/node state → Grafana worker saturation metrics → escalate to platform team |
| UNKNOWN | Ask for the exact missing field listed in the [Data Gaps] section, then re-triage |

In investigation mode, produce concrete findings — not a checklist of things to check. Replace "Evidence: None" with actual results from reading the DAG, querying AWS, or running Trino checks.

Use Fast mode when the first confirming check can settle the bucket in under ~5 minutes. Use Deep mode when bucket boundaries remain ambiguous after the DAG read plus one confirming check.

---

## Failure Buckets (Quick Reference)

| Bucket | Description | Key Signals | Owner |
|--------|-------------|-------------|-------|
| AIRFLOW_ORCHESTRATION | DAG/task config, scheduling, sensor, dependency, timeout | `upstream_failed`, `sensor_timeout`, `execution_timeout`, `dagrun_timeout`, retry failure | DAG author |
| UPSTREAM_DATA | Required input data missing, late, or corrupt | freshness validator failure, partition missing, S3 path not found, TiDB lag | Upstream DAG owner |
| SPARK_RESOURCE | EMR Spark resource exhaustion or cluster config | `OutOfMemoryError`, YARN container killed, `BOOTSTRAP_FAILURE`, executor loss from unfiltered join | DAG author → on-call |
| SPARK_RUNTIME | Spark application logic error or bad input | `NullPointerException`, `AnalysisException`, `STEP_FAILURE` (non-OOM), Livy session dead | DAG/script author |
| SQL_FAILURE | Presto/Trino deterministic query error | `SYNTAX_ERROR`, `TABLE_NOT_FOUND`, `COLUMN_NOT_FOUND`, `TYPE_MISMATCH` | Query author |
| QUERY_ENGINE_TRANSIENT | Trino coordinator/worker instability | `QUERY_EXPIRED`, `NO_NODES_AVAILABLE`, `context deadline exceeded`, `db_error: trino: query failed (200 OK)` | Platform team / add retry |
| PERMISSION_ACCESS | IAM/S3 infra permissions OR Trino catalog permissions | `AccessDeniedException`, `403`, `ACCESS_DENIED` in Trino, `TABLE_NOT_FOUND` when table exists | Infra (IAM/S3) or Data team (Trino) |
| INFRASTRUCTURE | Platform-level failure: cluster terminated, spot reclaim, Airflow worker crash | Cluster `TERMINATED` (no STEP_FAILURE), multiple DAGs failing simultaneously | Platform/infra team |
| UNKNOWN | Insufficient context to classify | No error text, too short, signals match 2+ buckets equally | — |

For detailed signal lists, root causes, ownership, and disambiguation tips, see [failure-buckets.md](references/failure-buckets.md).

---

## Phase 1: Triage Workflow

Follow these steps deterministically. Two engineers given the same input should reach the same bucket.

### Step 1 — Parse the Input

Extract from whatever the user provides:
- **Error class**: e.g., `OutOfMemoryError`, `NotFoundException`, `AirflowSensorTimeout`
- **Error message**: the human-readable string
- **System**: Spark / Presto / Airflow / K8s / unknown
- **Task state**: failed / skipped / sensor_timeout / upstream_failed
- **Stack trace keywords**: top 3 frame package names
- **Explicit context**: DAG name, task name, execution date, cluster ID

### Step 1.5 — Establish the Failure Timeline

Before matching buckets, determine (from context or by asking) when this last succeeded:

- **First-time failure** — DAG has never succeeded → likely config bug, missing dependency, or data that never existed
- **Newly broken (regression)** — was passing recently, now failing → something changed; proceed to Step 1.6
- **Recurring / intermittent** — fails on and off → transient infra, data timing, or resource contention at peak hours
- **Backfill failure** — execution date is historical → data may not exist for that date, or schema changed since original run

If unclear, ask: *"When did this last succeed? Is this a first run, a regression, or an intermittent failure?"*

This context directly affects confidence level and next steps.

### Step 1.6 — "What Changed Recently?" (for regressions)

If the failure is newly broken, **proactively** check the following before asking the user anything:

1. **Check git log on the DAG file** — if the DAG name is known, look for commits in the last 24–72h. A recent commit is strong evidence for SPARK_RUNTIME or SQL_FAILURE.
   ```bash
   git log --oneline --since="3 days ago" -- dags/<path>/<dag_file>.py
   ```
2. **Check the last successful run** — ask the user or look at Airflow run history to determine when the DAG last succeeded. The gap between last success and first failure is the change window.
3. Ask only if the above is inconclusive: *"Did any infra change happen recently? (cluster config, IAM policy, S3 bucket rename, cluster migration)"*
4. Ask: *"Did any upstream DAG or table schema change recently?"*

| Signal | Effect on classification |
|--------|--------------------------|
| Recent DAG commit + runtime error | SPARK_RUNTIME or SQL_FAILURE — raise confidence |
| Recent infra change, no DAG commit | INFRASTRUCTURE or PERMISSION_ACCESS — raise confidence |
| Last success was days/weeks ago | Widen the change window search |
| No recent changes found | Confidence stays; proceed to signal matching |

### Step 1.7 — Determine Failure Shape

Before finalizing, determine:

- **Deterministic** — same failure on every retry → code / data / config bug
- **Flaky** — different errors on retries, or passes on retry without code change → QUERY_ENGINE_TRANSIENT or INFRASTRUCTURE
- **Time-correlated** — fails at the same time every day → resource contention or data freshness window

Surface this in the Evidence section: `[Failure shape: Deterministic / Flaky / Unknown]`

### Step 2 — Match Signals to Buckets

Before matching, run two fast pre-checks:

1. **Infrastructure check**: Are multiple unrelated DAGs failing in the same time window? If yes → INFRASTRUCTURE first. Don't classify per-DAG until the platform issue is ruled out.
2. **Trino permission check**: If the error is `TABLE_NOT_FOUND` or `ACCESS_DENIED` from Trino, use the Trino analysis decision tree in [failure-buckets.md](references/failure-buckets.md) before classifying as SQL_FAILURE or PERMISSION_ACCESS.

Before finalizing the bucket, also check the false-positive notes in [failure-buckets.md](references/failure-buckets.md). Some signals (`TABLE_NOT_FOUND`, `upstream_failed`, `ExecutorLostFailure`, `QUERY_EXPIRED`) commonly belong to a neighboring bucket.

Then check extracted signals against bucket definitions:

- **Exactly 1 bucket matches**: proceed to Step 3
- **2+ buckets match**: ask ONE clarifying question using the disambiguation tips in [failure-buckets.md](references/failure-buckets.md)
- **0 buckets match**: classify as UNKNOWN; list the 2-3 fields from Step 1 that were missing
- Never ask more than 2 clarifying questions total

### Step 3 — Assign Confidence

- **High**: 2+ strong signals from one bucket, and at least one of them is an exact exception class or exact log phrase from that bucket
- **Medium**: 1 strong signal OR 2 indirect signals (task state + system type), but one confirming check is still needed
- **Low**: only system type known, no error text, conflicting signals, or multiple buckets remain plausible

Use `UNKNOWN` instead of forcing a bucket when confidence is Low and no remaining clarifying question can settle it.
Low-confidence outputs must name the exact artifact that would change the classification (for example: Airflow task log, Trino `SHOW TABLES` result, EMR `stderr.gz`, upstream DAG run state).

Do not mark a bucket High confidence unless the evidence section contains at least one direct quote or exact signal from the user input.
If you cannot support a claim with the user-provided log text, DAG code, AWS output, Trino output, or reference docs, remove that claim or downgrade confidence.

### Step 4 — Fill the Output Template

Use the template in the "Output Template" section below. After outputting the triage block, always ask whether the user wants to proceed to investigation.

---

## Phase 2: Live Investigation Workflow

Use this workflow after the user confirms they want to investigate. Do not begin investigation until Phase 1 is complete and the user has confirmed.

### Step 1 — Gather Minimum Runtime Context

Ask only for what is needed for the backend you are investigating.

**Minimum needed from the user — for ALL backends:**
- DAG name / ID
- execution date (or approximate time window)

Everything else — task states, error text, cluster IDs, run IDs, scheduler warnings — pull from Coralogix first before asking the user. See [coralogix-airflow-queries.md](references/coralogix-airflow-queries.md).

**Additional context to ask for only if Coralogix returns nothing:**

- Spark/EMR: EMR cluster ID, step ID, whether cluster is still running
- Trino/Presto: exact error text, query ID, whether retry succeeded
- Airflow-only: exact task state, target upstream DAG/task if `ExternalTaskSensor` is involved

Do NOT ask for an AWS profile or credentials — Friday AWS MCP handles authentication automatically.
Do not ask for artifacts the user already shared.
Stop and surface the blocker if the minimum is still missing after 2 asks.

### Step 2 — Read the DAG Definition

Search the repo for the DAG file. Common locations:
- `dags/spark/`
- `dags/presto/`
- `dags/python/`
- `dags/data-quality/`

Extract execution-path fields based on backend:
- **Spark/EMR**: `CLUSTER_CONFIG_FILE`, `spark_submit_params`, `python_packages`, step names, job entrypoint
- **Presto/Trino**: query text or query builder, catalog/schema/table references, execution helper, `user` or connection argument
- **Airflow-only**: sensors, dependencies, pools, timeouts, retries, external task references

If the DAG code is too dynamic to extract these fields cleanly, stop and ask for the rendered Airflow task log or the exact DAG file path instead of guessing.

### Step 3 — Route to the Right Investigator

#### Spark/EMR Investigation

**Friday AWS MCP (always use — no profile or credentials needed)**

Use `mcp__aws-api__call_aws` to call AWS directly without any user-supplied credentials. All commands run in the `prod` account (`ap-south-1` region by default). Never ask the user to run CLI commands and paste results.

| Operation | Example command |
|-----------|----------------|
| Describe cluster | `aws emr describe-cluster --cluster-id j-XXXX` |
| List steps | `aws emr list-steps --cluster-id j-XXXX` |
| Get step failure | `aws emr describe-step --cluster-id j-XXXX --step-id s-XXXX` |
| Check log size | `aws s3api head-object --bucket <bucket> --key "<key>" --query 'ContentLength'` |
| List log files | `aws s3 ls s3://<bucket>/<prefix>/<cluster-id>/steps/<step-id>/` |
| Download log | `aws s3 cp s3://<bucket>/<path>/stderr.gz - \| gunzip` |

Use `mcp__aws-api__suggest_aws_commands` if unsure of the exact command syntax.
Read-only only. Check log size before downloading; stop and ask if > 100 MB.
See [emr-log-access.md](references/emr-log-access.md) for the full command reference.

**Investigation order**
1. **Query Coralogix first** — pull the Airflow task log for the failed task to get the task timeline, error text, and any cluster ID / step ID logged by the operator. Use the templates in [coralogix-airflow-queries.md](references/coralogix-airflow-queries.md). The cluster ID (`j-XXXX`) is usually in the task log line `Cluster created:` or `Waiting for step on cluster:`.
2. Derive cluster name pattern from `CLUSTER_CONFIG_FILE` if cluster ID was not found in Coralogix
3. Find the cluster via Friday AWS MCP (active or terminated)
4. Describe the cluster to get `LogUri` and failure reason
5. List steps and identify the failed step ID
6. Pull `stderr.gz` from S3 (highest signal for failures); check file size before downloading
7. If root cause is still unclear, switch to Deep mode and pull driver container `stdout.gz` / `stderr.gz`

See [emr-log-access.md](references/emr-log-access.md) for complete AWS CLI commands, fast/deep guidance, S3 log layout, and download safety rules.

#### Trino/Presto Investigation

1. Read the DAG file and extract the final SQL (or the builder), catalog/schema/table names, and the `user` argument
2. Pull exact error text and query ID from Airflow task log
3. Check retry history — did the same query pass on retry without code change?
4. Run the smallest metadata check that resolves the ambiguity:
   - `TABLE_NOT_FOUND` → `SHOW TABLES IN <catalog>.<schema> LIKE '<table>%'`
   - `COLUMN_NOT_FOUND` → `DESCRIBE <catalog>.<schema>.<table>`
   - `ACCESS_DENIED` → try `SELECT 1 FROM <table> LIMIT 1` under the same DAG user
5. Switch to Deep mode only if query history / worker instability analysis is still needed

See [trino-query-investigation.md](references/trino-query-investigation.md) for the full decision tree, metadata checks, and common patterns.

#### Airflow Orchestration Investigation

1. **Query Coralogix first** — pull the full task execution timeline for the DAG run. This gives you task states, durations, scheduler warnings (`max_active_runs`, pool saturation), and whether a manual run was also triggered. Use the timeline template in [coralogix-airflow-queries.md](references/coralogix-airflow-queries.md).
2. Read the DAG file, then use [airflow-investigation.md](references/airflow-investigation.md)
3. Inspect sensor definitions, `execution_delta`, `external_task_id`, `execution_timeout`, `dagrun_timeout`, `max_active_runs`, pool assignments, and retry behavior
4. For sensor timeouts: check the target DAG's run history for the exact execution date the sensor was targeting
5. For `upstream_failed`: trace the dependency chain to the first failed task using Coralogix error-only filter, not the propagated `upstream_failed` task
6. For `dagrun_timeout` or queued runs: Coralogix scheduler logs will show pool saturation or `max_active_runs` hit — check those before blaming data or infra

#### Kubernetes / Worker Pod Investigation

Use **Friday K8s MCP** when: a task is stuck in `queued`/`running` with no progress, OOM is suspected, or multiple tasks fail simultaneously (suggesting worker pod issues). Pod name is usually in the Coralogix task log — extract it from there first.

| Operation | Command |
|-----------|---------|
| Check pod health | `kubectl get pods -n airflow` |
| Describe a pod (OOMKilled / eviction) | `kubectl describe pod <pod-name> -n airflow` |
| Tail pod logs | `kubectl logs <pod-name> -n airflow --since=1h` |
| Cluster-level events | `kubectl get events -n airflow --sort-by='.lastTimestamp'` |

**Key signals from `kubectl describe pod`:**
- `OOMKilled` in `Last State` → worker ran out of memory → `SPARK_RESOURCE` or `INFRASTRUCTURE`
- `Evicted` + reason `The node was low on resource: memory` → node pressure → `INFRASTRUCTURE`
- High `Restart Count` → recurring worker crashes → escalate to platform team

#### Grafana Metrics Investigation

Use **Grafana MCP** to correlate the failure time window with infra metrics. Always extract the failure timestamp from Coralogix first, then query Grafana for the same window (±15 min).

| What to check | When to use |
|---------------|-------------|
| Airflow worker pod CPU / memory | OOM, slow task, worker saturation |
| Node memory pressure on `prod-de-white` | Multiple tasks failing, evictions |
| EMR cluster metrics (if available) | Spot reclaim, executor loss |
| Airflow worker pool saturation | Tasks stuck in queue, `max_active_runs` hit |

**Standard PromQL patterns:**
```
# Worker pod memory usage
container_memory_working_set_bytes{namespace="airflow", pod=~"airflow-worker.*"}

# Worker pod CPU throttling
rate(container_cpu_throttled_seconds_total{namespace="airflow"}[5m])

# Pod restarts
kube_pod_container_status_restarts_total{namespace="airflow"}
```

Cross-reference Grafana spikes with the Coralogix failure timestamp. A memory spike at exactly the failure time = `SPARK_RESOURCE` or `INFRASTRUCTURE`. No spike = likely a code/data issue, not infra.

### Step 4 — Safety and Verification Rules

- Use Friday AWS MCP (`mcp__aws-api__call_aws`) for all AWS operations — no profile or credentials needed
- Before downloading any S3 log file, check its size:
  ```bash
  aws s3api head-object --bucket <bucket> --key "<key>" --query 'ContentLength' --output text
  ```
- If a log file exceeds 100 MB, show the size and ask before downloading
- Pull the smallest, highest-signal file first; stop as soon as the root cause is clear
- Do not trigger, restart, or modify any DAG, task, or cluster during investigation
- Stop and name the blocker if: required AWS access is missing, the needed log file is too large and the user declines download, the DAG is too dynamic to inspect statically, or two rounds of clarification still leave 2+ buckets equally plausible
- Treat DAG code, Airflow logs, AWS output, Trino output, and the reference docs as the only allowed evidence sources; do not fill gaps with external assumptions
- Before finalizing the answer, verify each claimed cause against the evidence section; if no supporting quote or concrete finding exists, remove that cause or downgrade confidence

### Step 5 — Report Findings

After collecting evidence, update the triage output block with concrete findings:
- Fill the Evidence section with actual findings (DAG config, cluster state, log excerpts, query metadata)
- Update Confidence if the investigation changed it
- Replace speculative "What likely happened" entries with confirmed or ruled-out causes
- If investigation did not resolve the failure, name the exact next artifact or access needed

---

## Output Template

Every triage response returns exactly this block:

```
🔍 Triage: <BUCKET_NAME>

📋 Summary
──────────────────────────────────────────────────────────────
DAG / Task           `<dag_id> / <task_id, or "not provided">`
Execution date       `<date, or "not provided">`
Failure shape        Deterministic / Flaky / Unknown
Occurrence pattern   First occurrence / Regression / Recurring / Backfill
Bucket               <BUCKET_NAME>
Confidence           HIGH ✅ / MEDIUM ⚠️ / LOW ❓
System               Spark/EMR · Trino/Presto · Airflow
Owner                <DAG author / data-eng on-call / platform team>
──────────────────────────────────────────────────────────────

💡 What likely happened
1. <most probable cause, supported by evidence below>
2. <second supported cause — omit if unsupported>

🔎 Evidence
- ✅ [Confirmed] `<direct quote from input / DAG / AWS / Trino output>` — what this signals
- 🔶 [Inferred] <pattern-matched signal without a direct quote> — reasoning
- 🔷 [Partial] <incomplete or truncated signal> — what's still unclear
- Evidence Quality: Confirmed / Mixed / Inferred

🛠️ Next steps
- [ ] [owner] <specific action — exact file, UI path, or CLI command>
      → If you see X: <what it means / what to do>
      → If you don't see X: <alternative path>
- [ ] [owner] <next action if needed>

🔗 Related failures
- <Isolated to this DAG/task, or systemic? If unknown: check Airflow UI for failures in the same time window>

⚠️ [Data Gaps]
- <MCP / log / artifact that was unavailable and what it would have confirmed>
- "None" if all needed data was available
```

After the triage block, always append:

```
──────────────────────────────────────────────────────────────
🔍 Want me to investigate further? Share: <minimum identifiers for this bucket>
```

**Output rules**:
- Lead with analysis, not gaps — if a log or tool is missing, don't open with that; note it in [Data Gaps] at the END only
- ✅ [Confirmed] = exact quote from log/DAG/AWS/Trino; 🔶 [Inferred] = pattern match, no direct quote; 🔷 [Partial] = incomplete or truncated source
- Evidence Quality summary line reflects the weakest label present in the Evidence section
- Failure shape must be stated — never leave as Unknown if retry behavior is known
- Next steps must name exact locations: file path, UI path, or CLI command — not "check the logs"
- → If/else branches in next steps turn checklist items into decision trees
- [Data Gaps] is never blank — write "None" explicitly if nothing is missing
- Never leave any field in the Summary table blank
- Use emoji section headers throughout — they render well in Slack and make sections scannable at a glance
- Keep each line under ~100 characters to avoid Slack truncation
- Wrap cluster IDs, step IDs, query IDs, and error strings in backticks

---

## References

| File | Contents |
|------|----------|
| [failure-buckets.md](references/failure-buckets.md) | All 9 bucket definitions — signals, root causes, ownership, disambiguation, confidence boundaries, and false positives |
| [coralogix-airflow-queries.md](references/coralogix-airflow-queries.md) | DataPrime query templates for Airflow task timelines, error extraction, scheduler warnings, EMR cluster ID extraction, and run-specific filtering |
| [emr-log-access.md](references/emr-log-access.md) | Friday AWS MCP commands to pull EMR step and container logs from S3, fast/deep investigation path, and log patterns |
| [trino-query-investigation.md](references/trino-query-investigation.md) | Workflow for investigating Trino/Presto DAG failures via Athena: SQL extraction, metadata checks, ownership, fast/deep path |
| [airflow-investigation.md](references/airflow-investigation.md) | Airflow-first workflow for sensors, dependencies, pools, retries, timeouts, queued tasks, and upstream run debugging |
| [triage-examples.md](references/triage-examples.md) | Worked examples covering clean, ambiguous, and messy failure inputs |
| [eval-cases.md](references/eval-cases.md) | Compact regression cases for bucket choice, confidence, and first follow-up/action |
| [v2-roadmap.md](references/v2-roadmap.md) | Planned v2 extensions and current v1 scope limitations |

## V1 Scope

This is a v1.3.0 skill focused on classification plus targeted live investigation. It can inspect DAG definitions and fetch runtime evidence when the necessary access and identifiers are available, but it does not auto-remediate or perform broad cross-system correlation. See [v2-roadmap.md](references/v2-roadmap.md) for planned extensions.

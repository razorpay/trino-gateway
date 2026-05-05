---
name: post-deployment-monitor
description: Monitor Kubernetes deployments post-release by analyzing code changes and checking metrics. Use when users ask to "monitor deployment", "check deployment health", "verify deployment", "analyze deployment impact", or provide a deployment name with namespace and cluster. Automates deployment image comparison, code diff analysis, repository skill loading, flow impact analysis, and Grafana metrics validation. Generates comprehensive health reports with risk assessment.
---

# Post-Deployment Monitoring

Automated workflow for monitoring Kubernetes deployments after updates by analyzing code changes and checking observability metrics.

**Purpose:** Validate deployment health by correlating code changes with production metrics to detect regressions, errors, and performance degradation.

---

## ⚠️ CRITICAL: SEQUENTIAL EXECUTION REQUIRED

**ALL phases MUST run one at a time, in strict order. NEVER parallelize across phases.**

```
Phase 1: Fetch deployment info (kubectl via remote-friday-mcp-server)
    ↓  [COMPLETE before starting Phase 2]
Phase 2: Get code diff + read changed files
    ↓  [COMPLETE before starting Phase 3]
Phase 3: Load repo skills → identify affected flows + endpoints
    ↓  [COMPLETE before starting Phase 4]
Phase 4: Build Grafana queries (write them out explicitly)
    ↓  [COMPLETE before starting Phase 5]
Phase 5: Execute Grafana queries → generate health report
```

Combining any two phases in a single step is **not allowed**.

---

## Quick Start

**User provides:**
- Deployment name (e.g., `payments-card-live`)
- Namespace (e.g., `payments-card`)
- Cluster (e.g., `cde-green`)

---

## Phase 1: Fetch Deployment Info from Kubernetes

**Tool:** `remote-friday-mcp-server::kubectl_execute` — ALWAYS use this, never `gh` or Bash for K8s.

### 1.1: List ReplicaSets to find current and previous images

Use `jsonpath` output (custom-columns fails on annotation paths):

```
command: get
options: rs -n <namespace> -l name=<deployment-name> -o jsonpath='{range .items[*]}{.metadata.annotations.deployment\.kubernetes\.io/revision}{"\t"}{.spec.replicas}{"\t"}{.spec.template.spec.containers[0].image}{"\t"}{.metadata.creationTimestamp}{"\n"}{end}'
context: <cluster>-eks
```

**Extract:**
- **Current**: Highest revision number with `replicas > 0`
- **Previous**: Second-highest revision (regardless of replica count)
- **Deployment time**: `creationTimestamp` of the current ReplicaSet
- **Commit SHAs**: Parse from image tag — e.g., `c.rzp.io/razorpay/offers-engine:api_2da394ff...` → `2da394ff...`

**Example Output:**
```
12   0   c.rzp.io/razorpay/offers-engine:api_7e9891499298...   2026-04-05T10:12:00Z
13   3   c.rzp.io/razorpay/offers-engine:api_2da394ff5437...   2026-04-06T18:26:36Z
```

**Result:**
- Current SHA: `2da394ff5437...` (deployed at `2026-04-06T18:26:36Z`)
- Previous SHA: `7e9891499298...`

### 1.2: Fallback if jsonpath fails

```
command: get
options: rs -n <namespace> -l name=<deployment-name> -o wide
context: <cluster>-eks
```

Then narrow down with:
```
command: describe
options: rs <replicaset-name> -n <namespace>
context: <cluster>-eks
```

**⚠️ Common Mistakes:**
- ❌ Using `deployment.kubernetes.io/change-cause` annotation (often stale)
- ❌ Reading image from the Deployment object (may not reflect live pods)
- ✅ Always use ReplicaSet revision history sorted by revision number

---

## Phase 2: Get Code Diff

**Tool:** Bash with `gh api` — use GitHub compare endpoint to get commits and changed files.

```bash
gh api repos/razorpay/<repo-name>/compare/<previous-sha>...<current-sha> \
  --jq '{
    total_commits: .total_commits,
    commits: [.commits[] | {
      sha: .sha[0:7],
      message: (.commit.message | split("\n")[0]),
      author: .commit.author.name
    }],
    files_changed: (.files | length),
    files: [.files[] | .filename],
    stats: {
      additions: ([.files[].additions] | add),
      deletions: ([.files[].deletions] | add)
    }
  }'
```

**Then read each changed file** using Read tool (if repo is in workspace) or `gh api` to get file content for analysis.

**Assess risk:**
- 1–10 commits → 🟢 Low
- 11–50 commits → 🟡 Medium
- 51–100 commits → 🔴 High
- 100+ commits → 🔴 Very High

**⚠️ Analyze ALL commits, not just the latest.**

---

## Phase 3: Load Repo Skills → Identify Affected Flows and Endpoints

Complete Phase 2 fully before starting this phase.

### 3.1: Discover skills

If repo is in the current workspace:
```
Glob: .claude/skills/**/*.md
```

If not in workspace, use `gh api`:
```bash
gh api repos/razorpay/<repo-name>/contents/.claude/skills/<service>/modules/technical/observability/metrics.md \
  --jq '.content' | base64 -d
```

### 3.2: Load in priority order

1. **Observability docs** — `modules/technical/observability/metrics.md`
   - Extract: metric namespace, metric names, label dimensions, datasource UID
   
2. **Domain flow docs** — `modules/domain/*/flows.md` for each affected entity
   - Cross-reference changed files against flows
   - Identify which flows are modified, which are downstream

3. **API routes** — `modules/integration/apis/*.md`
   - Map changed server methods to gRPC/HTTP endpoint names
   - Extract the metric action label values used per endpoint

### 3.3: Output of this phase

Explicitly state before moving to Phase 4:

```
Affected endpoints:
  - <endpoint-1> → metric action label: "<action-value-1>"
  - <endpoint-2> → metric action label: "<action-value-2>"

Metric namespace: <oe|cardps|...>
Datasource UID: <uid>
Deployment time (RFC3339): <timestamp>
Deployment time (2h before): <timestamp-minus-2h>
```

**⚠️ Metrics Discovery:**
- NEVER assume metric names — always read from observability docs
- Each service has unique naming (e.g., `oe_*` vs `cardps_*` vs `grpc_server_*`)

---

## Phase 4: Build Grafana Queries

Complete Phase 3 fully before starting this phase.

Write out ALL queries explicitly before executing any. Do not query Grafana yet.

### 4.1: Time boundaries

- **Before**: Use `startTime=<deployment-time-minus-2h>` in RFC3339 format (e.g., `2026-04-06T16:26:36Z`)
- **After**: Use `startTime=now`

**⚠️ Use `startTime` parameter in the MCP tool, NOT `@ timestamp` in PromQL.** The `@ modifier` approach often returns empty results with the Grafana MCP tool.

### 4.2: Query templates

Replace `<namespace>`, `<action_regex>`, `<metric_prefix>` with values from Phase 3.

**Error Rate:**
```promql
# BEFORE
sum(rate(<metric_prefix>_error_response_total{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m]))

# AFTER
sum(rate(<metric_prefix>_error_response_total{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m]))
```

**Traffic:**
```promql
# BEFORE
sum(rate(<metric_prefix>_requests_total{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m]))

# AFTER
sum(rate(<metric_prefix>_requests_total{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m]))
```

**P95 Latency:**
```promql
# BEFORE
histogram_quantile(0.95, sum(rate(<metric_prefix>_durations_ms_histogram_bucket{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m])) by (le))

# AFTER
histogram_quantile(0.95, sum(rate(<metric_prefix>_durations_ms_histogram_bucket{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m])) by (le))
```

**P99 Latency:**
```promql
# BEFORE
histogram_quantile(0.99, sum(rate(<metric_prefix>_durations_ms_histogram_bucket{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m])) by (le))

# AFTER
histogram_quantile(0.99, sum(rate(<metric_prefix>_durations_ms_histogram_bucket{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m])) by (le))
```

**Per-action error breakdown (always include):**
```promql
sum by (action, error) (rate(<metric_prefix>_error_response_total{kubernetes_namespace="<namespace>",action=~"<action_regex>"}[10m]))
```

**For gRPC services (Razorpay Foundation framework):**
```promql
# Error rate
sum(rate(grpc_server_handled_total{kubernetes_namespace="<namespace>", grpc_code!="OK"}[10m]))

# P95 latency
histogram_quantile(0.95, sum(rate(grpc_server_handled_duration_seconds_bucket{kubernetes_namespace="<namespace>"}[10m])) by (le))
```

### 4.3: Discover datasource UID (if not in skills docs)

```
Tool: grafana::list_datasources
params: type=prometheus
```

Default Victoria Metrics UID: `ALvd9Tgnz`

---

## Phase 5: Execute Grafana Queries + Generate Report

Complete Phase 4 fully (queries written out) before running any Grafana calls.

**Tool:** `grafana::query_prometheus`

Execute in this order:
1. Error rate BEFORE (`startTime=<2h-before-RFC3339>`)
2. Error rate AFTER (`startTime=now`)
3. Traffic BEFORE
4. Traffic AFTER
5. P95 latency BEFORE
6. P95 latency AFTER
7. P99 latency BEFORE
8. P99 latency AFTER
9. Per-action error breakdown AFTER

### Report Template

```markdown
# Post-Deployment Monitoring Report

**Deployment:** <name>
**Namespace:** <namespace>
**Cluster:** <cluster>
**Deployment Time:** <RFC3339 timestamp>

---

## Deployment Scope

- **Previous SHA:** <sha>
- **Current SHA:** <sha>
- **Total Commits:** <count>
- **Files Changed:** <count>
- **Lines Added / Deleted:** <+X / -Y>

---

## Commit Breakdown

### Commit 1: `<sha>` — <title>
**Author:** <name>
**Risk:** 🟢 Low / 🟡 Medium / 🔴 High
**Affects:** <endpoint(s)> → metric action: `<action-label>`
**Summary:** <what changed and why>

---

## Affected Endpoints

| Endpoint | Action Label | Changed By |
|----------|-------------|------------|
| <endpoint> | `<action>` | Commit <sha> |

---

## Metrics Analysis

> **Time-of-day note (if applicable):** BEFORE snapshot taken at <time IST>. AFTER at <time IST>. Traffic deltas may reflect time-of-day variation.

### 1. Traffic

| Action | Before | After | Change | Status |
|--------|--------|-------|--------|--------|
| `<action>` | X.XX/s | X.XX/s | X% | ✅/⚠️/❌ |
| Combined | X.XX/s | X.XX/s | X% | ✅/⚠️/❌ |

### 2. Error Rates

| Action | Error | Before | After | Status |
|--------|-------|--------|-------|--------|
| `<action>` | `<error-code>` | X.XX/s | X.XX/s | ✅/⚠️/❌ |

### 3. Latency

| Metric | Before | After | Change | Status |
|--------|--------|-------|--------|--------|
| P95 | Xms | Xms | +X% | ✅/⚠️/❌ |
| P99 | Xms | Xms | +X% | ✅/⚠️/❌ |

---

## Risk Assessment

**Overall Status:** ✅ HEALTHY / ⚠️ WARNINGS / ❌ ISSUES

### Findings

1. **`<endpoint>`** — ✅/⚠️/❌ <summary>
   - <specific metric observation>
   - <root cause hypothesis if error>

---

## Conclusion

**Status:** ✅ No regressions / ⚠️ Minor issues / ❌ Rollback recommended

**Report Generated:** <timestamp>
**Data Sources:** Victoria Metrics (Grafana), Kubernetes (remote-friday-mcp-server), GitHub
```

---

## Alert Thresholds

### Error Rate
- 🟢 **Green**: <10% increase
- 🟡 **Warning**: 10–20% increase
- 🔴 **Critical**: >20% increase

### 4XX Errors
- 🟢 **Green**: <15% increase
- 🟡 **Warning**: 15–30% increase
- 🔴 **Critical**: >30% increase OR new 4xx codes appearing

### Latency P95
- 🟢 **Green**: <20% increase
- 🟡 **Warning**: 20–30% increase
- 🔴 **Critical**: >30% increase

### Latency P99
- 🟢 **Green**: <30% increase
- 🟡 **Warning**: 30–50% increase
- 🔴 **Critical**: >50% increase

### Traffic Volume
- 🟢 **Green**: <10% change (or explainable by time-of-day)
- 🟡 **Warning**: 10–50% drop (unexplained)
- 🔴 **Critical**: >50% drop OR zero traffic on active endpoints

### Deployment Size Risk
- 🟢 **Low**: 1–10 commits
- 🟡 **Medium**: 11–50 commits
- 🔴 **High**: 51–100 commits
- 🔴 **Very High**: 100+ commits

---

## Troubleshooting

### jsonpath query returns empty
Use `-o wide` first to see available fields, then `describe` for the specific ReplicaSet.

### `@ timestamp` PromQL returns empty in Grafana MCP
Use `startTime=<RFC3339>` parameter in the tool call instead. Pass the evaluation time as the tool's `startTime`, not as an in-query modifier.

### Metric names return no data
Use `grafana::list_prometheus_metric_names` with a regex prefix (e.g., `^oe_server`) to discover actual metric names before querying.

### Label values differ from docs
Use `grafana::list_prometheus_label_values` with the metric name filter to find actual label values in use.

### No previous ReplicaSet
First deployment — report as "Initial deployment, no baseline available."

### Multiple ReplicaSets with replicas > 0
Rolling update still in progress. Use highest revision as current; wait for rollout to stabilize.

---

## Tool Reference

| Task | Tool |
|------|------|
| Get ReplicaSet images | `remote-friday-mcp-server::kubectl_execute` |
| Get commit diff | Bash `gh api repos/razorpay/<repo>/compare/<prev>...<curr>` |
| Read repo skill files | `Read` tool (if in workspace) or `gh api` for file content |
| Discover metric names | `grafana::list_prometheus_metric_names` |
| Discover label values | `grafana::list_prometheus_label_values` |
| Query metrics | `grafana::query_prometheus` |

**Default Victoria Metrics datasource UID:** `ALvd9Tgnz`

---

**Skill Version:** 3.0.0
**Last Updated:** 2026-04-07

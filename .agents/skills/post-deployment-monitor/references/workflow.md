# Post-Deployment Monitoring Workflow

This document details the complete workflow for monitoring a Kubernetes deployment after it has been updated.

## ⚠️ CRITICAL PRINCIPLES

1. **ALWAYS validate with Grafana metrics** - NEVER skip metrics based on code findings
   - Code analysis identifies potential issues
   - Metrics validation confirms actual impact
   - Code may be broken but have zero traffic (no impact)
   - Code may look safe but have production issues

2. **ALWAYS use deployment time as the analysis boundary**
   - Get exact pod creation timestamp
   - Compare metrics BEFORE deployment vs AFTER deployment
   - Look for changes/spikes starting EXACTLY at deployment time
   - Never use arbitrary time windows (yesterday, last week, etc.)

## Overview

Given a deployment name, namespace, and cluster, this workflow:
1. Retrieves deployment image information and **exact deployment timestamp**
2. Identifies code changes between deployments
3. **Analyzes ALL commits** (not just the most recent one)
4. Categorizes changes by risk level
5. Monitors affected components using repository skills
6. **Checks observability metrics using deployment time boundary** (MANDATORY)

## ⚠️ CRITICAL: Multi-Commit Analysis

**Common Mistake:** Only analyzing the most recent commit in a deployment.

**Correct Approach:**
1. Get the FULL diff between old and new deployment commits
2. List ALL commits in the range
3. Analyze each commit's changes
4. Aggregate statistics across all commits
5. Identify all new features, migrations, workers, etc.

**Example:**
```bash
# ❌ WRONG: Only looking at the HEAD commit
gh api repos/owner/repo/commits/<new-sha>

# ✅ CORRECT: Getting ALL commits in the deployment
gh api repos/owner/repo/compare/<old-sha>...<new-sha> \
  --jq '{total_commits, commits: [.commits[]]}'
```

## Complete Workflow

### Step 1: Get Deployment Images

**Tool:** `remote-friday-mcp-server`

Get current and previous images from the deployment:

```
kubectl_execute:
  cluster: <cluster-name>
  command: kubectl get deployment <deployment-name> -n <namespace> -o jsonpath='{.spec.template.spec.containers[0].image}'
```

Also check ReplicaSet history for previous image:

```
kubectl_execute:
  cluster: <cluster-name>
  command: kubectl get rs -n <namespace> --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-2].spec.template.spec.containers[0].image}'
```

**Output:** Current and previous image tags

### Step 2: Extract Commit SHAs

**Tool:** `scripts/parse_image_tag.py`

Parse both image tags to extract commit SHAs:

```bash
python scripts/parse_image_tag.py "registry.com/app:v1.2.3-abc123"
# Output: abc123

python scripts/parse_image_tag.py "registry.com/app:v1.2.4-def456"
# Output: def456
```

**Output:** Two commit SHAs (old and new)

### Step 3: Get Code Diff

**Tool:** `github` MCP server or `gh` CLI

First, identify the repository from the deployment name or namespace. Common patterns:
- Deployment name often matches repo name
- Check namespace annotations or labels for repo info
- Ask user if unclear

**CRITICAL:** Get the COMPLETE diff analyzing ALL commits, not just the most recent one.

#### Step 3.1: List All Commits in Range

```bash
gh api repos/<owner>/<repo>/compare/<old-sha>...<new-sha> --jq '{total_commits: .total_commits, commits: [.commits[] | {sha: .sha, message: .commit.message, author: .commit.author.name, date: .commit.author.date}]}'
```

**Output:** List of ALL commits included in deployment

#### Step 3.2: Get Aggregate Statistics

```bash
gh api repos/<owner>/<repo>/compare/<old-sha>...<new-sha> --jq '{total_commits: .total_commits, stats: {additions: .files | map(.additions) | add, deletions: .files | map(.deletions) | add, files_changed: (.files | length)}}'
```

**Output:**
- Total commits (e.g., 3 commits)
- Total lines added (e.g., 7704)
- Total lines deleted (e.g., 45)
- Total files changed (e.g., 73)

#### Step 3.3: Get Complete File Changes

```bash
gh api repos/<owner>/<repo>/compare/<old-sha>...<new-sha> --jq '.files[] | {filename, status, additions, deletions}'
```

**Output:** Complete list of all files changed across ALL commits

#### Step 3.4: Identify Major Changes

Look for:
- New files added (status: "added")
- Large files (>100 lines added)
- Database migrations
- New workers/services
- Configuration changes

```bash
gh api repos/<owner>/<repo>/compare/<old-sha>...<new-sha> --jq '.files[] | select(.additions > 100 or .status == "added") | {filename, status, additions, deletions}' | jq -s 'sort_by(.additions) | reverse'
```

**Output:** Major changes requiring special attention

### Step 4: Load Repository Skills

**Tool:** File system operations

Look for Claude skills in the repository:

```bash
# Skills should be in .claude/skills/ directory at repo root
ls -la .claude/skills/
```

Common skill locations:
- `.claude/skills/<service-name>/SKILL.md`
- `.claude/skills/<service-name>/domain/*/flows.md`
- `.claude/skills/<service-name>/integration/apis/*.md`

**Load:** All relevant skill documentation for the service

### Step 5: Analyze Impact

**Using:** Repository skills + code diff + ALL commits

**CRITICAL:** Analyze impact across ALL commits, not just the most recent one.

#### Step 5.1: Categorize Changes by Commit

For each commit in the deployment:
1. Summarize the commit's purpose (from commit message)
2. Identify the commit's scope (files changed)
3. Assess the commit's risk level (new features, bug fixes, refactors)

Example:
```
Commit 1: Add tracking level to offers (Low risk - additive)
Commit 2: Migrate segments table (HIGH risk - new DB table, worker)
Commit 3: Add panic handling to worker (Low risk - defensive)
```

#### Step 5.2: Identify High-Risk Changes

Look for across ALL commits:
- **New database migrations** - Schema changes
- **New workers/services** - Entirely new components
- **New API endpoints** - New routes exposed
- **Changed critical paths** - Payment processing, auth, etc.
- **Configuration changes** - New queues, topics, services

#### Step 5.3: Map to Business Flows

Using repository skills, identify:

1. **Changed files** - Which files were modified across ALL commits
2. **Affected flows** - Which business flows are impacted (from flows.md)
3. **Changed components** - Services, handlers, repositories affected
4. **Integration points** - External APIs or services touched

**Key questions:**
- What flows reference the changed files?
- What business logic changed?
- What API endpoints are affected?
- Are there new workers that need monitoring?
- Did database schema change?

**Output:**
- List of affected flows and their routes
- List of new components to monitor
- Assessment of deployment risk level

### Step 6: Extract Routes

**Tool:** `scripts/extract_routes.py` or manual extraction

For each affected flow, extract the API routes:

```bash
# If flows are well-documented
python scripts/extract_routes.py .claude/skills/<service>/domain/offer/flows.md

# Or manually parse the flow documentation
```

**Output:** List of routes to monitor (e.g., `POST /v1/offers`, `GET /v1/transactions/{id}`)

### Step 7: Check Grafana Metrics ⚠️ MANDATORY - NEVER SKIP

**Tool:** `grafana` MCP server

**⚠️ CRITICAL:** ALWAYS validate with metrics regardless of code analysis findings. Use deployment time as the critical boundary for before/after comparison.

#### 7.0: Get Exact Deployment Time (FIRST STEP)

```bash
kubectl get pods -n <namespace> -o json | jq -r '.items[0].metadata.creationTimestamp'
# Output: "2026-02-17T21:52:51Z"
```

**Use this timestamp as the boundary for all metric comparisons.**

For each route, check metrics before and after deployment:

#### 7.1: Check Error Rates - 4XX and 5XX (Before vs After Deployment)

**Time Windows:**
- **Before:** 2 hours before deployment timestamp
- **After:** From deployment timestamp to now

**7.1a: Query for HTTP 5XX Errors (Server failures):**

```promql
# Before deployment (2h window ending at deployment time)
sum by (route) (rate(http_requests_total{status=~"5..", route="<route>"}[5m]))
# Query from: deployment_time - 2h to deployment_time

# After deployment (from deployment time to now)
sum by (route) (rate(http_requests_total{status=~"5..", route="<route>"}[5m]))
# Query from: deployment_time to now
```

**Alert if:**
- >20% increase in 5xx rate after deployment
- New 5xx errors appearing exactly at deployment time
- Any 5xx spike starting at deployment timestamp

**7.1b: Query for HTTP 4XX Errors (Client errors - breaking changes):**

```promql
# Before deployment - by status code
sum by (route, status) (rate(http_requests_total{status=~"4..", route="<route>"}[5m]))

# After deployment - by status code
sum by (route, status) (rate(http_requests_total{status=~"4..", route="<route>"}[5m]))
```

**Alert if:**
- >30% increase in 4xx rate after deployment
- New 4xx error codes appearing at deployment time:
  - **400 Bad Request** → API contract broken (request validation changed)
  - **401 Unauthorized** → Authentication logic changed
  - **403 Forbidden** → Authorization/permission changes
  - **404 Not Found** → Routes removed or paths changed
  - **422 Unprocessable Entity** → Schema validation added/tightened
- Spike in specific 4xx codes (404, 403, 401) starting at deployment

**Why 4XX matters:**
- Indicates breaking changes in API contracts
- Shows authentication/authorization regressions
- Detects routing configuration issues
- Reveals request validation changes affecting clients

**Example:** If deployment at 21:52:51Z, compare:
- Before: 19:52:51Z to 21:52:51Z
- After: 21:52:51Z to now
- Check both aggregated 4xx/5xx rates AND individual status codes (400, 401, 403, 404, 500, 502, 503, 504)

#### 7.2: Check Latency (Before vs After Deployment)

**Time Windows:** Same as 7.1 (before/after deployment time)

Query for request latency (p95, p99):

```promql
# P95 latency before deployment
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m])) by (le, route))

# P95 latency after deployment
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m])) by (le, route))
```

**Alert if:**
- >30% increase in p95 latency after deployment
- >50% increase in p99 latency after deployment
- Latency spike starting exactly at deployment timestamp

#### 7.3: Check Traffic Volume (Before vs After Deployment)

**Critical for detecting routing issues or broken endpoints:**

```promql
# Request rate before deployment (2h average)
sum(rate(http_requests_total{route="<route>"}[5m]))

# Request rate after deployment (since deployment)
sum(rate(http_requests_total{route="<route>"}[5m]))
```

**Alert if:**
- Traffic drops >50% after deployment (routing issue)
- Zero traffic to previously active endpoints after deployment
- New endpoints receiving zero traffic (may indicate broken deployment)

**Example findings:**
- "Before: 100 req/sec, After: 0 req/sec" → Critical routing issue
- "Before: 0 req/sec, After: 0 req/sec" → New/unused feature, no impact from broken code

#### 7.4: Check for Deployment-Correlated Changes

Look specifically for changes starting at deployment timestamp:

**Error pattern changes:**
```promql
# Check for new error types appearing post-deployment
sum by (error_type) (increase(errors_total[1h]))
# Query before and after deployment time
```

**Timeout increases:**
```promql
# Check for timeout spikes at deployment time
sum(rate(http_requests_total{status="504"}[5m]))
```

**Use Sift for deployment time window:**
```
get_sift_analysis:
  query: "anomalies around <deployment_timestamp>"
  time_range: "<deployment_time - 30m> to <deployment_time + 30m>"
```

**Alert if:**
- New error types appear immediately after deployment
- Timeout rate increases starting at deployment time
- Sift detects anomaly spike at deployment timestamp

### Step 8: Generate Report

Compile findings into a structured report:

```markdown
# Post-Deployment Monitoring Report

**Deployment:** <deployment-name>
**Namespace:** <namespace>
**Cluster:** <cluster>
**Time:** <timestamp>

## Deployment Changes

**Previous Image:** <old-image>
**Current Image:** <new-image>
**Commit Range:** <old-sha>...<new-sha>

## Code Changes

<summary of diff - files changed, lines added/removed>

## Affected Flows

1. Flow X.Y: <flow-name>
   - Route: POST /v1/offers
   - Changes: <brief description>

2. Flow A.B: <flow-name>
   - Route: GET /v1/transactions/{id}
   - Changes: <brief description>

## Metrics Analysis

**Deployment Time:** 2026-02-17T21:52:51Z
**Before Window:** 19:52:51Z to 21:52:51Z (2 hours)
**After Window:** 21:52:51Z to now

### Route: POST /v1/offers
- **5xx Errors:** ✅ No increase (0.01% → 0.01%)
- **4xx Errors:** ⚠️ New 400 errors appearing at deployment time (0% → 2.3%)
- **Latency (p95):** ⚠️ Increased by 25% (120ms → 150ms)
- **Traffic Volume:** ✅ Stable (100 req/s → 98 req/s)
- **Anomalies:** ✅ None detected

### Route: GET /v1/transactions/{id}
- **5xx Errors:** ❌ Increased by 45% (0.1% → 0.145%)
- **4xx Errors:** ❌ New 404 errors at deployment (0% → 5.2%) - Route may be broken
- **Latency (p95):** ✅ Stable (80ms → 82ms)
- **Traffic Volume:** ❌ Dropped 60% (50 req/s → 20 req/s) - Routing issue
- **Anomalies:** ⚠️ Sift detected unusual traffic pattern at 21:53Z (1 min after deployment)

## Conclusion

**Status:** ⚠️ ISSUES DETECTED

**Issues Found:**
1. Route POST /v1/offers showing latency degradation
2. Route GET /v1/transactions/{id} showing increased 5xx errors
3. Anomaly detected in traffic pattern

**Recommended Actions:**
1. Investigate latency increase in offers endpoint
2. Check logs for 5xx errors in transactions endpoint
3. Review anomaly details in Sift investigation
4. Consider rollback if issues persist

**Next Steps:**
- Monitor for 15 more minutes
- If issues persist, prepare rollback plan
- Check error logs for root cause
```

## Common Issues & Troubleshooting

See [troubleshooting.md](troubleshooting.md) for common issues and solutions.

## Time Windows

**⚠️ CRITICAL:** Always use deployment time as the boundary, not arbitrary time windows.

**Deployment Time Approach (REQUIRED):**
1. Get exact deployment timestamp: `kubectl get pods -n <namespace> -o json | jq -r '.items[0].metadata.creationTimestamp'`
2. **Before window:** 2 hours before deployment timestamp
3. **After window:** From deployment timestamp to now
4. **Minimum monitoring duration:** 30 minutes post-deployment (to allow issues to surface)

**Example:**
- Deployment time: `2026-02-17T21:52:51Z`
- Before window: `2026-02-17T19:52:51Z` to `2026-02-17T21:52:51Z`
- After window: `2026-02-17T21:52:51Z` to `now`
- Minimum wait: Check metrics after `2026-02-17T22:22:51Z` (30 min after deployment)

**Why deployment time is critical:**
- Detects issues starting EXACTLY when new code deployed
- Avoids false positives from unrelated time periods
- Correlates metric changes with deployment event
- Enables root cause analysis: "Did this start at deployment?"

**Deprecated approaches (DO NOT USE):**
- ❌ "Same time yesterday" - Doesn't correlate with deployment
- ❌ "Last 7 days average" - Smooths over deployment boundary
- ❌ "Last 30 minutes vs previous 30 minutes" - Arbitrary, not deployment-aligned

## Thresholds

**5xx errors (Server failures):**
- ✅ Green: <10% increase
- ⚠️ Warning: 10-20% increase
- ❌ Critical: >20% increase

**4xx errors (Client errors - breaking changes):**
- ✅ Green: <15% increase
- ⚠️ Warning: 15-30% increase
- ❌ Critical: >30% increase OR new 4xx codes appearing (400, 401, 404)

**Latency (p95):**
- ✅ Green: <20% increase
- ⚠️ Warning: 20-30% increase
- ❌ Critical: >30% increase

**Latency (p99):**
- ✅ Green: <30% increase
- ⚠️ Warning: 30-50% increase
- ❌ Critical: >50% increase

**Traffic volume:**
- ✅ Green: <10% change
- ⚠️ Warning: 10-50% drop (routing issue)
- ❌ Critical: >50% drop OR zero traffic on previously active endpoints

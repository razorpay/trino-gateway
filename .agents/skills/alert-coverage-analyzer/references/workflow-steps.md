# Detailed Workflow Steps

This document provides step-by-step implementation details for the Alert Coverage Analyzer workflow.

### 1. Verify Razorpay Repository

Run the verification script:
```bash
python3 scripts/verify_razorpay_repo.py .
```

If output is "false", **STOP** and inform user: "This is not a Razorpay repository. This skill only works with Razorpay repos."

### 2. Identify Application Name

Run the service identification script:
```bash
python3 scripts/identify_service.py .
```

Display the identified service name to the user. If output is "unknown" or user disagrees, ask user to provide the correct application name.

### 2.5 Check for Repo Skill (NEW - MANDATORY)

**BEFORE starting analysis, check if a repo skill exists:**

```bash
ls -la .agents/skills/
```

**If a skill exists (e.g., `<service-name>-skill/`):**

**MANDATORY:** Read these modules FIRST (before scanning code):

1. **`modules/technical/observability/metrics.md`** - Existing metrics inventory
2. **`modules/technical/observability/logging.md`** - Log patterns and alerts
3. **`modules/technical/observability/tracing.md`** - Tracing setup
4. **`modules/technical/database/`** - DB monitoring patterns
5. **`modules/technical/integrations/`** - External service monitoring (Kafka, Redis, etc.)

**Then ask user:**
```
"I found a <service-name>-skill in your repo with observability documentation.
I'll use it to understand your existing monitoring setup and avoid false positives.

Key findings from skill:
- Existing metrics: <count from metrics.md>
- SLIs/SLOs: <list from observability docs>
- Known gaps: <list documented gaps>
"
```

**If no skill exists:**
```
"No repo skill found. I'll proceed with code analysis, but this may have lower accuracy.

Recommendation: Consider creating a repo skill with an observability section for:
- Better gap identification
- Documented SLIs/SLOs
- Faster future audits
"
```

**Why this is critical:**
- Repo skills document what SHOULD exist (SLIs/SLOs)
- Prevents recommending metrics that exist elsewhere (logs, CloudWatch)
- Provides service-specific context (naming conventions, critical flows)

### 3. Set Up GitHub CLI

Check if `gh` is installed:
```bash
which gh
```

If not found:
1. Install via Homebrew: `brew install gh`
2. Authenticate: `gh auth login`
3. Guide user through the authentication process (device flow)
4. Verify: `gh auth status`

### 4. Locate Alert-Rules Repository

First, ask the user if they already have the alert-rules repository cloned locally:
```
Do you already have the razorpay/alert-rules repository cloned? If yes, please provide the path.
```

**If user provides a path:**
```bash
# Verify it's the correct repository
cd <user-provided-path>
git remote -v | grep "razorpay/alert-rules"
```

If verification succeeds, use that path. If it fails, inform user and proceed to clone.

**If user says no or verification fails:**

Clone the Razorpay alert-rules repository to `/tmp`:
```bash
cd /tmp && gh repo clone razorpay/alert-rules
```

Store the alert-rules path (either user-provided or `/tmp/alert-rules`) for use in subsequent steps.

### 5. Read Existing Alerts (ENHANCED - Multi-Source Discovery)

Search for ALL alert types related to the service:

#### Application Alerts (Prometheus)
```bash
find <alert-rules-path> -name "*<service-name>*" -type f | grep -E "rules/(prod|nonprod)-rules"
```

#### Infrastructure Alerts (NEW)
```bash
# RDS/Database alerts
find <alert-rules-path> -name "rds_*<service-name>*.yaml" -o -name "rds_generated_alerts_*<service-name>*.yaml"

# EKS/Container alerts
find <alert-rules-path> -name "eks_*<service-name>*.yaml"

# Lambda alerts
find <alert-rules-path> -name "lambda_*<service-name>*.yaml"

# ElastiCache alerts
find <alert-rules-path> -name "elasticache_*<service-name>*.yaml"
```

**For EACH alert file found:**

1. Read the file
2. Extract and categorize alerts:
   ```bash
   grep -E "alert:|expr:" <alert-file>.yaml
   ```

3. Create a coverage matrix:

| Alert Type | Metric | Threshold | File | Source |
|-----------|--------|-----------|------|--------|
| HTTP 5xx | `http_responses_total{code=~"5.*"}` | >1/5m | `ledger_rules_app.yaml` | Prometheus |
| DB CPU | `aws_rds_cpuutilization_maximum` | >60% | `rds_generated_alerts_*.yaml` | CloudWatch |
| DB Memory | `aws_rds_freeable_memory_maximum` | <262MB | `rds_generated_alerts_*.yaml` | CloudWatch |
| DB Connections | `aws_rds_database_connections_maximum` | >1500 | `rds_generated_alerts_*.yaml` | CloudWatch |
| DB Write Latency | `aws_rds_write_latency_maximum` | >100ms | `rds_generated_alerts_*.yaml` | CloudWatch |

**Extract for each alert:**
- Alert groups and names
- Metrics being monitored
- Current thresholds
- Coverage areas (HTTP, latency, business flows, DB infrastructure, cache, Kafka, etc.)
- Source (Prometheus, CloudWatch, etc.)

**Present summary to user:**
```
Found monitoring across multiple sources:

**Application Alerts (Prometheus):**
- <count> alert files
- <count> total alerts
- Coverage: HTTP, business flows, cache, Kafka

**Infrastructure Alerts (CloudWatch):**
- RDS: <count> DB instances monitored (CPU, memory, connections, write latency)
- EKS: <if found>
- Lambda: <if found>
```

### 5.5 Check for Log-Based Alerts (NEW - Critical)

**Always ASK user about log-based monitoring:**

```
"Before I analyze gaps, I need to understand your complete monitoring setup.

Do you have log-based alerts configured in Coralogix, Datadog, or CloudWatch Logs?

Common log-based alerts:
1. Application panics/crashes
2. Error rate spikes by error type
3. Timeout patterns
4. Security events (auth failures, injection attempts)
5. Business anomalies

If yes, please share:
- Platform used (Coralogix, Datadog, CloudWatch)
- What's being monitored (e.g., panic alerts, error patterns)
- Example alert URL or screenshot (if available)
"
```

**Why this is critical:**
- Panic tracking via logs is BETTER than metrics (provides stack traces, context)
- Many error patterns are detected via log aggregation
- Avoids recommending Prometheus metrics for things already monitored in logs

**If user confirms log alerts exist:**
```
"✅ Noted: Log-based alerts exist for <list areas>
I won't recommend adding Prometheus metrics for these areas."
```

### 5.6 Check for Cloud-Native Monitoring (NEW - Critical for DB)

**Always ASK user about cloud provider monitoring:**

```
"Do you have cloud-native monitoring enabled for your infrastructure?

**For Databases:**
1. RDS Performance Insights? (Provides query-level latency, top queries, wait events)
2. CloudWatch Insights for database logs?
3. Slow query logs enabled?

**For Other Services:**
4. Application Load Balancer metrics?
5. Lambda CloudWatch metrics?
6. ElastiCache metrics?
7. AWS X-Ray for distributed tracing?

If you have Performance Insights or similar query-level monitoring, I won't recommend
adding application-level DB query metrics (redundant monitoring)."
```

**Why this is critical:**
- RDS Performance Insights provides comprehensive query-level monitoring
- Avoids recommending DB query duration metrics when better visibility already exists
- Prevents redundant instrumentation

**If user confirms Performance Insights:**
```
"✅ Noted: RDS Performance Insights provides query-level monitoring
I won't recommend adding DB query duration metrics to application code."
```

### 6. Scan Repository for Metrics

**Perform thorough repository scan:**
- **Go services:** Search for Prometheus metrics in `app/metric/`, `internal/metrics/`, `pkg/metrics/`
- **Node services:** Search for metrics in `src/metrics.js`, `lib/metrics/`
- **Locate metric definitions:** Use `grep -r "prometheus.New" .` or `grep -r "NewCounterVec\|NewHistogramVec\|NewGaugeVec" .`

Store in memory:
- All existing metrics (name, type, labels)
- Where metrics are defined (file:line)
- Business flows that emit metrics

### 6.5 Verify Existing Coverage with User (NEW - CRITICAL)

**BEFORE analyzing gaps, verify your understanding with the user:**

This step prevents false positives by confirming assumptions about existing monitoring.

**Present summary and ask for verification:**

```
"I've analyzed monitoring across multiple sources. Before identifying gaps,
let me verify my understanding:

📊 **Current Coverage Summary:**

**Application Metrics (Prometheus):**
- Found <count> metrics in <location>
- Coverage: HTTP (<list>), Business flows (<list>), Cache (<list>), Jobs (<list>)

**Infrastructure (CloudWatch):**
- RDS: <count> instances - CPU, memory, connections, write latency
- <Other infrastructure if found>

**Questions to verify:**

1. **Histogram Metrics:**
   - I see histogram metrics like `<service>_response_time`
   - These automatically provide message counts via the `_count` suffix - correct?
   - Example: `ledger_journal_create_job_pg_response_time_count` tracks message count?

2. **Job Processing:**
   - For async jobs, are success/error metrics tracked in processing functions?
   - Example: HandleHelper, ProcessMessage, ConsumeMessage functions?
   - Or are there separate counter metrics I should look for?

3. **Database Monitoring:**
   - Do you have RDS Performance Insights enabled?
   - This would provide query-level latency, top queries, wait events
   - If yes, I won't recommend adding DB query metrics to application code

4. **Log-Based Alerts:**
   - Do you have Coralogix/Datadog alerts for:
     - Application panics/crashes?
     - Error pattern detection?
   - If yes, I won't recommend adding panic metrics to Prometheus

5. **Distributed Tracing:**
   - I see OpenTracing instrumentation in the code
   - Is the tracing pipeline to Coralogix currently working or broken?
   - This helps understand if you have app-to-DB query correlation

6. **External Service Calls:**
   - For dependencies like <list external services from code>,
   - Do you track error counts? Latency? Both?

Please confirm or correct this understanding before I present gaps."
```

**Wait for user response. Only proceed to gap analysis after user confirms.**

**Common corrections from users:**
- "Histogram provides count via _count suffix" ✅
- "Job errors tracked in HandleHelper function" ✅
- "We have Performance Insights for DB queries" ✅
- "Coralogix has panic alerts via logs" ✅
- "Tracing pipeline is broken but instrumentation exists" ℹ️

**Update your understanding based on user corrections, then proceed.**

### 7. Identify Missing Metrics (After User Verification)

**ONLY after user confirms your understanding in Step 6.5**, compare existing metrics against business-critical flows.

**Important:** Use verified information from user:
- If user confirmed Performance Insights exists → Don't recommend DB query metrics
- If user confirmed Coralogix panic alerts exist → Don't recommend panic metrics
- If user confirmed histogram _count provides message tracking → Don't recommend separate counters
- If user confirmed HandleHelper tracks job errors → Don't recommend separate error counters

**If repo skill is available:**

**CRITICAL:** Start with the **Observability/Monitoring section** of the repo skill:
1. **Compare documented metrics vs. actual metrics:**
   - What metrics does the repo skill say should exist?
   - What metrics are actually implemented (from step 7)?
   - Identify the delta - these are your primary gaps
2. **Review documented monitoring gaps:**
   - Check if the observability section lists known blind spots
   - These are pre-validated gaps that need metrics
3. **Cross-reference SLIs/SLOs:**
   - For each SLI mentioned (e.g., "p99 latency < 200ms"), verify a metric exists
   - For each SLO (e.g., "99.9% uptime"), verify metrics to measure it
4. **Validate critical flows:**
   - Use the repo skill to understand all business-critical flows
   - Cross-reference each flow with existing metrics to identify gaps
   - Pay special attention to flows marked as critical or revenue-impacting
5. **Check external dependencies:**
   - Use the architecture documentation to identify all external dependencies
   - Verify each dependency has latency metrics (p99, p95, p50) and 5XX error metrics

**For all cases:**
Consult `references/metric-examples.md` for common patterns.

**Key areas to check:**
1. **HTTP endpoints** - Do all critical routes have request count and latency metrics?
2. **Error tracking** - Are all error types being counted?
3. **Business flows** - Payment creation, order processing, etc.
4. **Database operations** - Query latency, connection pool metrics
5. **Cache operations** - Hit/miss ratio
6. **Worker processing** - Kafka consumer metrics
7. **Panics/crashes** - Recovery tracking
8. **Downstream services** - Latency metrics (p99, p95, p50) and 5XX error metrics for external API calls

**Note:** External service alerts are handled by those services themselves - do not add alerts for downstream API calls. **Exception:** Always add latency metrics (p99, p95, p50) and 5XX error metrics for downstream service calls to track dependency health and SLA compliance.

**CRITICAL: High Cardinality Label Restrictions**

**NEVER use these as metric labels (causes memory exhaustion):**
- `merchant_id` - Unbounded (millions of merchants)
- `terminal_id` - Unbounded (millions of terminals)
- `user_id` - Unbounded (millions of users)
- `payment_id` - Unbounded (millions of payments)
- `order_id` - Unbounded (millions of orders)
- `transaction_id` - Unbounded
- `request_id` - Unbounded
- Any unique identifier with millions of possible values

**Safe labels (low cardinality - use these):**
- `status` - Few values (success, failure, pending, etc.)
- `method` - Few values (GET, POST, PUT, DELETE)
- `route` - Few values (API endpoints)
- `code` - Few values (HTTP status codes: 200, 404, 500)
- `gateway` - Few values (payment gateways: razorpay, paytm, etc.)
- `event_type` - Few values (payment_created, order_paid, etc.)
- `operation` - Few values (create, update, delete, etc.)
- `cache_type` - Few values (redis, memory, etc.)

**Rule:** Label cardinality should be < 100 unique values. If unbounded, DO NOT use as label.

For each missing metric, provide:
- **Metric name** (following naming conventions)
- **Type** (Counter/Histogram/Gauge)
- **Labels** (what dimensions to track - **ONLY LOW CARDINALITY**)
- **Location** (where to add in code)
- **Business need** - **Emphasize the business impact:**
  - Revenue impact (e.g., "Detect payment failures affecting GMV")
  - User experience (e.g., "Identify slow checkouts causing cart abandonment")
  - SLA compliance (e.g., "Track p99 latency to meet merchant SLA")
  - Operational efficiency (e.g., "Prevent DB connection exhaustion causing downtime")

Present the list to the user and ask for confirmation before proceeding.

### 8. Add Metrics to Application Code

After user confirmation, create a new branch:
```bash
git checkout -b add-metrics-<service-name>
```

Add metric definitions following the existing patterns:
- **Go:** Add to `app/metric/metric.go` or `internal/metrics/metrics.go`
- **Node:** Add to `src/metrics.js`

For each metric:
1. Define the metric (Counter/Histogram/Gauge)
2. Register it with Prometheus
3. Add metric increments/observations in the relevant code paths
4. Follow existing naming conventions (namespace prefix)

Commit the changes:
```bash
git add .
git commit -m "Add missing metrics for <service-name>

- Add <metric1> to track <business flow>
- Add <metric2> to track <business flow>
...

These metrics enable alerts for critical business flows.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### 9. Create Alert Rules

Navigate to alert-rules repository (using the path from step 4) and create a new branch:
```bash
cd <alert-rules-path>
git checkout -b add-alerts-<service-name>
```

Create or update the service's alert file (e.g., `rules/prod-rules/<service-name>_rules.yaml`).

For each new metric, create appropriate alerts using patterns from `references/alert-patterns.md`:

**Alert structure:**
```yaml
groups:
  - name: <Group Name>
    interval: 60s
    rules:
      - alert: "<Alert Description>"
        expr: <PromQL query with threshold>
        for: <duration>
        labels:
          severity: critical
          bu: <business-unit>
          pod: <pod-name>
          service: <service-name>
          live: true
          slack_channel: "<channel>"
        annotations:
          identifier: <unique_id>
          description: <description with {{$value}}>
          Runbook: <google-doc-url>
          vajra_link: <vajra-dashboard-url>
```

**Determine appropriate thresholds based on:**
- Existing alerts for similar metrics
- Business criticality
- Service traffic patterns
- SLA requirements

Commit the changes:
```bash
git add .
git commit -m "Add alert rules for <service-name> metrics

- Add alerts for <metric1>
- Add alerts for <metric2>
...

These alerts monitor critical business flows and prevent incidents.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### 10. Push Branches and Create PRs

Push application repo branch:
```bash
cd <application-repo>
git push -u origin add-metrics-<service-name>
```

Create PR for application repo:
```bash
gh pr create --title "Add missing metrics for <service-name>" --body "## Summary
- Added <count> new metrics to track critical business flows

## Metrics Added
- \`<metric1>\` - <description>
- \`<metric2>\` - <description>
...

## Business Impact
- Enables monitoring of <flow>
- Prevents <incident type>
- Tracks <SLA/KPI>

## Testing
- [ ] Metrics verified in local Prometheus
- [ ] Metric names follow conventions
- [ ] Labels are appropriate

🤖 Generated with Claude Code"
```

Push alert-rules repo branch:
```bash
cd <alert-rules-path>
git push -u origin add-alerts-<service-name>
```

Create PR for alert-rules repo:
```bash
gh pr create --title "Add alert rules for <service-name>" --body "## Summary
- Added <count> alert rules for new <service-name> metrics

## Alerts Added
- <alert1> - Threshold: <value>
- <alert2> - Threshold: <value>
...

## Coverage
- [x] HTTP errors (4xx/5xx)
- [x] Latency (p99/p95/p90/p50)
- [x] Business flows
- [x] External services
- [x] Resource utilization

## Related PR
Application repo: <link-to-metrics-pr>

🤖 Generated with Claude Code"
```

### 11. Handle Failures

If PR creation fails, diagnose and resolve using the troubleshooting guide.

**Common failure scenarios:**
- Authentication failures (`gh auth status` → re-authenticate)
- Branch already exists remotely (delete and re-push)
- Permission issues (verify write access)
- Network connectivity problems

**For detailed troubleshooting steps, see:**
- [Troubleshooting Guide](references/troubleshooting.md) - Complete failure scenarios, retry logic, and manual fallback instructions

### 12. Provide PR Links

Display PR links to user:
```
✅ Application Metrics PR: <pr-url>
✅ Alert Rules PR: <pr-url>

Please review the PRs and ensure:
1. Metrics are instrumented correctly in the code
2. Alert thresholds are appropriate for production traffic
3. Runbook links are added (if missing)
4. Slack channels are correct
```

# Monitoring Sources Checklist

When analyzing alert coverage, check **ALL** these sources before identifying gaps. Many services use multi-tool monitoring strategies.

---

## 1. Application Metrics (Prometheus)

**Location:** Application codebase
- Go: `internal/provider/metric/metric.go`, `internal/boot/hooks/metric.go`, `app/metric/`
- Node: `src/metrics.js`, `lib/metrics/`
- Python: `app/metrics.py`, `metrics/`

**Alert files:** `alert-rules/rules/prod-rules/<service>_*.yaml`

**What to check:**
- Counter metrics (error counts, request counts)
- Histogram metrics (latency distributions)
- Gauge metrics (current values, pool sizes)
- Metric naming conventions (service prefix)

**Common patterns:**
```go
// Go
metric.NewPrometheusService(metric.Details{
    Namespace: "ledger",
    Name: "http_requests_total",
    MetricType: metric.Counter,
    Labels: []string{"method", "status"},
})
```

---

## 2. Infrastructure Metrics (CloudWatch/Cloud Provider)

**Alert files:**
- RDS: `alert-rules/rules/prod-rules/rds_generated_alerts_*<service>*.yaml`
- EKS/Containers: `alert-rules/rules/prod-rules/eks_*<service>*.yaml`
- Lambda: `alert-rules/rules/prod-rules/lambda_*<service>*.yaml`
- ElastiCache: `alert-rules/rules/prod-rules/elasticache_*<service>*.yaml`

**Always ASK user:**
```
"Do you have infrastructure monitoring enabled via CloudWatch/Cloud provider?
- RDS instances for this service?
- EKS/container metrics?
- Lambda functions?
- ElastiCache clusters?"
```

**Common CloudWatch RDS metrics:**
- `aws_rds_cpuutilization_maximum` - CPU usage
- `aws_rds_freeable_memory_maximum` - Available memory
- `aws_rds_database_connections_maximum` - Active connections
- `aws_rds_write_latency_maximum` - Write query latency
- `aws_rds_read_latency_maximum` - Read query latency
- `aws_rds_read_iops_maximum` - Read operations/sec
- `aws_rds_write_iops_maximum` - Write operations/sec
- `aws_rds_disk_queue_depth_maximum` - Pending I/O operations

---

## 3. Cloud-Native Database Monitoring

**Always ASK user:**
```
"Do you have database monitoring enabled?
- RDS Performance Insights?
- CloudWatch Insights for queries?
- Slow query logs?
- Query-level latency tracking?"
```

**RDS Performance Insights provides:**
- Top SQL queries by execution time
- Per-query latency (avg, max, p99)
- Wait event analysis (I/O, locks, CPU)
- Database load metrics
- Query execution counts

**Important:** If Performance Insights is enabled, **DO NOT** recommend adding query-level metrics to application code - the visibility already exists.

---

## 4. Log-Based Alerts (Coralogix/Datadog/CloudWatch Logs)

**Always ASK user:**
```
"Do you have log-based alerts configured in Coralogix, Datadog, or other platforms?

Common log alerts:
- Application panics/crashes
- Error rate spikes by error type
- Timeout patterns
- Security events (auth failures, SQL injection attempts)
- Business anomalies (unusual transaction patterns)"
```

**Why this matters:**
- Panic tracking via logs is better than metrics (provides stack traces)
- Error pattern detection often uses log aggregation
- Security events are typically log-based

**Example Coralogix alert:**
```
Alert: "Ledger Service Panics"
Query: message:"panic" AND service:"ledger"
Threshold: > 0 in 5 minutes
```

**Important:** If log-based alerts exist, **DO NOT** recommend adding equivalent Prometheus metrics.

---

## 5. Distributed Tracing

**Always ASK user:**
```
"Do you have distributed tracing enabled?
- OpenTracing/Jaeger setup?
- Tracing pipeline to Coralogix/Datadog/AWS X-Ray?
- Trace-based alerts?
- DB query tracing visibility?"
```

**What tracing provides:**
- Request flow across services
- DB query latency with context
- Cache operation performance
- External service call tracking
- Correlation between app operations and infrastructure

**Important:** If tracing is enabled (even if pipeline is broken), you already have instrumentation - just need to fix the pipeline, not add metrics.

---

## 6. Repo Skills (Service Documentation)

**MANDATORY CHECK:** Before analyzing gaps, check for repo skill:

```bash
ls -la .agents/skills/
```

**If a skill exists (e.g., `<service-name>-skill/`):**

### **MUST READ - Observability Module:**
```
.agents/skills/<service>-skill/modules/technical/observability/
├── metrics.md          # Existing metrics inventory
├── logging.md          # Log patterns and alerts
└── tracing.md          # Tracing setup
```

**What to look for:**
- **Existing metrics list** - What's already instrumented
- **SLIs/SLOs** - Service level objectives requiring metrics
- **Known gaps** - Documented monitoring blind spots
- **Metric naming conventions** - Service-specific patterns
- **Alert escalation policies** - Severity levels and channels

### **Also useful modules:**
```
.agents/skills/<service>-skill/modules/
├── technical/database/          # DB patterns, connection pooling
├── technical/integrations/      # External service calls
│   ├── kafka/                   # Kafka consumer patterns
│   └── redis/                   # Cache patterns
└── domain/                      # Business flows requiring monitoring
```

**Important:** The repo skill is the **authoritative source** for what monitoring should exist. Always cross-reference it.

---

## 7. Custom Monitoring Tools

**Always ASK user:**
```
"Do you use any custom monitoring/alerting tools?
- Internal monitoring platforms
- APM tools (New Relic, Dynatrace)
- Business intelligence dashboards
- Custom alerting scripts"
```

---

## Verification Workflow

### Step 1: Discover All Sources

For each source above:
1. Check if it exists (file search, ask user)
2. If exists, read/document what's monitored
3. Note coverage (what metrics/alerts exist)

### Step 2: Create Coverage Matrix

| Monitoring Area | Source | Coverage | Gaps |
|----------------|--------|----------|------|
| HTTP API | Prometheus | ✅ Requests, latency, errors | None |
| Database Infra | CloudWatch RDS | ✅ CPU, memory, connections, write latency | ❌ Read latency |
| Database Queries | Performance Insights | ✅ Query-level analysis | None |
| Application Crashes | Coralogix Logs | ✅ Panic alerts | None |
| Kafka Consumers | Prometheus | ✅ Lag, processing time, errors | None |
| External Services | Prometheus | ✅ Error counts | ❌ Latency |

### Step 3: Verify With User

**BEFORE presenting gaps, verify assumptions:**

```
"I've analyzed monitoring across multiple sources. Before recommending additions,
let me verify my understanding:

1. **Prometheus Metrics:**
   - Found ~600 metrics in internal/provider/metric/metric.go
   - Histogram metrics provide message counts via _count suffix - correct?
   - Job success/errors tracked in HandleHelper functions - correct?

2. **Infrastructure Monitoring:**
   - Found RDS CloudWatch alerts for 5 DB instances
   - Do you have Performance Insights enabled for query-level visibility?

3. **Log-Based Alerts:**
   - Do you have Coralogix alerts for panics/crashes?
   - Do you have error pattern detection in logs?

4. **Distributed Tracing:**
   - Found OpenTracing instrumentation in code
   - Is the tracing pipeline to Coralogix currently working?

Please confirm or correct this understanding."
```

### Step 4: Only Then Present Gaps

After user confirms, present **ACTUAL** gaps not covered by any source.

---

## Common Mistakes to Avoid

### ❌ Mistake 1: Assuming Prometheus is the only monitoring
**Reality:** Most services use 3-5 monitoring tools
- Prometheus for app metrics
- CloudWatch for infrastructure
- Coralogix for logs and traces
- Performance Insights for DB queries

### ❌ Mistake 2: Not understanding histogram metrics
**Wrong:** "No message count metric found"
**Right:** Histogram `_response_time` automatically creates `_response_time_count`

### ❌ Mistake 3: Recommending DB query metrics when Performance Insights exists
**Wrong:** "Add DB query duration metric"
**Right:** "Performance Insights already provides query-level latency - no metric needed"

### ❌ Mistake 4: Ignoring repo skill documentation
**Wrong:** Scan code without context
**Right:** Read `.agents/skills/<service>-skill/modules/technical/observability/` FIRST

### ❌ Mistake 5: Not asking about log-based alerts
**Wrong:** "Add panic metric to Prometheus"
**Right:** "Do you have Coralogix log alerts for panics? (You do) - No metric needed"

---

## Examples from Real Services

### Example 1: Ledger Service

**Multi-source monitoring discovered:**
- ✅ Prometheus: ~600 metrics, 403 alerts
- ✅ CloudWatch RDS: 5 DB instances (CPU, memory, connections, write latency)
- ✅ Performance Insights: Query-level DB monitoring
- ✅ Coralogix: Log-based panic alerts
- ❌ Distributed Tracing: Pipeline broken (but instrumentation exists)

**Actual gaps (only 2):**
1. External service latency (Splitz, WDA)
2. Broken tracing pipeline

**False positives avoided (3):**
- ❌ DB query metrics (Performance Insights covers this)
- ❌ Panic metrics (Coralogix log alerts cover this)
- ❌ Kafka consumer metrics (Histogram _count + HandleHelper cover this)

---

## Checklist Summary

Before presenting gaps, verify:

- [ ] Read repo skill observability modules (if available)
- [ ] Checked Prometheus metrics in application code
- [ ] Searched for infrastructure alert files (RDS, EKS, Lambda)
- [ ] Asked user about Performance Insights / cloud DB monitoring
- [ ] Asked user about Coralogix / log-based alerts
- [ ] Asked user about distributed tracing setup
- [ ] Verified assumptions about histogram metrics (_count suffix)
- [ ] Verified assumptions about job processing (HandleHelper patterns)
- [ ] Created coverage matrix across all sources
- [ ] Presented findings to user for verification BEFORE final gap analysis

**Only after all checks, present actual gaps.**

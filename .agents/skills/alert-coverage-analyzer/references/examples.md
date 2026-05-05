# Detailed Examples - Alert Coverage Analyzer

## Example 1: Complete Workflow

**User:** "Analyze alert coverage for this service"

**Skill Response:**
```
✅ Verified Razorpay repository
✅ Identified service: payment-links
✅ GitHub CLI authenticated

Do you already have the razorpay/alert-rules repository cloned? If yes, please provide the path.
```

**User:** "Yes, /Users/myuser/razorpay/alert-rules"

**Skill Response:**
```
✅ Verified alert-rules repository at /Users/myuser/razorpay/alert-rules

Do you have a repo skill created for this service?
If yes, it will help me understand the codebase structure, business flows, and existing patterns better.
```

**User:** "Yes, payment-links-skill"

**Skill Response:**
```
✅ Loading payment-links-skill for contextual analysis
✅ Found Observability/Monitoring section with:
   - 12 documented SLIs/SLOs
   - 3 known monitoring gaps
   - Metric naming convention: payment_links_*
   - Critical flows: payment creation, refund processing, webhook delivery

Reading existing alerts...
Found: /Users/myuser/razorpay/alert-rules/rules/prod-rules/payment_links_rules.yaml
- 8 alert groups
- 32 active alerts
- Coverage: HTTP errors, latency (p99, p95), pod count, CPU/memory

Scanning repository for metrics...
Found 45 existing metrics in app/metric/metric.go

Cross-referencing with repo skill observability section...
- ✅ 8/12 SLIs have metrics
- ❌ 4/12 SLIs missing metrics
- ❌ 3 known monitoring gaps confirmed (from repo skill)
- ✅ All critical flows have some coverage
- ❌ 2 critical flows missing detailed metrics

Analyzing coverage gaps...

⚠️  Found 5 missing business-critical metrics (3 from repo skill observability gaps, 2 from code analysis):

1. payment_links_refund_processing_duration_ms
   Type: Histogram
   Labels: status, gateway
   Location: app/services/refund_service.go:125
   Business Impact: Track refund SLA compliance - 95% refunds must complete <5s per merchant agreement

2. payment_links_webhook_delivery_failure_total
   Type: Counter
   Labels: event_type, status_code
   Location: app/workers/webhook_worker.go:78
   Business Impact: Detect webhook failures preventing customer notifications - affects merchant trust

3. payment_links_payment_method_distribution
   Type: Counter
   Labels: method, status
   Location: app/services/payment_service.go:201
   Business Impact: Track payment method adoption - informs gateway partnership decisions

4. payment_links_cache_eviction_total
   Type: Counter
   Labels: cache_type, reason
   Location: app/cache/redis_cache.go:89
   Business Impact: Detect cache thrashing causing increased DB load and slower API responses

5. payment_links_db_query_slow_total
   Type: Counter
   Labels: operation, table
   Location: app/database/repository.go:203
   Business Impact: Track slow queries (>100ms) causing API latency - prevents P99 SLA violations

Proceed with adding these metrics and creating alerts? (y/n)
```

**User:** "y"

**Skill Response:**
```
Creating branch: add-metrics-payment-links
✅ Added 5 metrics to app/metric/metric.go
✅ Instrumented metrics in 5 locations
✅ Committed changes

Creating branch in alert-rules: add-alerts-payment-links
✅ Added 15 alert rules (3 alerts per metric: critical threshold, warning, trend)
✅ Committed alert rules

Pushing branches...
✅ Pushed add-metrics-payment-links to razorpay/payment-links
✅ Pushed add-alerts-payment-links to razorpay/alert-rules

Creating PRs...
✅ Application Metrics PR: https://github.com/razorpay/payment-links/pull/1234
✅ Alert Rules PR: https://github.com/razorpay/alert-rules/pull/5678

Please review the PRs and ensure:
1. Metrics are instrumented correctly in the code
2. Alert thresholds are appropriate for production traffic
3. Runbook links are added (if missing)
4. Slack channels are correct
```

## Example 2: Non-Razorpay Repository

**User:** "Check alert coverage"

**Skill Response:**
```
❌ This is not a Razorpay repository. This skill only works with Razorpay repos.

Git remote: https://github.com/other-org/some-project
```

## Example 3: Service Name Unknown

**User:** "Analyze alerts"

**Skill Response:**
```
✅ Verified Razorpay repository
⚠️  Could not auto-detect service name

Checked:
- go.mod: not found
- package.json: not found
- app/metric/metric.go: not found

Please provide the service name (e.g., terminals, router, pg-router):
```

**User:** "payment-links"

**Skill Response:**
```
✅ Using service name: payment-links
Proceeding with analysis...
```

## Example 4: With Multi-Source Monitoring Verification

**User:** "Analyze alert coverage for ledger"

**Skill Response:**
```
✅ Verified Razorpay repository: ledger
✅ Found ledger-skill with observability documentation

Reading existing alerts...
**Application Alerts (Prometheus):**
- 9 alert files
- 403 total alerts
- Coverage: HTTP, business flows, cache, Kafka

**Infrastructure Alerts (CloudWatch):**
- RDS: 5 DB instances monitored (CPU, memory, connections, write latency)

I've analyzed monitoring across multiple sources. Before identifying gaps,
let me verify my understanding:

📊 **Current Coverage Summary:**

**Application Metrics (Prometheus):**
- Found ~600 metrics in internal/provider/metric/metric.go
- Coverage: HTTP requests, Business flows, Cache, Jobs

**Infrastructure (CloudWatch):**
- RDS: 5 instances - CPU, memory, connections, write latency

**Questions to verify:**

1. **Histogram Metrics:**
   - I see histogram metrics like `ledger_journal_create_job_pg_response_time`
   - These automatically provide message counts via the `_count` suffix - correct?

2. **Job Processing:**
   - For async jobs, are success/error metrics tracked in HandleHelper functions?

3. **Database Monitoring:**
   - Do you have RDS Performance Insights enabled for query-level visibility?

4. **Log-Based Alerts:**
   - Do you have Coralogix alerts for panics/crashes?

5. **Distributed Tracing:**
   - Is the tracing pipeline to Coralogix working or broken?

Please confirm or correct this understanding before I present gaps.
```

**User:** "Yes, histogram provides count. HandleHelper tracks errors. Performance Insights enabled. Coralogix has panic alerts. Tracing pipeline is broken."

**Skill Response:**
```
✅ Understanding confirmed. Analyzing actual gaps...

**Verified Coverage:**
- ✅ Kafka message counts (histogram _count)
- ✅ Job errors (HandleHelper tracking)
- ✅ DB query monitoring (Performance Insights)
- ✅ Panic alerts (Coralogix logs)

**Actual Gaps Found (2):**

1. External service latency tracking
   - Missing: Splitz, WDA call latency metrics (p99, p95, p50)
   - Reason: Track dependency health and SLA compliance

2. Broken distributed tracing pipeline
   - Issue: OpenTracing instrumentation exists but pipeline not working
   - Fix: Configuration/pipeline repair (no new instrumentation needed)

Proceed with adding external service metrics? (y/n)
```

## Example 5: First-Time Alert Setup

**User:** "Set up monitoring for new service"

**Skill Response:**
```
✅ Verified Razorpay repository: new-service
⚠️  No existing alert files found for new-service
⚠️  No repo skill found

This appears to be first-time alert setup. I'll analyze the codebase to recommend
comprehensive monitoring coverage.

**Recommendation:** Create a repo skill first for better analysis accuracy.
Proceed without repo skill? (y/n)
```

**User:** "y"

**Skill Response:**
```
Scanning repository for patterns...
Found:
- 12 HTTP endpoints
- 3 database tables
- 2 Kafka consumers
- 1 Redis cache integration

Analyzing critical flows from code...

**Recommended Metrics (15):**

**HTTP Monitoring (3 metrics):**
1. new_service_http_requests_total (Counter)
2. new_service_http_durations_ms_histogram (Histogram)
3. new_service_rzp_errors (Counter)

**Database (2 metrics):**
4. new_service_db_query_duration (Histogram)
5. new_service_db_connection_pool (Gauge)

**Cache (2 metrics):**
6. new_service_cache_hit_and_miss (Counter)
7. new_service_cache_latency (Histogram)

**Kafka (3 metrics):**
8. new_service_kafka_consumer_message_consumption_total (Counter)
9. new_service_kafka_consumer_message_processed_failure_total (Counter)
10. new_service_kafka_consumer_message_processing_duration (Histogram)

**Business Flows (5 metrics - identified from code):**
11-15. <service-specific metrics based on code analysis>

Proceed with adding these metrics and creating alerts? (y/n)
```

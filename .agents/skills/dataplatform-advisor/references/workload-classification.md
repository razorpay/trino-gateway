# Workload Classification Framework

This document provides a comprehensive framework for classifying data workloads to guide system selection decisions.

## Overview

Choosing the right data system starts with understanding your workload characteristics. This framework helps you classify workloads across six key dimensions and map them to appropriate systems.

## The Six Dimensions

### 1. Latency Requirements

**Question**: What is the required p99 latency for queries?

**Categories:**

| Category | Range | Example Use Cases | Suitable Systems |
|----------|-------|-------------------|------------------|
| Ultra-low | < 10ms | User authentication, fraud detection | Aurora, DynamoDB |
| Low | 10-100ms | User-facing dashboards, APIs | TiDB, Pinot, ClickHouse |
| Medium | 100ms-10s | Internal dashboards, BI tools | Trino, OLAP |
| High | 10s-minutes | Batch analytics, reports | Trino, Spark |
| Very High | Minutes-hours | ETL, ML training | Spark |

**Measurement Tips:**
- Measure at p99, not average (p50)
- Consider end-to-end latency (network + query)
- Account for query variability (simple vs complex)

**Common Mistakes:**
- Using average latency (hides tail latencies)
- Not accounting for concurrent load
- Optimizing for p50 when users experience p99

### 2. Data Freshness

**Question**: How fresh must the data be?

**Categories:**

| Category | Lag | Pipeline Type | Suitable Systems |
|----------|-----|---------------|------------------|
| Real-time | < 1 second | Direct writes | Aurora, DynamoDB, TiDB |
| Near real-time | 1-60 seconds | CDC, streaming | TiDB, Pinot, ClickHouse |
| Minutes | 1-30 minutes | Micro-batches | Pinot, Iceberg (streaming) |
| Hours | 1-12 hours | Batch ETL | Iceberg, data warehouse |
| T-1 (previous day) | 12-24 hours | Daily batch | Iceberg, data warehouse |

**Measurement Tips:**
- Define freshness as "time from source event to queryable"
- Consider acceptable staleness for business decisions
- Distinguish between write and read freshness

**Example Scenarios:**

*E-commerce fraud detection:*
- Required: < 1 second
- Rationale: Must block fraudulent transactions before settlement
- System: Direct writes to DynamoDB, Flink for real-time scoring

*Merchant dashboard (payment stats):*
- Required: 5-10 minutes acceptable
- Rationale: Merchants check periodically, not second-by-second
- System: CDC → Pinot (real-time ingestion)

*Monthly financial reporting:*
- Required: T-1 (previous day)
- Rationale: Reports generated once per month
- System: Daily batch to Iceberg

### 3. Concurrency

**Question**: How many concurrent users/queries?

**Categories:**

| Category | Concurrent Queries | QPS | Suitable Systems |
|----------|-------------------|-----|------------------|
| Very Low | 1-10 | < 1 | Any system |
| Low | 10-50 | 1-10 | Trino, TiDB |
| Medium | 50-500 | 10-100 | TiDB, Pinot, ClickHouse |
| High | 500-5000 | 100-1000 | Pinot, ClickHouse |
| Very High | > 5000 | > 1000 | Pinot, ClickHouse (scaled) |

**Measurement Tips:**
- Peak concurrent users, not average
- Consider query duration (long queries hold resources)
- Account for spikes (e.g., dashboard refresh at top of hour)

**System Capabilities:**

*Trino:*
- Shared cluster: 10-50 concurrent queries
- Higher concurrency causes queuing
- Not suitable for user-facing APIs

*Pinot:*
- Designed for 1000+ QPS
- Sub-second latency maintained under load
- Excellent for user-facing dashboards

*TiDB:*
- Handles 100s of concurrent connections
- Row-oriented (good for point queries)
- Higher cost than column stores

### 4. Query Shape

**Question**: Are queries fixed patterns or ad-hoc?

**Categories:**

| Category | Description | Examples | Suitable Systems |
|----------|-------------|----------|------------------|
| Fixed | Predefined queries, parameterized | Dashboard metrics, API endpoints | OLAP (Pinot, ClickHouse) |
| Semi-fixed | Known patterns, variable filters | BI reports with user filters | TiDB, OLAP |
| Ad-hoc | Unpredictable queries, exploration | Data analysis, debugging | Trino, Spark |
| Mixed | Combination of above | Internal analytics platform | Trino + Pinot |

**Fixed Query Example:**
```sql
-- Dashboard: Merchant payment volume by day
SELECT
  DATE(created_at) as day,
  SUM(amount) as total_amount,
  COUNT(*) as txn_count
FROM payments
WHERE merchant_id = ?
  AND created_at >= ?
  AND created_at < ?
GROUP BY DATE(created_at)
ORDER BY day DESC
```

**Ad-hoc Query Example:**
```sql
-- Analyst exploring anomalies
SELECT
  merchant_id,
  payment_method,
  COUNT(*) as count,
  AVG(amount) as avg_amount
FROM payments
WHERE created_at >= DATE '2024-01-01'
  AND status = 'FAILED'
  AND amount > 10000
GROUP BY merchant_id, payment_method
HAVING COUNT(*) > 100
ORDER BY count DESC
```

**System Implications:**

*OLAP (Pinot/ClickHouse):*
- Optimized for fixed queries
- Uses pre-built indexes (star-tree, inverted)
- Struggles with arbitrary joins or filters

*Trino:*
- Handles any SQL query
- No pre-built indexes (full scan with predicate pushdown)
- Variable latency depending on query complexity

### 5. Consistency Requirements

**Question**: Need read-after-write guarantees?

**Categories:**

| Category | Description | Use Cases | Suitable Systems |
|----------|-------------|-----------|------------------|
| Strong | Read-after-write guaranteed | Transactional operations, user profile updates | Aurora, TiDB |
| Eventual | Eventually consistent, stale reads OK | Analytics, reporting | Iceberg, Pinot, ClickHouse |
| Session | Consistent within user session | Application state | Aurora, DynamoDB |
| Causal | Respects causal ordering | Event processing | Kafka, Flink |

**Example Scenarios:**

*User updates profile → immediately queries it:*
- Required: Strong consistency (read-after-write)
- System: Aurora or TiDB

*Analyst queries yesterday's payment data:*
- Required: Eventual consistency (data settled overnight)
- System: Iceberg + Trino

*Dashboard shows "processing" transactions:*
- Required: Eventual consistency (5-10 min lag OK)
- System: CDC → Pinot

### 6. Data Quality / Error Tolerance

**Question**: What error tolerance is acceptable?

**Categories:**

| Category | Accuracy | Use Cases | Suitable Systems |
|----------|----------|-----------|------------------|
| Exact | 100% accurate, no data loss | Financial transactions, compliance | Aurora (ACID), Iceberg (with validation) |
| High | < 0.1% error rate | Business reporting, SLAs | All systems with strong validation |
| Medium | < 1% error rate | Internal dashboards, trends | CDC pipelines (at-least-once) |
| Approximate | 1-5% error rate acceptable | Exploratory analysis, AB tests | Sampling, approximate algorithms |

**Error Sources:**

*CDC Pipelines:*
- At-least-once delivery → duplicates possible
- Solution: Idempotent consumers, deduplication

*Streaming Aggregations:*
- Late-arriving events → undercounting
- Solution: Watermarks, late-data handling

*Data Quality Issues:*
- Schema mismatches → parsing failures
- Solution: Schema validation, dead-letter queues

---

## Workload Classification Questionnaire

Use this questionnaire to classify a new workload:

```yaml
workload_classification:
  # Basic information
  name: "Merchant Payment Dashboard"
  owner: "Payments Team"
  persona: "Merchant (external user)"

  # Dimension 1: Latency
  latency:
    required_p99: "< 100ms"
    category: "Low"
    notes: "User-facing dashboard, interactive"

  # Dimension 2: Freshness
  freshness:
    required: "5-10 minutes"
    category: "Near real-time"
    notes: "Merchants check stats periodically"

  # Dimension 3: Concurrency
  concurrency:
    peak_users: 5000
    peak_qps: 500
    category: "High"
    notes: "10K merchants, ~50% check daily"

  # Dimension 4: Query Shape
  query_shape:
    type: "Fixed"
    patterns:
      - "Payment volume by day (last 30 days)"
      - "Top payment methods"
      - "Success rate by hour"
    notes: "Pre-defined charts, parameterized by merchant_id"

  # Dimension 5: Consistency
  consistency:
    required: "Eventual"
    notes: "5-10 min lag acceptable, no read-after-write needed"

  # Dimension 6: Data Quality
  data_quality:
    required: "High (< 0.1% error)"
    notes: "Merchant-facing data must be accurate"

  # Computed archetype
  archetype: "User-Facing Dashboard"
  recommended_system: "OLAP (Pinot)"
```

---

## Workload Archetypes

Based on the six dimensions, workloads typically fall into these archetypes:

### Archetype 1: User-Facing Dashboard

**Characteristics:**
- Latency: < 100ms (low)
- Freshness: Minutes (near real-time)
- Concurrency: High (100s-1000s QPS)
- Query Shape: Fixed patterns
- Consistency: Eventual
- Quality: High

**Recommended System**: **OLAP (Pinot, ClickHouse)**

**Data Flow:**
```
Source DB → CDC → Kafka → Spark (pre-join, denormalize) → Pinot
```

**Examples:**
- Merchant payment dashboard
- Customer transaction history
- Real-time analytics dashboards

**Why OLAP?**
- Sub-second latency under high concurrency
- Star-tree indexes for fixed query patterns
- Real-time ingestion from Kafka
- Cost-efficient at scale

### Archetype 2: Ad-hoc Exploration

**Characteristics:**
- Latency: Seconds to minutes (medium-high)
- Freshness: Hours to T-1 (batch)
- Concurrency: Low (1-50 users)
- Query Shape: Ad-hoc, unpredictable
- Consistency: Eventual
- Quality: High

**Recommended System**: **Trino + Iceberg**

**Data Flow:**
```
Source DB → Batch ETL → Iceberg → Trino (interactive queries)
```

**Examples:**
- Data analyst exploration
- Business intelligence tools (Metabase, Looker)
- Debugging production issues

**Why Trino + Iceberg?**
- Handles any SQL query (joins, subqueries, CTEs)
- Iceberg provides low-cost storage
- Trino offers interactive latency (seconds)
- No need for pre-built indexes

### Archetype 3: Operational Reporting

**Characteristics:**
- Latency: Sub-second to seconds
- Freshness: Minutes (near real-time)
- Concurrency: Medium (10-100 users)
- Query Shape: Semi-fixed (parameterized)
- Consistency: Eventual or strong
- Quality: High

**Recommended System**: **TiDB or OLAP**

**Data Flow (TiDB):**
```
Source DB → CDC → Kafka → Flink → TiDB
```

**Data Flow (OLAP):**
```
Source DB → CDC → Kafka → Spark → Pinot
```

**Examples:**
- Internal operations dashboard
- Customer support tools
- Transaction monitoring

**Why TiDB?**
- MySQL-compatible (easy migration)
- Row-oriented (good for point queries)
- Strong consistency if needed

**Why OLAP?**
- Better for high concurrency
- Lower cost at scale
- Real-time ingestion

### Archetype 4: Historical Analytics

**Characteristics:**
- Latency: Minutes to hours
- Freshness: Hours to T-1
- Concurrency: Low (batch jobs)
- Query Shape: Ad-hoc or batch
- Consistency: Eventual
- Quality: Exact

**Recommended System**: **Iceberg + Trino/Spark**

**Data Flow:**
```
Source DB → Batch ETL → Iceberg → {Trino (interactive) | Spark (batch)}
```

**Examples:**
- Monthly financial reports
- Historical trend analysis
- Data science feature engineering

**Why Iceberg?**
- Unlimited retention (low-cost S3)
- ACID transactions for correctness
- Time travel for audits
- Schema evolution

### Archetype 5: Batch Processing / ETL

**Characteristics:**
- Latency: Hours (not latency-sensitive)
- Freshness: N/A (generates derived data)
- Concurrency: N/A (batch jobs)
- Query Shape: Fixed transformations
- Consistency: N/A
- Quality: Exact

**Recommended System**: **Spark**

**Data Flow:**
```
Multiple sources → Spark → Iceberg/OLAP/TiDB
```

**Examples:**
- ETL pipelines
- Data quality checks
- Backfills and migrations
- ML feature engineering

**Why Spark?**
- Optimized for throughput, not latency
- Rich transformation APIs (SQL, DataFrame)
- Fault-tolerant (checkpoints, retries)
- Integrates with all data systems

### Archetype 6: Real-time Alerting

**Characteristics:**
- Latency: Milliseconds to seconds
- Freshness: Real-time
- Concurrency: Low (alert rules)
- Query Shape: Fixed (threshold checks)
- Consistency: Real-time
- Quality: High

**Recommended System**: **Flink or OLAP**

**Data Flow (Flink):**
```
Source → Kafka → Flink (CEP) → Alert Service
```

**Data Flow (OLAP):**
```
Source → Kafka → Pinot → Alert Service (polling)
```

**Examples:**
- Fraud detection
- Anomaly detection
- SLA monitoring

**Why Flink?**
- Real-time stream processing
- Complex event processing (CEP)
- Stateful computations

**Why OLAP?**
- Simple alert rules (threshold queries)
- No need for complex stream processing

### Archetype 7: Transactional Access

**Characteristics:**
- Latency: Milliseconds
- Freshness: Real-time (read-after-write)
- Concurrency: High
- Query Shape: Point lookups, updates
- Consistency: Strong (ACID)
- Quality: Exact

**Recommended System**: **Aurora or TiDB**

**Data Flow:**
```
Application → TiDB (read + write)
```

**Examples:**
- User profile management
- Inventory management
- Order processing

**Why Aurora/TiDB?**
- ACID transactions
- Read-after-write consistency
- Point query optimization (B-tree indexes)

---

## Decision Tree

Use this decision tree to quickly map to an archetype:

```
1. Do you need ACID transactions or writes?
   YES → Transactional Access (TiDB/Aurora)
   NO → Continue

2. Do you need latency < 100ms?
   YES → Is it user-facing with high concurrency?
         YES → User-Facing Dashboard (OLAP)
         NO → Operational Reporting (TiDB)
   NO → Continue

3. Are queries fixed patterns or ad-hoc?
   FIXED → Is data real-time?
           YES → User-Facing Dashboard (OLAP)
           NO → Historical Analytics (Iceberg + Trino)
   AD-HOC → Ad-hoc Exploration (Trino + Iceberg)

4. Is this for ETL/transformation?
   YES → Batch Processing (Spark)
   NO → Real-time Alerting (Flink)
```

---

## Examples

### Example 1: Payment Reconciliation Dashboard

**Requirements:**
- Show payment reconciliation status for finance team
- Latency: < 2 seconds OK (internal tool)
- Freshness: T-1 (runs daily at 2 AM)
- Concurrency: 5-10 finance analysts
- Query Shape: Ad-hoc (filtering by merchant, date, status)
- Consistency: Eventual
- Quality: Exact (financial data)

**Classification:**
```yaml
archetype: "Ad-hoc Exploration"
recommended_system: "Trino + Iceberg"
rationale:
  - Low concurrency (10 users)
  - Ad-hoc queries (variable filters)
  - T-1 freshness (batch ingestion)
  - Exact accuracy required
```

### Example 2: Fraud Detection API

**Requirements:**
- Real-time fraud scoring for transactions
- Latency: < 50ms (blocks payment)
- Freshness: Real-time (during payment)
- Concurrency: 1000+ TPS
- Query Shape: Fixed (lookup + scoring)
- Consistency: Strong (read-after-write)
- Quality: Exact

**Classification:**
```yaml
archetype: "Transactional Access"
recommended_system: "TiDB or DynamoDB"
rationale:
  - Ultra-low latency required
  - Real-time consistency needed
  - High throughput (1000 TPS)
  - Fixed access pattern (point lookup)
```

### Example 3: Marketing Campaign Analytics

**Requirements:**
- Campaign performance metrics for marketing team
- Latency: < 500ms (dashboard charts)
- Freshness: 10-15 minutes OK
- Concurrency: 50-100 marketers
- Query Shape: Fixed (pre-defined KPIs)
- Consistency: Eventual
- Quality: High

**Classification:**
```yaml
archetype: "User-Facing Dashboard"
recommended_system: "OLAP (Pinot)"
rationale:
  - Fixed queries (KPIs)
  - Medium-high concurrency (100 users)
  - Near real-time freshness (15 min)
  - Sub-second latency required
```

---

## Anti-Patterns

### Anti-Pattern 1: Using OLAP for Ad-hoc Exploration

**Problem:** OLAP systems (Pinot, ClickHouse) require fixed schemas and queries.

**Symptoms:**
- Queries fail with "column not indexed"
- Need to add indexes for each new query
- High operational overhead

**Solution:** Use Trino + Iceberg for ad-hoc exploration.

### Anti-Pattern 2: Using Trino for User-Facing APIs

**Problem:** Trino has variable latency and shared resources.

**Symptoms:**
- API timeouts during peak load
- Unpredictable response times
- Poor user experience

**Solution:** Use OLAP (Pinot) or TiDB for user-facing APIs.

### Anti-Pattern 3: Using Aurora for Analytics

**Problem:** Aurora is optimized for OLTP, not OLAP.

**Symptoms:**
- Slow analytical queries
- High cost (provisioned IOPS)
- Production impact (resource contention)

**Solution:** Use CDC to replicate to TiDB or Iceberg.

### Anti-Pattern 4: Using TiDB for Large Scans

**Problem:** TiDB is row-oriented, inefficient for large scans.

**Symptoms:**
- Slow aggregations over large datasets
- High cost (storage + compute)

**Solution:** Use Iceberg (columnar) for analytical scans.

---

## Summary

**Key Takeaways:**

1. **Classify before choosing**: Use the six dimensions to understand your workload.
2. **Map to archetypes**: Most workloads fit into 7 common archetypes.
3. **Validate boundaries**: Check if the system supports your requirements.
4. **Avoid anti-patterns**: Don't force systems into roles they weren't designed for.

**Next Steps:**

1. Fill out the workload classification questionnaire
2. Identify the archetype
3. Review system boundaries (see [system-boundaries.md](system-boundaries.md))
4. Validate your choice with the decision framework (see [decision-framework.md](decision-framework.md))

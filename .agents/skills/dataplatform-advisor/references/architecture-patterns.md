# Architecture Patterns and Anti-Patterns

Common architectural patterns for data platform design, with real-world examples and anti-patterns to avoid.

## Overview

This document describes proven architecture patterns for common data platform use cases, along with anti-patterns that lead to poor performance, high costs, or operational issues.

---

## Pattern 1: Lambda Architecture (Batch + Streaming)

### Description

Dual-path architecture with both batch and streaming pipelines serving different tiers.

### Architecture

```
Source (Aurora)
    ↓
    ├─→ CDC → Kafka ─────────→ Stream Processing → OLAP/TiDB (real-time)
    │                                   ↓
    │                              Low latency
    │                              (seconds-minutes)
    │
    └─→ Batch Export → S3 → Spark → Iceberg (historical)
                                        ↓
                                   High latency
                                   (hours-days)
```

### Use Cases

- **Real-time tier**: User-facing dashboards, operational queries
- **Batch tier**: Historical analytics, ML, compliance reporting

### Example: Payment Analytics

```
Real-time Path:
  Aurora → CDC → Kafka → Flink → Pinot
  - Freshness: ~5 minutes
  - Retention: 90 days
  - Use case: Merchant dashboards

Batch Path:
  Aurora → Daily Export → S3 → Spark → Iceberg
  - Freshness: T-1 (daily)
  - Retention: Unlimited
  - Use case: Historical analysis, ML features
```

### Implementation

**Real-time Pipeline:**
```python
# Flink job: Kafka → Pinot
stream = env.addSource(KafkaSource("cdc.payments"))
aggregated = stream \
    .keyBy(lambda x: x.merchant_id) \
    .window(TumblingEventTimeWindows.of(Time.minutes(5))) \
    .aggregate(SumAggregator())
aggregated.addSink(PinotSink())
```

**Batch Pipeline:**
```python
# Spark job: S3 → Iceberg (daily)
df = spark.read.parquet("s3://exports/payments/2024-01-01/")
df.writeTo("warehouse.payments.transactions").append()
```

### Pros

- ✓ Best of both worlds (real-time + batch)
- ✓ Real-time tier optimized for low latency
- ✓ Batch tier optimized for cost (S3)
- ✓ Isolation (failures in one tier don't affect the other)

### Cons

- ✗ Dual code paths (maintain two pipelines)
- ✗ Data consistency (real-time vs batch may diverge)
- ✗ Operational overhead (manage two tiers)

### When to Use

- Need both real-time dashboards AND historical analytics
- Different SLAs for different use cases
- Budget allows dual infrastructure

### Anti-Pattern: Single Path Forced to Serve Both

**Bad**:
```
Aurora → CDC → Kafka → Spark → Iceberg
                               ↓
                          Trino (queries)
  Analyst: "Why is my dashboard slow?"
  Engineer: "Use the real-time tier (OLAP)"
```

**Why it fails**: Iceberg optimized for batch, not serving dashboards.

**Fix**: Add real-time tier (OLAP) for dashboards.

---

## Pattern 2: Kappa Architecture (Streaming-Only)

### Description

Single streaming pipeline serves all use cases (eliminates batch layer).

### Architecture

```
Source (Aurora)
    ↓
CDC → Kafka
    ↓
Stream Processing (Flink)
    ↓
    ├─→ OLAP (real-time serving)
    └─→ Iceberg (storage, queryable via Trino)
```

### Use Cases

- Simplify architecture (single code path)
- Real-time requirements across all use cases
- Avoid dual maintenance

### Example: Real-time Analytics Platform

```
Aurora → CDC → Kafka → Flink
                         ↓
                         ├─→ Pinot (dashboards)
                         └─→ Iceberg (analytics)

All data flows through streaming pipeline
Iceberg serves as both storage and historical analytics
```

### Implementation

```python
# Single Flink job writes to multiple sinks
stream = env.addSource(KafkaSource("cdc.payments"))

# Real-time aggregation → Pinot
stream.keyBy(...).window(...).aggregate(...).addSink(PinotSink())

# Raw events → Iceberg
stream.addSink(IcebergSink("warehouse.payments.transactions"))
```

### Pros

- ✓ Single code path (less maintenance)
- ✓ Consistent semantics (same processing for all tiers)
- ✓ Real-time across all use cases
- ✓ Simpler architecture

### Cons

- ✗ Streaming overhead for batch workloads
- ✗ Backfills harder (need to replay Kafka)
- ✗ All data must flow through stream (no direct batch loads)

### When to Use

- Real-time requirements dominate
- Team has strong streaming expertise
- Want to avoid dual maintenance

### Anti-Pattern: Kappa for Batch-Heavy Workloads

**Bad**:
```
S3 Historical Data (1TB) → Kafka → Flink → Iceberg
  "We need to backfill 3 years of data"
  Problem: Kafka retention, replay overhead
```

**Why it fails**: Batch workloads better handled by Spark (higher throughput).

**Fix**: Use Lambda (batch path for backfills, streaming for real-time).

---

## Pattern 3: Data Lakehouse (Unified Analytics)

### Description

Single storage layer (Iceberg) with multiple compute engines for different workloads.

### Architecture

```
Iceberg Lakehouse (S3 + Metadata)
    ↑
    ├─ Ingestion (Spark, Flink)
    ↓
    ├─→ Trino (interactive queries)
    ├─→ Spark (batch processing, ML)
    └─→ Presto/Athena (ad-hoc)
```

### Use Cases

- Unified storage for all analytical data
- Multiple personas (analysts, engineers, data scientists)
- Cost-efficient (S3 storage)

### Example: Centralized Data Platform

```
Data Ingestion:
  - Batch: Spark → Iceberg (daily)
  - Streaming: Flink → Iceberg (micro-batch)

Data Consumption:
  - Analysts: Trino + Metabase
  - Engineers: Spark jobs
  - Data Scientists: Spark + Jupyter
```

### Implementation

**Ingestion (Batch):**
```python
# Spark writes to Iceberg
df.writeTo("warehouse.payments.transactions") \
  .using("iceberg") \
  .partitionedBy("created_date") \
  .append()
```

**Consumption (Interactive):**
```sql
-- Trino queries Iceberg
SELECT merchant_id, SUM(amount)
FROM iceberg.warehouse.payments.transactions
WHERE created_date >= DATE '2024-01-01'
GROUP BY merchant_id;
```

**Consumption (Batch):**
```python
# Spark reads from Iceberg
df = spark.read.table("warehouse.payments.transactions")
features = df.groupBy("merchant_id").agg(...)
```

### Pros

- ✓ Single source of truth (no data duplication)
- ✓ Low storage cost (S3 vs SSD)
- ✓ Schema evolution (Iceberg ACID)
- ✓ Multi-engine support

### Cons

- ✗ Not for low-latency serving (seconds, not milliseconds)
- ✗ Not for high-concurrency (< 50 queries typically)
- ✗ Requires good data governance (schema management)

### When to Use

- Centralized analytics platform
- Multiple compute engines needed
- Cost-sensitive (large data volumes)

### Anti-Pattern: Lakehouse for Real-time Serving

**Bad**:
```
User Dashboard API → Trino → Iceberg
  p99 latency: 5-10 seconds
  User: "Why is my dashboard so slow?"
```

**Why it fails**: Iceberg not designed for sub-second serving.

**Fix**: Use OLAP (Pinot) for dashboards, Iceberg for historical analytics.

---

## Pattern 4: OLAP-First (Pre-Aggregation)

### Description

Pre-aggregate and denormalize data for OLAP systems, optimized for user-facing dashboards.

### Architecture

```
Source (Aurora)
    ↓
CDC → Kafka
    ↓
Spark (pre-join, denormalize, aggregate)
    ↓
OLAP (Pinot/ClickHouse)
    ↓
Dashboard API (sub-second queries)
```

### Use Cases

- User-facing dashboards (< 100ms latency)
- High concurrency (1000s QPS)
- Fixed query patterns

### Example: Merchant Dashboard

```
Pipeline:
  Aurora (transactions, merchants, products)
    ↓
  CDC → Kafka
    ↓
  Spark Structured Streaming:
    - Join transactions + merchants + products
    - Denormalize (single wide table)
    - Pre-aggregate by merchant + day
    ↓
  Pinot (star-tree indexes)
    ↓
  Dashboard (< 100ms queries)
```

### Implementation

**Pre-aggregation Pipeline:**
```python
# Spark: Join + Denormalize
transactions = spark.readStream.format("kafka").load()
merchants = spark.read.table("warehouse.merchants")
products = spark.read.table("warehouse.products")

enriched = transactions \
    .join(merchants, "merchant_id", "left") \
    .join(products, "product_id", "left") \
    .select(
        "merchant_id",
        "merchant_name",
        "product_category",
        "amount",
        "created_at"
    )

# Write to Pinot
enriched.writeStream \
    .format("pinot") \
    .option("table", "merchant_analytics") \
    .start()
```

**Query (Dashboard):**
```sql
-- Pinot: Pre-joined, fast query
SELECT
    DATE(created_at),
    SUM(amount),
    COUNT(*)
FROM merchant_analytics
WHERE merchant_id = 12345
  AND created_at >= NOW() - INTERVAL '30' DAY
GROUP BY DATE(created_at);

-- Latency: 50ms (star-tree index)
```

### Pros

- ✓ Sub-second latency (pre-aggregated)
- ✓ High concurrency (OLAP optimized)
- ✓ Consistent performance (indexed)
- ✓ Simple queries (no joins at query time)

### Cons

- ✗ Data duplication (raw + pre-aggregated)
- ✗ Pipeline complexity (pre-aggregation logic)
- ✗ Less flexible (pre-defined aggregations)
- ✗ Higher storage cost (denormalized)

### When to Use

- User-facing dashboards with strict latency SLAs
- Fixed query patterns (known in advance)
- High concurrency requirements

### Anti-Pattern: Query-Time Joins in OLAP

**Bad**:
```sql
-- Pinot doesn't support joins
SELECT t.amount, m.name
FROM transactions t
JOIN merchants m ON t.merchant_id = m.id;
-- Error: Joins not supported
```

**Why it fails**: OLAP systems designed for single-table queries.

**Fix**: Pre-join in Spark, denormalize before ingesting to OLAP.

---

## Pattern 5: Hot-Warm-Cold Tiering

### Description

Multi-tier architecture with data aging from hot (expensive, fast) to cold (cheap, slow).

### Architecture

```
Hot Tier (Aurora)
  - Freshness: Real-time
  - Retention: Days-weeks
  - Latency: Milliseconds
  - Cost: High
    ↓ (CDC after 24 hours)
Warm Tier (TiDB)
  - Freshness: Minutes
  - Retention: Months
  - Latency: Milliseconds-seconds
  - Cost: Medium
    ↓ (Archive after 90 days)
Cold Tier (Iceberg)
  - Freshness: Hours-days
  - Retention: Years
  - Latency: Seconds-minutes
  - Cost: Low
```

### Use Cases

- Cost optimization (age out old data)
- Different SLAs by data age
- Compliance (long-term retention)

### Example: Transaction Lifecycle

```
Recent (< 24 hours):
  Aurora (application DB)
  - Use case: Live transactions, real-time updates
  - Latency: < 10ms

Recent (< 90 days):
  TiDB (operational analytics)
  - Use case: Customer support, debugging
  - Latency: 10-100ms

Historical (90 days+):
  Iceberg (data lake)
  - Use case: Historical analysis, ML, compliance
  - Latency: Seconds-minutes
```

### Implementation

**Hot → Warm:**
```python
# CDC to TiDB (real-time)
Aurora → Debezium → Kafka → Flink → TiDB
```

**Warm → Cold:**
```python
# Archive to Iceberg (daily)
spark.read.table("tidb.transactions") \
    .filter(col("created_at") < current_date() - 90) \
    .writeTo("warehouse.transactions_archive") \
    .append()

# Delete from TiDB
spark.sql("DELETE FROM tidb.transactions WHERE created_at < CURRENT_DATE - 90")
```

**Unified Query (Federation):**
```sql
-- Trino: Query across tiers
SELECT * FROM (
  SELECT * FROM tidb.transactions WHERE created_date >= CURRENT_DATE - 90
  UNION ALL
  SELECT * FROM iceberg.warehouse.transactions_archive WHERE created_date < CURRENT_DATE - 90
)
WHERE merchant_id = 12345;
```

### Pros

- ✓ Cost optimization (cheaper storage for old data)
- ✓ Performance optimization (hot tier fast for recent queries)
- ✓ Compliance (long-term retention in cheap storage)
- ✓ Scalability (cold tier scales to petabytes)

### Cons

- ✗ Complexity (manage data aging, archival)
- ✗ Query routing (users must know which tier)
- ✗ Data consistency (eventual consistency across tiers)

### When to Use

- Large data volumes (TBs-PBs)
- Queries skewed toward recent data
- Cost-sensitive workloads

### Anti-Pattern: Single Tier for All Data

**Bad**:
```
Store 5 years of data in TiDB
  - Storage cost: 5TB × $0.08/GB/mo = $410/month
  - Query all 5 years: Slow, expensive
```

**Why it fails**: Paying high cost for cold data (rarely queried).

**Fix**: Archive to Iceberg after 90 days (save ~70% storage cost).

---

## Pattern 6: Event Sourcing + CQRS

### Description

Store raw events (event sourcing) and build read-optimized views (CQRS).

### Architecture

```
Commands (Writes)
    ↓
Event Store (Kafka, Iceberg)
    ↓
Event Processors (Flink, Spark)
    ↓
Read Models (OLAP, TiDB, Iceberg)
    ↓
Queries (Reads)
```

### Use Cases

- Audit trails (compliance, debugging)
- Rebuild state from events
- Multiple read models for different use cases

### Example: Payment Events

```
Event Store (Kafka + Iceberg):
  - payment.created
  - payment.authorized
  - payment.captured
  - payment.refunded

Read Models:
  1. OLAP (Merchant Dashboard):
     - Pre-aggregated: SUM(amount) by merchant
  2. TiDB (Customer Support):
     - Latest payment state by transaction_id
  3. Iceberg (Analytics):
     - Full event history for analysis
```

### Implementation

**Event Store:**
```python
# Write events to Kafka
producer.send("payment.events", event)

# Persist to Iceberg (audit trail)
spark.readStream \
    .format("kafka") \
    .load() \
    .writeStream \
    .format("iceberg") \
    .option("path", "warehouse.events.payments") \
    .start()
```

**Read Model 1 (OLAP):**
```python
# Flink: Aggregate events → Pinot
events.keyBy(e => e.merchant_id) \
    .window(TumblingEventTimeWindows.of(Time.days(1))) \
    .aggregate(new SumAggregator()) \
    .addSink(PinotSink())
```

**Read Model 2 (TiDB):**
```python
# Flink: Materialize latest state → TiDB
events.keyBy(e => e.transaction_id) \
    .process(new StateMaterializationFunction()) \
    .addSink(JdbcSink("tidb.payment_states"))
```

### Pros

- ✓ Full audit trail (replay events)
- ✓ Multiple read models (optimize for each use case)
- ✓ Flexibility (rebuild views from events)
- ✓ Decoupling (event store separate from read models)

### Cons

- ✗ Complexity (event schema design, versioning)
- ✗ Eventual consistency (read models lag behind events)
- ✗ Storage overhead (store both events and read models)

### When to Use

- Audit requirements (compliance, forensics)
- Multiple read patterns for same data
- Need to rebuild state from history

### Anti-Pattern: Direct State Mutations

**Bad**:
```sql
-- Update payment status directly
UPDATE payments SET status = 'CAPTURED' WHERE id = 123;

-- Problem: Lost history (can't replay events)
```

**Why it fails**: Lost audit trail, can't reconstruct history.

**Fix**: Store events, materialize state from events.

---

## Pattern 7: Self-Service Analytics Platform

### Description

Platform that allows analysts and data scientists to explore data without engineering support.

### Architecture

```
Data Lake (Iceberg)
    ↓
Query Engines:
  ├─ Trino (interactive SQL)
  ├─ Spark (notebooks, jobs)
  └─ Jupyter (Python, R)
    ↓
Governance:
  ├─ Apache Ranger (access control)
  ├─ Data Catalog (metadata)
  └─ Data Quality (validation)
```

### Use Cases

- Empower analysts (reduce dependency on engineers)
- Centralized data access
- Governed self-service

### Example: Analytics Platform

```
Data Sources:
  - Iceberg lakehouse (all analytical data)
  - TiDB (operational data)
  - External APIs (via connectors)

Access Methods:
  - Metabase (BI tool) → Trino
  - Jupyter Notebooks → Spark
  - Airflow (scheduled jobs) → Spark

Governance:
  - Ranger: Column-level access control
  - Data Catalog: Searchable metadata
  - Data Quality: Automated validation
```

### Implementation

**Data Catalog:**
```yaml
dataset:
  name: "warehouse.payments.transactions"
  description: "Payment transactions (all history)"
  owner: "payments-team@company.com"
  schema:
    - name: "transaction_id"
      type: "BIGINT"
      description: "Unique transaction identifier"
      pii: false
    - name: "merchant_id"
      type: "BIGINT"
      pii: false
    - name: "amount"
      type: "DECIMAL(18,2)"
      pii: false
  retention: "Unlimited"
  freshness: "T-1 (daily batch)"
```

**Access Control (Ranger):**
```yaml
policy:
  name: "mask-pii-columns"
  resources:
    database: "warehouse"
    table: "customers"
    column: "email"
  access:
    - group: "analysts"
      permissions: ["SELECT"]
      mask: "MASK_HASH"  # Show hashed email
    - group: "security-team"
      permissions: ["SELECT"]
      mask: "NONE"  # Show full email
```

### Pros

- ✓ Self-service (analysts don't need engineers)
- ✓ Governed (access control, data quality)
- ✓ Centralized (single source of truth)
- ✓ Scalable (multiple users, datasets)

### Cons

- ✗ Requires mature data culture
- ✗ Governance overhead (policies, metadata)
- ✗ Cost management (unoptimized queries)

### When to Use

- Large analytics team (50+ analysts)
- Mature data culture
- Need to scale analytics without scaling engineering

### Anti-Pattern: Ungovened Self-Service

**Bad**:
```
Analysts: "We have access to production DB!"
  - Run expensive queries on Aurora (kill production)
  - No access control (see PII)
  - No cost controls (run $10K queries)
```

**Why it fails**: Impacts production, compliance risks, cost overruns.

**Fix**: Replicate to dedicated analytics tier with governance (Ranger, quotas).

---

## Anti-Pattern Catalog

### Anti-Pattern 1: The Monolith

**Description**: Single system for all workloads.

**Example**:
```
Aurora for everything:
  - Transactional writes ✓
  - User-facing dashboards ✗
  - Historical analytics ✗
  - ML feature engineering ✗
```

**Why it fails**: No system optimized for all workloads.

**Fix**: Multi-tier architecture (hot/warm/cold, OLAP for dashboards).

### Anti-Pattern 2: Over-Engineering

**Description**: Building complex architecture for simple use case.

**Example**:
```
Use case: 10 analysts querying 100GB data
Architecture: Kafka + Flink + Pinot + Iceberg + Trino + Spark
Problem: Operational overhead >> value
```

**Why it fails**: Complexity outweighs benefits.

**Fix**: Start simple (Trino + Iceberg), add complexity as needed.

### Anti-Pattern 3: Technology-Driven

**Description**: Choosing technology before understanding requirements.

**Example**:
```
Engineer: "Let's use Flink because it's cool!"
Use case: Daily batch ETL
Problem: Flink overkill for batch
```

**Why it fails**: Wrong tool for the job.

**Fix**: Classify workload first, then choose system.

### Anti-Pattern 4: Data Duplication Without Reason

**Description**: Copying data to multiple systems without clear use case.

**Example**:
```
Data in 5 systems:
  - Aurora (source)
  - TiDB (why?)
  - Pinot (why?)
  - Iceberg (why?)
  - Redshift (why?)

Cost: 5x storage, 5x maintenance
```

**Why it fails**: High cost, no clear benefit.

**Fix**: Define use case for each tier, eliminate duplicates.

### Anti-Pattern 5: Ignoring Data Freshness

**Description**: Using batch tier for real-time use cases.

**Example**:
```
Use case: Real-time fraud detection
Architecture: Daily batch → Iceberg
Problem: T-1 freshness (fraud succeeds)
```

**Why it fails**: Freshness requirement not met.

**Fix**: Use real-time tier (Flink, TiDB, OLAP).

---

## Summary

**Key Patterns:**
1. **Lambda**: Dual path (batch + streaming)
2. **Kappa**: Single streaming path
3. **Lakehouse**: Unified storage, multiple compute
4. **OLAP-First**: Pre-aggregate for dashboards
5. **Hot-Warm-Cold**: Data tiering by age
6. **Event Sourcing**: Store events, build read models
7. **Self-Service**: Governed analytics platform

**Choosing the Right Pattern:**
- Consider workload requirements (latency, cost, flexibility)
- Evaluate trade-offs (complexity vs performance vs cost)
- Start simple, add complexity as needed
- Avoid anti-patterns (monolith, over-engineering, duplication)

**Next Steps:**
1. Identify your use cases
2. Map to patterns (may combine multiple)
3. Validate with [System Boundaries](system-boundaries.md)
4. Design with [Decision Framework](decision-framework.md)

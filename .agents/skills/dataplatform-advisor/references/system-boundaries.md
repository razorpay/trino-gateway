# System Boundaries and Anti-Patterns

This document defines what each system should and should NOT be used for, helping you avoid common anti-patterns.

## Overview

Every data system has a "design envelope" - a set of workloads it's optimized for. Using a system outside its design envelope leads to poor performance, high costs, or operational issues.

---

## Aurora (MySQL)

### Designed For ✓

**Transactional workloads (OLTP):**
- Point queries (SELECT by primary key)
- Small range scans (indexed queries)
- Transactional updates and inserts
- ACID-compliant operations
- Read-after-write consistency

**Characteristics:**
- Sub-10ms latency for indexed queries
- 100s-1000s TPS
- Concurrent transactions

**Examples:**
```sql
-- ✓ Point query
SELECT * FROM users WHERE id = 12345;

-- ✓ Indexed lookup
SELECT * FROM orders WHERE user_id = 12345 ORDER BY created_at DESC LIMIT 10;

-- ✓ Transaction
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE id = 123;
INSERT INTO transactions (account_id, amount) VALUES (123, -100);
COMMIT;
```

### NOT Designed For ✗

**Analytical workloads (OLAP):**
- Large table scans
- Complex aggregations
- Multi-table joins (fact + dimensions)
- Historical reporting

**Why it fails:**
- Row-oriented storage (reads unnecessary columns)
- Impacts production (resource contention)
- Expensive (provisioned IOPS for scans)

**Anti-pattern examples:**
```sql
-- ✗ Full table scan
SELECT merchant_id, SUM(amount)
FROM transactions  -- 100M rows
WHERE created_at >= '2024-01-01'
GROUP BY merchant_id;
-- Problem: Scans entire table, kills production performance

-- ✗ Large join
SELECT t.*, m.*, p.*
FROM transactions t
JOIN merchants m ON t.merchant_id = m.id
JOIN products p ON t.product_id = p.id
WHERE t.created_at >= '2024-01-01';
-- Problem: Joins large tables, high memory usage
```

**Solution**: Use CDC to replicate to TiDB or Iceberg for analytics.

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Table scan size | < 100K rows | Slow queries, high IOPS cost |
| Query complexity | Simple queries | Query timeouts, CPU spikes |
| Concurrent queries | < 100 | Connection pool exhaustion |
| Retention | Weeks-months | High storage cost |

---

## TiDB

### Designed For ✓

**Operational analytics (HTAP):**
- Near real-time queries (minutes latency acceptable)
- Medium-scale aggregations (millions of rows)
- Point queries with freshness requirements
- Transactional access with analytical queries

**Characteristics:**
- 10-100ms latency for indexed queries
- High concurrency (100s connections)
- Horizontal scalability
- MySQL compatibility

**Examples:**
```sql
-- ✓ Recent data aggregation
SELECT merchant_id, SUM(amount)
FROM transactions
WHERE created_at >= NOW() - INTERVAL 1 DAY
GROUP BY merchant_id;

-- ✓ Point query with joins
SELECT t.*, m.name
FROM transactions t
JOIN merchants m ON t.merchant_id = m.id
WHERE t.id = 12345;

-- ✓ Operational reporting
SELECT DATE(created_at), status, COUNT(*)
FROM transactions
WHERE merchant_id = 12345
  AND created_at >= NOW() - INTERVAL 30 DAY
GROUP BY DATE(created_at), status;
```

### NOT Designed For ✗

**Large-scale analytics:**
- Full table scans (100s of millions of rows)
- Multi-year historical analysis
- ML feature engineering (large-scale joins)

**Why it fails:**
- Row-oriented storage (inefficient for column scans)
- High cost (SSD storage vs S3)
- Limited retention (cost prohibitive)

**Anti-pattern examples:**
```sql
-- ✗ Large historical scan
SELECT merchant_id, AVG(amount)
FROM transactions  -- 1B rows
WHERE created_at >= '2020-01-01'
GROUP BY merchant_id;
-- Problem: Row-oriented storage is slow, expensive for large scans

-- ✗ Complex multi-table join
SELECT t.*, m.*, p.*, c.*
FROM transactions t
JOIN merchants m ON t.merchant_id = m.id
JOIN products p ON t.product_id = p.id
JOIN categories c ON p.category_id = c.id
WHERE t.created_at >= '2020-01-01';
-- Problem: Large distributed joins, high memory usage
```

**Solution**: Use Iceberg + Trino/Spark for large-scale analytics.

### NOT Designed For ✗

**User-facing dashboards (high QPS):**
- 1000+ QPS
- Sub-100ms latency requirements
- Fixed query patterns

**Why it fails:**
- Row-oriented storage (slower than columnar OLAP)
- Higher cost per QPS than OLAP
- No pre-aggregation (query-time aggregation)

**Anti-pattern example:**
```sql
-- ✗ Dashboard query (1000 QPS)
SELECT DATE(created_at), SUM(amount)
FROM transactions
WHERE merchant_id = ?
  AND created_at >= ?
GROUP BY DATE(created_at);
-- Problem: Query-time aggregation at scale, expensive
```

**Solution**: Pre-aggregate to OLAP (Pinot) for user-facing dashboards.

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Scan size | < 10M rows | Slow queries, high cost |
| Retention | < 1 year | High storage cost ($0.08/GB/mo) |
| QPS | < 1000 | Higher cost than OLAP |
| Joins | 2-3 tables | Distributed join overhead |

---

## OLAP (Pinot / ClickHouse)

### Designed For ✓

**User-facing dashboards:**
- Sub-second latency
- High concurrency (100s-1000s QPS)
- Fixed query patterns
- Pre-defined aggregations

**Characteristics:**
- 10-100ms latency (p99)
- Very high concurrency
- Star-tree indexes for fast aggregations
- Real-time ingestion

**Examples:**
```sql
-- ✓ Dashboard query (fixed pattern)
SELECT DATE(created_at), SUM(amount), COUNT(*)
FROM payments
WHERE merchant_id = ?
  AND created_at >= ?
  AND created_at < ?
GROUP BY DATE(created_at);

-- ✓ Pre-defined KPI
SELECT payment_method, COUNT(*), SUM(amount)
FROM payments
WHERE merchant_id = ?
  AND created_at >= NOW() - INTERVAL 30 DAYS
GROUP BY payment_method;

-- ✓ Real-time metrics
SELECT COUNT(*), SUM(amount)
FROM payments
WHERE created_at >= NOW() - INTERVAL 5 MINUTES;
```

### NOT Designed For ✗

**Ad-hoc exploration:**
- Unpredictable queries
- Complex joins
- Schema changes

**Why it fails:**
- Fixed schema (must pre-define indexes)
- No joins (single-table queries only)
- Requires schema evolution for new queries

**Anti-pattern examples:**
```sql
-- ✗ Ad-hoc exploration
SELECT merchant_id, product_id, AVG(amount)
FROM payments
WHERE amount > 1000
  AND status = 'REFUNDED'
  AND payment_method IN ('card', 'upi')
GROUP BY merchant_id, product_id
HAVING AVG(amount) > 5000;
-- Problem: Unindexed filters, complex predicates

-- ✗ Join
SELECT p.*, m.name
FROM payments p
JOIN merchants m ON p.merchant_id = m.id;
-- Problem: Pinot doesn't support joins
```

**Solution**: Use Trino + Iceberg for ad-hoc exploration.

### NOT Designed For ✗

**Complex transformations:**
- ETL logic
- Data quality checks
- Multi-source joins

**Why it fails:**
- Query-only (no writes except ingestion)
- Single-table limitation
- Not a compute engine

**Anti-pattern example:**
```sql
-- ✗ Transformation
SELECT merchant_id,
       amount * currency_rate AS amount_usd,
       CASE
         WHEN amount > 10000 THEN 'high'
         WHEN amount > 1000 THEN 'medium'
         ELSE 'low'
       END AS amount_category
FROM payments;
-- Problem: Better done in Spark during ingestion
```

**Solution**: Pre-compute transformations in Spark, ingest to Pinot.

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Query patterns | Fixed (pre-indexed) | Slow queries, missing indexes |
| Joins | None (denormalize) | Not supported |
| Schema changes | Infrequent | Reindex overhead |
| Retention | Months (configurable) | High storage cost for long retention |

---

## Iceberg (Lakehouse)

### Designed For ✓

**Historical analytics:**
- Large-scale scans (100s GB-TBs)
- Multi-year retention
- Complex queries (joins, subqueries)
- ML feature engineering

**Characteristics:**
- Low storage cost ($0.023/GB/mo)
- Unlimited retention
- ACID transactions
- Schema evolution

**Examples:**
```sql
-- ✓ Historical analysis
SELECT merchant_id, SUM(amount), AVG(amount)
FROM warehouse.transactions
WHERE created_date >= '2020-01-01'
GROUP BY merchant_id;

-- ✓ Complex join
SELECT t.*, m.name, p.category
FROM warehouse.transactions t
JOIN warehouse.merchants m ON t.merchant_id = m.id
JOIN warehouse.products p ON t.product_id = p.id
WHERE t.created_date >= '2023-01-01';

-- ✓ ML feature engineering
SELECT merchant_id,
       COUNT(*) as txn_count,
       AVG(amount) as avg_amount,
       STDDEV(amount) as stddev_amount
FROM warehouse.transactions
WHERE created_date >= '2023-01-01'
GROUP BY merchant_id;
```

### NOT Designed For ✗

**Low-latency queries:**
- Sub-second latency
- User-facing APIs
- Real-time dashboards

**Why it fails:**
- S3 latency (minimum seconds)
- No indexes (full scan with predicate pushdown)
- Query planning overhead

**Anti-pattern examples:**
```sql
-- ✗ User-facing API
SELECT * FROM warehouse.transactions
WHERE id = 12345;
-- Problem: Scans Parquet files, seconds latency (vs ms for OLAP/TiDB)

-- ✗ Dashboard query (high QPS)
SELECT DATE(created_date), SUM(amount)
FROM warehouse.transactions
WHERE merchant_id = ?
  AND created_date >= ?
GROUP BY DATE(created_date);
-- Problem: Too slow for user-facing (seconds), can't handle high QPS
```

**Solution**: Pre-aggregate to OLAP or TiDB for low-latency serving.

### NOT Designed For ✗

**Real-time ingestion:**
- Millisecond-latency writes
- Small micro-batches (< 1 minute)

**Why it fails:**
- File-based (not record-based)
- Small file problem (too many small files)
- S3 eventual consistency

**Anti-pattern example:**
```python
# ✗ Real-time ingestion (per-record)
for record in stream:
    df = spark.createDataFrame([record])
    df.writeTo("warehouse.transactions").append()
# Problem: Creates 1 file per record (millions of small files)
```

**Solution**: Batch writes (5-15 minute micro-batches), use Flink/Spark Streaming.

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Latency | Seconds-minutes | Not applicable for real-time |
| File size | > 100MB per file | Small file problem (slow queries) |
| Concurrency | < 50 queries | S3 throttling, query queuing |
| Write frequency | > 1 minute batches | Too many small files |

---

## Trino

### Designed For ✓

**Ad-hoc exploration:**
- Analyst queries (variable patterns)
- BI tools (Metabase, Looker)
- Debugging (one-off queries)
- Cross-system queries (federation)

**Characteristics:**
- Seconds-minutes latency
- Handles any SQL query
- No pre-built indexes
- Low-medium concurrency

**Examples:**
```sql
-- ✓ Ad-hoc exploration
SELECT merchant_id, payment_method, COUNT(*)
FROM iceberg.warehouse.transactions
WHERE created_date >= DATE '2024-01-01'
  AND amount > 1000
GROUP BY merchant_id, payment_method
ORDER BY COUNT(*) DESC
LIMIT 100;

-- ✓ Complex query
WITH merchant_stats AS (
  SELECT merchant_id, AVG(amount) as avg_amount
  FROM iceberg.warehouse.transactions
  WHERE created_date >= DATE '2024-01-01'
  GROUP BY merchant_id
)
SELECT m.name, s.avg_amount
FROM merchant_stats s
JOIN mysql.prod.merchants m ON s.merchant_id = m.id
WHERE s.avg_amount > 1000;

-- ✓ Cross-system join
SELECT i.transaction_id, p.payment_count
FROM iceberg.warehouse.transactions i
LEFT JOIN pinot.default.merchant_stats p ON i.merchant_id = p.merchant_id
WHERE i.created_date = CURRENT_DATE;
```

### NOT Designed For ✗

**User-facing APIs:**
- Sub-second latency
- High concurrency (1000s QPS)
- SLA requirements

**Why it fails:**
- Variable latency (depends on query complexity)
- Shared cluster (resource contention)
- No latency guarantees

**Anti-pattern examples:**
```sql
-- ✗ API endpoint
GET /merchants/{id}/stats
→ SELECT SUM(amount) FROM iceberg.warehouse.transactions
  WHERE merchant_id = ?
-- Problem: Unpredictable latency (1-30 seconds), shared resources

-- ✗ Dashboard (100s QPS)
SELECT DATE(created_date), SUM(amount)
FROM iceberg.warehouse.transactions
WHERE merchant_id = ?
GROUP BY DATE(created_date);
-- Problem: Shared cluster, queuing, variable latency
```

**Solution**: Use OLAP (Pinot) or TiDB for user-facing APIs.

### NOT Designed For ✗

**Heavy transformations:**
- Large ETL jobs
- Backfills (multi-hour jobs)
- ML training

**Why it fails:**
- In-memory execution (OOM on large aggregations)
- No fault tolerance (query fails on node failure)
- Shared cluster (impacts other users)

**Anti-pattern example:**
```sql
-- ✗ Large ETL
INSERT INTO iceberg.warehouse.merchant_features
SELECT merchant_id,
       COUNT(*) as txn_count,
       AVG(amount) as avg_amount,
       -- 50 more features
FROM iceberg.warehouse.transactions
WHERE created_date >= '2020-01-01'  -- 3 years of data
GROUP BY merchant_id;
-- Problem: Large aggregation, OOM, impacts shared cluster
```

**Solution**: Use Spark for heavy transformations (fault-tolerant, isolated).

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Latency | Seconds-minutes | Not suitable for APIs |
| Concurrency | 10-50 queries | Queuing, resource contention |
| Memory | Cluster memory | OOM, query failures |
| Long-running | < 30 minutes | Impacts other users |

---

## Spark

### Designed For ✓

**Batch processing:**
- ETL pipelines (hours-long jobs)
- Backfills (historical data processing)
- ML training (feature engineering, model training)
- Data quality checks

**Characteristics:**
- High throughput (100s GB/sec)
- Fault-tolerant (lineage-based recovery)
- Isolated (ephemeral clusters)
- Cost-efficient (pay-per-job)

**Examples:**
```python
# ✓ ETL pipeline
df = spark.read.table("warehouse.raw.transactions")

clean = df.filter(col("amount") > 0) \
    .withColumn("created_date", to_date(col("created_at"))) \
    .dropDuplicates(["transaction_id"])

enriched = clean.join(merchant_dim, "merchant_id", "left")

enriched.writeTo("warehouse.analytics.transactions_enriched") \
    .using("iceberg") \
    .createOrReplace()

# ✓ ML feature engineering
features = spark.read.table("warehouse.transactions") \
    .groupBy("merchant_id") \
    .agg(
        count("*").alias("txn_count"),
        avg("amount").alias("avg_amount"),
        stddev("amount").alias("stddev_amount")
    )

features.write.mode("overwrite").parquet("s3://ml/features/")

# ✓ Backfill
for date in date_range('2020-01-01', '2024-01-01'):
    df = spark.read.parquet(f"s3://raw/transactions/{date}/")
    df.writeTo("warehouse.transactions").append()
```

### NOT Designed For ✗

**Interactive queries:**
- Ad-hoc exploration (seconds latency)
- User queries (immediate response)

**Why it fails:**
- High startup overhead (10-60 seconds)
- Optimized for throughput, not latency
- Batch-oriented

**Anti-pattern example:**
```python
# ✗ Interactive query
result = spark.sql("""
  SELECT merchant_id, SUM(amount)
  FROM warehouse.transactions
  WHERE created_date = CURRENT_DATE
  GROUP BY merchant_id
""").toPandas()
# Problem: 30-60 sec startup overhead for 5-sec query
```

**Solution**: Use Trino for interactive queries.

### NOT Designed For ✗

**Real-time processing:**
- Millisecond latency
- Continuous event processing

**Why it fails:**
- Micro-batch (seconds latency)
- Not true streaming

**Anti-pattern example:**
```python
# ✗ Real-time alerting
stream = spark.readStream.format("kafka").load()
alerts = stream.filter(col("amount") > 10000)
alerts.writeStream.foreach(send_alert).start()
# Problem: Micro-batch delay (seconds), use Flink for real-time
```

**Solution**: Use Flink for true real-time processing.

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Latency | Minutes-hours | Not suitable for interactive |
| Startup overhead | 10-60 seconds | Slow for quick queries |
| State management | Limited | Use Flink for stateful streaming |

---

## Flink

### Designed For ✓

**Real-time stream processing:**
- CDC pipelines (Aurora → TiDB)
- Real-time ETL (transform events)
- Stateful computations (windows, joins)
- Event-driven applications

**Characteristics:**
- Millisecond-second latency
- Exactly-once semantics
- Stateful operators
- Continuous processing

**Examples:**
```java
// ✓ CDC pipeline
DataStream<Change> cdc = env.addSource(new KafkaSource("cdc.transactions"));
cdc.addSink(new JdbcSink("tidb.transactions"));

// ✓ Real-time aggregation
DataStream<Transaction> transactions = env.addSource(...);
transactions
    .keyBy(txn -> txn.merchantId)
    .window(TumblingEventTimeWindows.of(Time.minutes(5)))
    .aggregate(new SumAggregator())
    .addSink(new PinotSink());

// ✓ Stateful computation
DataStream<Alert> alerts = transactions
    .keyBy(txn -> txn.merchantId)
    .flatMap(new FraudDetectionFunction());  // Stateful
```

### NOT Designed For ✗

**Batch processing:**
- Historical data processing
- Large-scale ETL (hours-long jobs)

**Why it fails:**
- Optimized for streaming, not batch
- Higher overhead than Spark for batch

**Anti-pattern example:**
```java
// ✗ Batch ETL
DataStream<Transaction> batch = env.readFile(...);
batch.map(...).addSink(...);
// Problem: Use Spark for batch (better throughput, fault tolerance)
```

**Solution**: Use Spark for batch processing.

### NOT Designed For ✗

**Ad-hoc queries:**
- Interactive exploration
- One-off queries

**Why it fails:**
- Continuous processing (not query engine)
- Requires job deployment

**Anti-pattern example:**
```sql
-- ✗ Ad-hoc query
SELECT merchant_id, SUM(amount)
FROM kafka_transactions
WHERE created_at >= CURRENT_DATE
GROUP BY merchant_id;
-- Problem: Use Trino for queries (Flink is for continuous processing)
```

**Solution**: Use Trino or Spark SQL for queries.

### Boundaries Summary

| Boundary | Limit | What Happens When Exceeded |
|----------|-------|----------------------------|
| Batch workloads | Not designed | Use Spark instead |
| Ad-hoc queries | Not designed | Use Trino instead |
| Stateless pipelines | Simple CDC | Consider managed CDC (Debezium) |

---

## Summary Table

| System | Designed For | NOT Designed For | Common Anti-Pattern |
|--------|-------------|------------------|---------------------|
| **Aurora** | OLTP, transactions | Analytics | Using for reporting |
| **TiDB** | Operational analytics | Large scans, user-facing dashboards | Using for historical analysis |
| **OLAP** | User dashboards (fixed queries) | Ad-hoc exploration, joins | Using for analyst queries |
| **Iceberg** | Historical analytics, ML | Low-latency, real-time | Using for APIs |
| **Trino** | Ad-hoc exploration | User-facing APIs, heavy ETL | Using for dashboards |
| **Spark** | Batch ETL, ML | Interactive queries | Using for ad-hoc queries |
| **Flink** | Real-time streaming | Batch processing | Using for batch ETL |

---

## Key Principles

1. **Stay within design envelope**: Use systems for their intended purpose
2. **Don't extend beyond boundaries**: Adding caching/indexing won't fix fundamental mismatch
3. **Choose purpose-built systems**: Use the right tool for the job
4. **Replicate, don't consolidate**: Replicate data to appropriate tiers (CDC, ETL)

## When in Doubt

Ask these questions:

1. **What's the workload archetype?** (See [workload-classification.md](workload-classification.md))
2. **What are the latency/concurrency requirements?**
3. **Is the system designed for this workload?** (Check this document)
4. **What are the anti-patterns?** (See examples above)

If the system isn't designed for the workload, find the right system or replicate data to appropriate tier.

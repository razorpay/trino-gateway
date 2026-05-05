# Data Flow Patterns

This document describes the complete data flow patterns in the platform, including implementation details, examples, and best practices.

## Overview

Data flows through the platform in several distinct patterns, each optimized for different use cases. Understanding these patterns is critical for designing reliable data pipelines and choosing the right architecture.

## Pattern 1: CDC (Change Data Capture) Pattern

### Architecture

```
Source DB (Aurora/DynamoDB)
    ↓
Debezium CDC Connector
    ↓
Kafka Topic (raw changes)
    ↓
Multiple Consumers:
    ├→ TiDB (real-time replica)
    ├→ Pinot (real-time analytics)
    └→ Iceberg (historical lake)
```

### How It Works

1. **Source Database**: Application services write to Aurora (MySQL) or DynamoDB
2. **CDC Capture**: Debezium reads database transaction logs (binlog for MySQL, streams for DynamoDB)
3. **Event Publishing**: Changes published to Kafka as structured events (INSERT, UPDATE, DELETE)
4. **Event Ordering**: Kafka partitioning preserves ordering per primary key
5. **Consumer Processing**: Downstream systems consume and apply changes

### Implementation Details

**Debezium Configuration:**
```yaml
connector:
  name: payments-cdc
  connector.class: io.debezium.connector.mysql.MySqlConnector
  database.hostname: aurora-payments.rds.amazonaws.com
  database.user: debezium_user
  database.include.list: payments
  table.include.list: payments.transactions,payments.refunds
  snapshot.mode: initial
  topic.prefix: cdc.payments
```

**Kafka Topic Structure:**
- Topic naming: `cdc.<source_db>.<table_name>`
- Partitioning: By primary key hash
- Retention: 7 days (replayable)
- Format: Avro with schema registry

**Downstream Consumption:**

*TiDB Sync:*
```python
# Spark Structured Streaming job applies CDC events to TiDB
# Supports upserts and deletes via foreachBatch
# Latency: ~10 seconds end-to-end
spark.readStream \
  .format("kafka") \
  .option("kafka.bootstrap.servers", "kafka:9092") \
  .option("subscribe", "cdc.payments.transactions") \
  .load() \
  .writeStream \
  .foreachBatch(lambda df, epoch_id: write_to_tidb(df)) \
  .option("checkpointLocation", "s3://checkpoints/tidb-sink") \
  .trigger(processingTime="10 seconds") \
  .start()
```

*Iceberg Sync:*
```python
# Spark Structured Streaming
spark.readStream \
  .format("kafka") \
  .option("kafka.bootstrap.servers", "kafka:9092") \
  .option("subscribe", "cdc.payments.transactions") \
  .load() \
  .writeStream \
  .format("iceberg") \
  .option("path", "warehouse.payments.transactions") \
  .option("checkpointLocation", "s3://checkpoints/payments") \
  .trigger(processingTime="5 minutes") \
  .start()
```

### Use Cases

- **Real-time replication**: Keep TiDB in sync with production databases
- **Event sourcing**: Capture all changes for audit and replay
- **Real-time analytics**: Feed OLAP systems with fresh data
- **Data lake ingestion**: Build historical datasets in Iceberg

### Best Practices

1. **Schema Evolution**: Use Avro schema registry for backward compatibility
2. **Idempotency**: Consumers must handle duplicate events (at-least-once delivery)
3. **Monitoring**: Track CDC lag, consumer lag, and error rates
4. **Backpressure**: Configure consumer batch sizes to prevent overwhelming downstream
5. **Dead Letter Queues**: Route failed messages for manual review

### Example: Payment Transaction Flow

```
Payment Service writes to Aurora:
  INSERT INTO transactions (id, merchant_id, amount, status)
  VALUES (123, 456, 1000, 'SUCCESS')

  ↓

Debezium captures from binlog (< 1 second):
  {
    "op": "c",  // create
    "before": null,
    "after": {"id": 123, "merchant_id": 456, "amount": 1000, "status": "SUCCESS"},
    "ts_ms": 1675890123456
  }

  ↓

Published to Kafka topic: cdc.payments.transactions
  Partition: 3 (hash of id=123)
  Offset: 987654

  ↓

Consumed by multiple systems:
  - TiDB: Applied via Spark Streaming (~10 sec)
  - Pinot: Indexed for dashboards (~30 sec)
  - Iceberg: Batch written every 5 min
```

### Freshness Guarantees

- **TiDB**: ~10 seconds (p99)
- **Pinot**: ~30 seconds (p99)
- **Iceberg**: 5-15 minutes (micro-batches)

### Failure Scenarios

**Source Database Unavailable:**
- Debezium retries with exponential backoff
- Kafka retains events for 7 days (can catch up)

**Consumer Failures:**
- Kafka consumer group maintains offsets
- Consumers resume from last committed offset
- No data loss (at-least-once semantics)

**Schema Changes:**
- Avro schema registry validates compatibility
- Breaking changes rejected (must use new topic)

---

## Pattern 2: Batch Ingestion Pattern

### Architecture

```
Source DB (Aurora)
    ↓
RDS Snapshot → S3
    ↓
Spark Batch Job
    ↓
Iceberg Table (MERGE/OVERWRITE)
```

### How It Works

1. **Snapshot Creation**: Periodic RDS snapshots exported to S3
2. **Data Extraction**: Spark reads Parquet files from S3
3. **Transformation**: Apply business logic, cleansing, enrichment
4. **Loading**: Write to Iceberg using MERGE (upsert) or OVERWRITE

### Implementation Details

**Snapshot Export:**
```bash
# Automated via AWS RDS Export
aws rds start-export-task \
  --export-task-identifier payments-daily-export \
  --source-arn arn:aws:rds:us-east-1:123456:snapshot:payments-2024-01-01 \
  --s3-bucket-name rds-exports \
  --s3-prefix payments/2024-01-01/ \
  --iam-role-arn arn:aws:iam::123456:role/rds-export
```

**Spark ETL:**
```python
# Read from S3
df = spark.read.parquet("s3://rds-exports/payments/2024-01-01/transactions/")

# Transform
df_clean = df.filter(col("status").isNotNull()) \
  .withColumn("created_date", to_date(col("created_at")))

# Write to Iceberg (upsert by primary key)
df_clean.writeTo("warehouse.payments.transactions") \
  .using("iceberg") \
  .tableProperty("write.merge.mode", "merge-on-read") \
  .option("mergeSchema", "true") \
  .createOrReplace()
```

### Use Cases

- **Historical backfills**: Load historical data into data lake
- **Full table refreshes**: Periodic snapshots for slowly changing data
- **Initial migrations**: One-time bulk loads from legacy systems
- **Cross-region replication**: Copy data across AWS regions

### Best Practices

1. **Incremental Loading**: Use watermarks to load only new/changed data
2. **Partitioning**: Partition Iceberg tables by date for efficient queries
3. **Schema Validation**: Validate schema before writing
4. **Idempotency**: Use MERGE instead of INSERT to handle reruns
5. **Cost Optimization**: Schedule during off-peak hours

### Example: Merchant Master Data

```
Daily batch job (runs at 2 AM UTC):

1. RDS Export (30 min):
   Aurora snapshot → S3 Parquet

2. Spark Transformation (15 min):
   - Read from S3
   - Join with dimension tables
   - Apply business rules
   - Deduplicate

3. Iceberg Load (10 min):
   - MERGE INTO warehouse.merchants
   - Update existing records
   - Insert new records

Total latency: ~1 hour
Freshness: T-1 (previous day's data)
```

---

## Pattern 3: Stream Processing Pattern

### Architecture

```
Kafka Topic (raw events)
    ↓
Spark Structured Streaming / Flink
    ↓
Transform + Enrich + Aggregate
    ↓
Target System (Pinot / TiDB / Iceberg)
```

### How It Works

1. **Source**: Read from Kafka topics
2. **Processing**: Real-time transformations, joins, aggregations
3. **Sink**: Write to downstream systems

### Implementation Details

**Spark Structured Streaming:**
```python
# Read from Kafka
events = spark.readStream \
  .format("kafka") \
  .option("kafka.bootstrap.servers", "kafka:9092") \
  .option("subscribe", "cdc.payments.transactions") \
  .load()

# Parse and transform
payments = events.select(
  from_json(col("value").cast("string"), payment_schema).alias("data")
).select("data.*")

# Enrich with merchant data (stream-static join)
enriched = payments.join(
  merchant_dim,
  payments.merchant_id == merchant_dim.id,
  "left"
)

# Aggregate (windowed)
aggregated = enriched.groupBy(
  window(col("created_at"), "5 minutes"),
  col("merchant_id")
).agg(
  sum("amount").alias("total_amount"),
  count("*").alias("transaction_count")
)

# Write to Pinot
aggregated.writeStream \
  .format("pinot") \
  .option("table", "payment_aggregates") \
  .option("checkpointLocation", "s3://checkpoints/") \
  .start()
```

**Spark Streaming for TiDB Sink:**
```python
# Read from Kafka
kafka_stream = spark.readStream \
  .format("kafka") \
  .option("kafka.bootstrap.servers", "kafka:9092") \
  .option("subscribe", "cdc.payments.transactions") \
  .load()

# Transform and filter
transactions = kafka_stream.select(
  from_json(col("value").cast("string"), schema).alias("data")
).select("data.*") \
  .filter(col("status") == "SUCCESS")

# Write to TiDB using foreachBatch
def write_to_tidb(batch_df, batch_id):
  batch_df.write \
    .format("jdbc") \
    .option("url", "jdbc:mysql://tidb-server:4000/warehouse") \
    .option("dbtable", "transactions_enriched") \
    .option("user", "spark_user") \
    .option("password", "***") \
    .option("batchsize", 1000) \
    .option("isolationLevel", "READ_COMMITTED") \
    .mode("append") \  # UPSERT handled via TiDB ON DUPLICATE KEY
    .save()

transactions.writeStream \
  .foreachBatch(write_to_tidb) \
  .option("checkpointLocation", "s3://checkpoints/tidb-sink") \
  .trigger(processingTime="10 seconds") \
  .start()
```

### Use Cases

- **Real-time dashboards**: Pre-aggregate data for Pinot/ClickHouse
- **Data enrichment**: Join streams with dimension tables
- **Real-time alerting**: Detect anomalies and trigger actions
- **Data quality**: Validate and cleanse in real-time
- **TiDB replication**: Real-time CDC sync from Aurora to TiDB

### Best Practices

1. **Watermarking**: Handle late-arriving data with watermarks
2. **State Management**: Use checkpoints for fault tolerance
3. **Backpressure**: Configure buffer sizes appropriately
4. **Monitoring**: Track processing lag and throughput
5. **Testing**: Test with historical data replay

---

## Pattern 4: Lakehouse Pattern

### Architecture

```
Kafka → Spark → Iceberg (storage layer)
                    ↓
                    ├→ Trino (interactive queries)
                    └→ Spark (batch processing)
```

### How It Works

1. **Ingestion**: Spark consumes from Kafka
2. **Storage**: Write to Iceberg with ACID guarantees
3. **Consumption**: Query with Trino or process with Spark

### Key Features

- **ACID Transactions**: Iceberg provides snapshot isolation
- **Time Travel**: Query historical versions
- **Schema Evolution**: Add/drop columns safely
- **Partition Evolution**: Change partitioning without rewriting data

### Implementation

```python
# Write with MERGE (upserts)
spark.sql("""
  MERGE INTO warehouse.payments.transactions t
  USING updates u
  ON t.id = u.id
  WHEN MATCHED THEN UPDATE SET *
  WHEN NOT MATCHED THEN INSERT *
""")

# Query with Trino
SELECT merchant_id, SUM(amount)
FROM iceberg.payments.transactions
WHERE created_date >= CURRENT_DATE - INTERVAL '30' DAY
GROUP BY merchant_id
```

---

## Pattern 5: Federation Pattern

### Architecture

```
Trino Coordinator
    ↓
    ├→ Iceberg Connector (S3 data)
    ├→ MySQL Connector (Aurora)
    ├→ Pinot Connector (OLAP)
    └→ Memory Connector (temp tables)
```

### How It Works

Trino federates queries across multiple data sources in a single SQL query.

### Implementation

```sql
-- Join data from Iceberg, MySQL, and Pinot
SELECT
  i.transaction_id,
  m.merchant_name,
  p.payment_count
FROM iceberg.warehouse.transactions i
JOIN mysql.prod.merchants m ON i.merchant_id = m.id
LEFT JOIN pinot.default.merchant_stats p ON m.id = p.merchant_id
WHERE i.created_date >= CURRENT_DATE - INTERVAL '7' DAY
```

### Use Cases

- **Cross-system exploration**: Join data without copying
- **Migration validation**: Compare old vs new systems
- **Debugging**: Query production data alongside analytics

### Best Practices

1. **Push down predicates**: Filter before federation
2. **Broadcast joins**: Join small dimension tables
3. **Limit cross-cluster joins**: Network overhead is high
4. **Cache dimension tables**: Use Trino's connector caching

---

## Choosing the Right Pattern

| Pattern | Latency | Complexity | Use Case |
|---------|---------|------------|----------|
| CDC | Seconds | Medium | Real-time replication |
| Batch | Hours | Low | Historical loads |
| Stream Processing | Seconds-Minutes | High | Real-time transformation |
| Lakehouse | Minutes | Medium | Analytical storage |
| Federation | Seconds-Minutes | Low | Cross-system queries |

---

## Monitoring and Observability

### Key Metrics

**CDC Pattern:**
- Debezium lag (ms behind source)
- Kafka consumer lag (messages behind)
- Event throughput (events/sec)

**Batch Pattern:**
- Job duration (minutes)
- Records processed (count)
- Data quality issues (count)

**Stream Processing:**
- Processing lag (seconds)
- Checkpoint interval (seconds)
- Backpressure events (count)

### Alerting Rules

```yaml
alerts:
  - name: CDC lag high
    condition: debezium_lag_ms > 60000
    severity: warning

  - name: Consumer lag critical
    condition: kafka_consumer_lag > 1000000
    severity: critical

  - name: Batch job failed
    condition: job_status == 'FAILED'
    severity: critical
```

---

## Common Issues and Solutions

### Issue 1: CDC Lag Increasing

**Symptoms:** Debezium lag growing, consumers falling behind

**Root Causes:**
- High write volume on source database
- Slow consumers (TiDB write bottleneck)
- Network issues

**Solutions:**
- Scale consumer parallelism (increase Kafka partitions)
- Optimize downstream writes (batch inserts)
- Add consumer instances

### Issue 2: Duplicate Events

**Symptoms:** Same event processed multiple times

**Root Causes:**
- Consumer rebalancing
- Offset commit failures
- Exactly-once semantics not enabled

**Solutions:**
- Implement idempotent consumers (upsert vs insert)
- Enable exactly-once semantics (Kafka transactions)
- Use unique constraints in target systems

### Issue 3: Schema Incompatibility

**Symptoms:** Consumers failing with deserialization errors

**Root Causes:**
- Schema evolved without backward compatibility
- Schema registry misconfiguration

**Solutions:**
- Enforce schema compatibility checks
- Use schema evolution best practices (add optional fields)
- Version topics for breaking changes

---

## References

- [Debezium Documentation](https://debezium.io/documentation/)
- [Apache Iceberg Specification](https://iceberg.apache.org/spec/)
- [Kafka Streams Documentation](https://kafka.apache.org/documentation/streams/)
- [Trino Federation Guide](https://trino.io/docs/current/connector.html)

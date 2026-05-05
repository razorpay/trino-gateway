# Storage Systems Deep Dive

This document provides detailed information about each storage tier in the data platform, including internal architectures, capabilities, and best practices.

## Overview

The platform uses a multi-tier storage architecture, each optimized for different access patterns and cost profiles:

| Tier | System | Latency | Cost | Use Case |
|------|--------|---------|------|----------|
| Hot | Aurora, DynamoDB | Milliseconds | High | Transactional, application DB |
| Warm | TiDB | Milliseconds | Medium-High | Operational queries, debugging |
| OLAP | Pinot, ClickHouse | Milliseconds | Medium | User-facing dashboards |
| Cold | Iceberg Lakehouse | Seconds | Low | Historical analytics, ML |

---

## Hot Storage: Aurora & DynamoDB

### Purpose

Source of truth for application data. Optimized for transactional workloads with ACID guarantees.

### Aurora (MySQL)

**Architecture:**
```
Application Layer
    ↓
Aurora Cluster (Multi-AZ)
    ├→ Writer Instance (single)
    └→ Reader Instances (0-15 replicas)
    ↓
Storage Layer (6-way replication across AZs)
```

**Key Features:**
- MySQL-compatible
- Up to 15 read replicas
- Automated failover (< 30 seconds)
- Point-in-time recovery
- Continuous backup to S3

**Performance Characteristics:**
- Latency: 1-5ms (single-region)
- Throughput: 500K reads/sec, 100K writes/sec (per instance)
- IOPS: Auto-scaling (up to 256K IOPS)
- Storage: Auto-scaling (10GB to 128TB)

**Use Cases:**
- Application database (payments, users, merchants)
- Transactional workloads requiring ACID
- Low-latency point queries and updates

**Cost Model:**
```
Cost = Instance cost + Storage cost + IOPS cost + Backup cost

Example (db.r6g.xlarge):
- Instance: $0.29/hour (~$210/month)
- Storage: $0.10/GB/month
- IOPS: $0.20 per 1M requests
- Backup: $0.021/GB/month

Total for 1TB database: ~$500-800/month
```

**Limitations:**
- **Not for analytics**: Row-oriented, slow for large scans
- **Cost**: Expensive for large datasets
- **Scaling**: Vertical scaling (instance size limits)
- **Production impact**: Heavy queries affect application performance

**Best Practices:**
1. **Enable CDC**: Replicate to warm/cold tiers for analytics
2. **Read replicas**: Offload read traffic from writer
3. **Connection pooling**: Limit connection overhead
4. **Monitoring**: Track slow queries, IOPS saturation
5. **Backups**: Automate snapshots for disaster recovery

### DynamoDB

**Architecture:**
```
Application Layer
    ↓
DynamoDB API
    ↓
Partition Key Routing
    ↓
Storage Nodes (3-way replication)
```

**Key Features:**
- NoSQL key-value/document store
- Auto-scaling throughput
- Single-digit millisecond latency
- Global tables (multi-region)
- DynamoDB Streams (CDC)

**Performance Characteristics:**
- Latency: 1-10ms (p99)
- Throughput: Millions of requests/sec (auto-scales)
- Partition: 3000 RCU / 1000 WCU per partition
- Storage: Unlimited (auto-partitioning)

**Use Cases:**
- User sessions and profiles
- Real-time voting/leaderboards
- IoT data ingestion
- Shopping carts

**Cost Model:**
```
On-Demand:
- Write: $1.25 per 1M requests
- Read: $0.25 per 1M requests
- Storage: $0.25/GB/month

Provisioned (cheaper for predictable load):
- Write: $0.00065 per WCU-hour
- Read: $0.00013 per RCU-hour
```

**Limitations:**
- **No joins**: Must denormalize data
- **Query limitations**: Can only query by partition key + sort key
- **Item size**: Max 400KB per item
- **Hot partitions**: Uneven access patterns cause throttling

**Best Practices:**
1. **Partition key design**: Distribute load evenly
2. **DynamoDB Streams**: Enable for CDC to analytics tier
3. **Global secondary indexes**: Support additional access patterns
4. **On-demand vs provisioned**: Choose based on traffic predictability
5. **Avoid scans**: Use queries with partition key

---

## Warm Storage: TiDB

### Purpose

Distributed SQL database for operational queries requiring low latency and fresh data (near real-time).

### Architecture

```
Application Layer
    ↓
TiDB Server (SQL layer, stateless)
    ↓
PD (Placement Driver - metadata, scheduling)
    ↓
TiKV (Storage layer, distributed key-value)
    └→ RocksDB (LSM-tree on local SSD)
```

**Components:**

1. **TiDB Server**: MySQL-compatible SQL layer (stateless, horizontally scalable)
2. **PD (Placement Driver)**: Cluster metadata, timestamp oracle, scheduling
3. **TiKV**: Distributed transactional key-value store (Raft consensus)

**Key Features:**
- MySQL-compatible (works with existing tools)
- Horizontal scalability (add nodes for capacity)
- ACID transactions (distributed transactions via 2PC)
- Real-time analytics (TiFlash columnar engine)
- Online DDL (schema changes without downtime)

### Performance Characteristics

- **Latency**: 5-50ms (p99) for point queries
- **Throughput**: 100K+ QPS (depends on cluster size)
- **Concurrency**: High (100s of concurrent connections)
- **Storage**: Petabyte-scale (3-way replication)

### Data Ingestion

**CDC from Aurora:**
```
Aurora → Debezium → Kafka → Spark Streaming → TiDB
Latency: ~10 seconds end-to-end
```

**Real-time Writes:**
```
Application → TiDB (direct writes)
Latency: 5-20ms
```

### Use Cases

**Ideal:**
- Operational dashboards (internal teams)
- Customer support tools (near real-time data)
- Debugging production issues (fresh data)
- Transactional + analytical hybrid workloads

**Not Ideal:**
- User-facing dashboards (use OLAP for higher concurrency)
- Large analytical scans (use Iceberg for lower cost)
- Batch ETL (use Spark for throughput)

### Cost Model

```
Cost = Compute cost + Storage cost + Network cost

Example (3-node cluster):
- TiDB Server (2x c5.2xlarge): $0.34/hour × 2 = $490/month
- TiKV (3x i3.2xlarge): $0.624/hour × 3 = $1,350/month
- PD (3x c5.large): $0.085/hour × 3 = $185/month

Total: ~$2,000/month for 3-node cluster (500GB-1TB capacity)

Storage cost: ~$0.08/GB/month (including replication)
```

**Cost Comparison:**
- **vs Aurora**: 2-3x more expensive per GB (distributed architecture overhead)
- **vs Iceberg**: 3-4x more expensive (SSD vs S3)
- **vs Pinot**: Similar cost (both use SSD)

### TiFlash (Columnar Engine)

**Architecture:**
```
TiDB Server
    ↓
    ├→ TiKV (row-oriented, OLTP)
    └→ TiFlash (columnar, OLAP)
```

**Use Case**: Run analytical queries on same cluster without impacting OLTP.

**Limitations:**
- Adds storage cost (columnar replica)
- Still slower than dedicated OLAP (Pinot)
- Better suited for hybrid workloads

### Scaling

**Vertical Scaling:**
- Increase instance sizes (CPU, memory)
- Downtime required for some nodes

**Horizontal Scaling:**
- Add TiDB servers (SQL layer) for more concurrency
- Add TiKV nodes (storage) for more capacity
- Online operation (no downtime)

### Best Practices

1. **Schema design**: Use proper indexes (secondary, covering)
2. **Hotspot avoidance**: Avoid auto-increment PKs (use UUID or SHARD_ROW_ID_BITS)
3. **Read/write separation**: Use TiFlash for analytical queries
4. **Monitoring**: Track slow queries, hotspot regions
5. **Capacity planning**: Monitor disk usage (TiKV local SSD)

---

## OLAP Storage: Pinot & ClickHouse

### Purpose

Real-time OLAP systems for user-facing dashboards and high-concurrency analytical queries.

### Apache Pinot

**Architecture:**
```
Query Layer
    ↓
Broker (query routing, scatter-gather)
    ↓
Server (query execution, data storage)
    ├→ Real-time Segments (Kafka ingestion)
    └→ Offline Segments (batch from S3)
    ↓
Deep Storage (S3)
```

**Components:**

1. **Controller**: Cluster management, segment assignment
2. **Broker**: Query routing, result merging
3. **Server**: Data storage, query execution
4. **Minion**: Offline tasks (compaction, rebalancing)

**Key Features:**
- Sub-second latency at high concurrency (1000+ QPS)
- Real-time ingestion from Kafka
- Star-tree indexes for fast aggregations
- Inverted indexes for filtering
- Tiered storage (hot/cold)

**Performance Characteristics:**
- **Latency**: 10-100ms (p99) for indexed queries
- **Throughput**: 1000+ QPS per cluster
- **Concurrency**: Very high (10K+ concurrent users)
- **Freshness**: Seconds (real-time from Kafka)

**Data Model:**

*Schema:*
```json
{
  "schemaName": "payment_events",
  "dimensionFieldSpecs": [
    {"name": "merchant_id", "dataType": "LONG"},
    {"name": "payment_method", "dataType": "STRING"},
    {"name": "status", "dataType": "STRING"}
  ],
  "metricFieldSpecs": [
    {"name": "amount", "dataType": "DOUBLE"},
    {"name": "fee", "dataType": "DOUBLE"}
  ],
  "dateTimeFieldSpecs": [
    {"name": "created_at", "dataType": "LONG", "format": "EPOCH", "granularity": "1:MILLISECONDS"}
  ]
}
```

*Star-Tree Index:*
```
Dimensions: [merchant_id, payment_method, created_at_day]
Metrics: [SUM(amount), COUNT(*)]

Pre-aggregated combinations:
- (merchant_id, *, *) → fast merchant-level queries
- (*, payment_method, *) → fast payment method breakdown
- (merchant_id, payment_method, created_at_day) → fast time-series
```

**Ingestion:**

*Real-time (Kafka):*
```
Kafka Topic → Pinot Real-time Segment → Persist to S3 (hourly)
Latency: < 1 second
```

*Batch (S3):*
```
Spark → Parquet/Avro on S3 → Pinot Offline Segment
Latency: Daily batch
```

**Use Cases:**

**Ideal:**
- User-facing dashboards (merchant analytics, customer dashboards)
- High-concurrency APIs (100s-1000s QPS)
- Fixed query patterns (pre-defined KPIs)
- Real-time ingestion (Kafka streams)

**Not Ideal:**
- Ad-hoc exploration (rigid schema)
- Complex joins (single-table queries only)
- Unindexed queries (full table scans are slow)

**Cost Model:**
```
Cost = Server instances + Storage (S3) + Network

Example (3-broker, 6-server cluster):
- Brokers (3x c5.2xlarge): $0.34/hour × 3 = $735/month
- Servers (6x i3.2xlarge): $0.624/hour × 6 = $2,700/month
- Deep storage (S3): $0.023/GB/month

Total for 5TB dataset: ~$3,500/month (serves 100K+ QPS)

Cost per QPS: ~$0.035/QPS/month (highly cost-efficient)
```

**Best Practices:**

1. **Schema design**: Denormalize and pre-join (no joins at query time)
2. **Indexing**: Use star-tree for aggregations, inverted for filters
3. **Partitioning**: Partition by time (enables pruning)
4. **Retention**: Configure retention policies (delete old segments)
5. **Monitoring**: Track query latency, segment health

### ClickHouse

**Architecture:**
```
Query Layer
    ↓
ClickHouse Server (distributed query)
    ↓
Shards (data partitioning)
    ↓
Replicas (data replication)
    ↓
Local Storage (MergeTree tables)
```

**Key Features:**
- Columnar storage (efficient compression)
- Vectorized query execution
- Real-time ingestion
- Materialized views for pre-aggregation
- Distributed queries (sharding + replication)

**Performance Characteristics:**
- **Latency**: 10-500ms (depends on query complexity)
- **Throughput**: 100s QPS per server
- **Concurrency**: Medium-high (100s concurrent queries)
- **Compression**: 10-50x compression ratios

**Data Model:**

*MergeTree Table:*
```sql
CREATE TABLE payments (
    created_date Date,
    merchant_id UInt64,
    amount Decimal(18, 2),
    status String
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_date)
ORDER BY (merchant_id, created_date)
SETTINGS index_granularity = 8192;
```

**Use Cases:**

**Ideal:**
- Internal dashboards (BI tools)
- Log analytics (high compression)
- Time-series analytics
- Pre-aggregated reports

**Not Ideal:**
- User-facing APIs (lower concurrency than Pinot)
- Transactional updates (append-only)
- Small point queries (optimized for scans)

**Cost Model:**
```
Similar to Pinot, slightly lower storage cost (better compression)
```

**Pinot vs ClickHouse:**

| Feature | Pinot | ClickHouse |
|---------|-------|------------|
| Concurrency | Very High (1000+ QPS) | Medium-High (100s QPS) |
| Latency | Lower (10-100ms) | Variable (10-500ms) |
| Real-time Ingestion | Excellent (Kafka-native) | Good (Kafka + custom) |
| Joins | None (denormalize) | Limited (distributed joins slow) |
| Use Case | User-facing dashboards | Internal analytics |

---

## Cold Storage: Iceberg Lakehouse

### Purpose

Low-cost, petabyte-scale storage for historical analytics, ML, and long-term retention.

### Architecture

```
Metadata Layer
    ↓
Iceberg Catalog (Hive Metastore / Glue / Nessie)
    ↓
Metadata Files (JSON manifests on S3)
    ↓
Data Files (Parquet on S3)
```

**Components:**

1. **Catalog**: Tracks current table metadata (pointer to latest snapshot)
2. **Metadata Files**: Manifests (file locations, partitions, stats)
3. **Data Files**: Parquet files (columnar, compressed)

**Key Features:**
- ACID transactions (snapshot isolation)
- Time travel (query historical versions)
- Schema evolution (add/drop columns safely)
- Partition evolution (change partitioning without rewrite)
- Hidden partitioning (no partition predicates in queries)
- Copy-on-write and merge-on-read

### Performance Characteristics

- **Latency**: Seconds to minutes (depends on query, data size)
- **Throughput**: High for batch scans (100s GB/sec)
- **Storage cost**: $0.023/GB/month (S3 standard)
- **Compute cost**: Pay-per-query (Trino, Spark)

### Data Model

**Table Creation:**
```sql
CREATE TABLE warehouse.payments.transactions (
    transaction_id BIGINT,
    merchant_id BIGINT,
    amount DECIMAL(18, 2),
    created_at TIMESTAMP,
    created_date DATE
)
USING iceberg
PARTITIONED BY (days(created_date))
TBLPROPERTIES (
    'write.format.default' = 'parquet',
    'write.parquet.compression-codec' = 'snappy'
);
```

**Partitioning:**
```
Partitioning by day:
s3://warehouse/payments/transactions/data/
  created_date=2024-01-01/
    part-00000.parquet
    part-00001.parquet
  created_date=2024-01-02/
    part-00000.parquet
```

### ACID Transactions

**Snapshot Isolation:**
```
Write 1: Add new data → Snapshot 1
Write 2: Update data → Snapshot 2
Write 3: Delete data → Snapshot 3

Readers see consistent snapshot (no dirty reads)
Concurrent writes coordinated via optimistic concurrency
```

**MERGE (Upsert):**
```sql
MERGE INTO warehouse.payments.transactions t
USING updates u
ON t.transaction_id = u.transaction_id
WHEN MATCHED THEN UPDATE SET *
WHEN NOT MATCHED THEN INSERT *;
```

### Time Travel

**Query Historical Versions:**
```sql
-- Query snapshot as of timestamp
SELECT * FROM warehouse.payments.transactions
FOR SYSTEM_TIME AS OF TIMESTAMP '2024-01-01 00:00:00';

-- Query specific snapshot ID
SELECT * FROM warehouse.payments.transactions
FOR SYSTEM_VERSION AS OF 12345;

-- Show snapshot history
SELECT * FROM warehouse.payments.transactions.snapshots;
```

### Schema Evolution

**Add Column (non-breaking):**
```sql
ALTER TABLE warehouse.payments.transactions
ADD COLUMN currency STRING;

-- Old data: NULL for new column
-- New data: Populated
-- Queries work seamlessly
```

**Rename Column (breaking - use views):**
```sql
-- Create view with old column name
CREATE VIEW warehouse.payments.transactions_v1 AS
SELECT transaction_id AS old_txn_id, ...
FROM warehouse.payments.transactions;
```

### Ingestion Patterns

**Batch (Spark):**
```python
# Daily batch load
df = spark.read.parquet("s3://raw/payments/2024-01-01/")

df.writeTo("warehouse.payments.transactions") \
  .using("iceberg") \
  .append()  # or .overwritePartitions() for idempotency
```

**Streaming (Spark Structured Streaming):**
```python
# Micro-batch from Kafka
spark.readStream \
  .format("kafka") \
  .option("subscribe", "cdc.payments.transactions") \
  .load() \
  .writeStream \
  .format("iceberg") \
  .option("checkpointLocation", "s3://checkpoints/") \
  .trigger(processingTime="5 minutes") \
  .start()
```

**CDC (with MERGE):**
```python
# Apply CDC events (upserts + deletes)
cdc_df = spark.read.format("kafka").load()

# Separate inserts/updates from deletes
upserts = cdc_df.filter(col("op") != "d")
deletes = cdc_df.filter(col("op") == "d")

# Apply MERGE
spark.sql("""
  MERGE INTO warehouse.payments.transactions t
  USING upserts u
  ON t.id = u.id
  WHEN MATCHED THEN UPDATE SET *
  WHEN NOT MATCHED THEN INSERT *
""")

# Delete
spark.sql("""
  DELETE FROM warehouse.payments.transactions
  WHERE id IN (SELECT id FROM deletes)
""")
```

### Query Engines

**Trino (Interactive):**
```sql
-- Ad-hoc exploration
SELECT merchant_id, SUM(amount)
FROM iceberg.warehouse.payments.transactions
WHERE created_date >= DATE '2024-01-01'
GROUP BY merchant_id;

-- Latency: Seconds to minutes
```

**Spark (Batch):**
```python
# ML feature engineering
df = spark.read.table("warehouse.payments.transactions")

features = df.groupBy("merchant_id").agg(
    avg("amount").alias("avg_amount"),
    count("*").alias("txn_count")
)

features.write.mode("overwrite").parquet("s3://ml/features/")
```

### Use Cases

**Ideal:**
- Historical analytics (multi-year retention)
- ML feature engineering (large-scale processing)
- Data lake (central repository)
- Auditing (time travel)

**Not Ideal:**
- User-facing APIs (seconds latency)
- Real-time queries (batch ingestion)
- High-concurrency serving (use OLAP)

### Cost Model

```
Storage: $0.023/GB/month (S3 standard)
Compute: Pay-per-query (Trino or Spark)

Example (10TB dataset):
- Storage: 10,000 GB × $0.023 = $230/month
- Compute (Trino): $0.01-0.05 per query (depends on scan size)

Total: ~$300-500/month (storage + moderate query volume)

Cost per GB: ~$0.03/GB/month (vs $0.08 for TiDB, $0.10 for Aurora)
```

### Best Practices

1. **Partitioning**: Partition by date for efficient pruning
2. **Compaction**: Regularly compact small files (avoid small file problem)
3. **Retention**: Configure snapshot retention (delete old metadata)
4. **Bucketing**: Use bucketing for joins (co-locate data)
5. **Metadata caching**: Use Iceberg metadata caching in Trino

---

## Storage Tier Comparison

| Feature | Hot (Aurora) | Warm (TiDB) | OLAP (Pinot) | Cold (Iceberg) |
|---------|-------------|-------------|--------------|----------------|
| **Latency** | 1-5ms | 5-50ms | 10-100ms | Seconds |
| **Freshness** | Real-time | ~10s (CDC) | ~30s (CDC) | Minutes-Hours |
| **Cost** | $0.10/GB/mo | $0.08/GB/mo | $0.05/GB/mo | $0.023/GB/mo |
| **ACID** | Yes | Yes | No | Yes |
| **Joins** | Excellent | Good | None | Good |
| **Concurrency** | Medium | High | Very High | Low |
| **Retention** | Weeks-Months | Months | Months | Years |

---

## Choosing the Right Storage Tier

**Decision Factors:**

1. **Latency**: How fast do queries need to be?
2. **Freshness**: How fresh must data be?
3. **Concurrency**: How many concurrent users?
4. **Cost**: What's the storage/query budget?
5. **Retention**: How long to keep data?

**Examples:**

*User-facing payment dashboard:*
- Latency: < 100ms → OLAP (Pinot)

*Analyst exploring historical trends:*
- Latency: Seconds OK → Cold (Iceberg + Trino)

*Customer support tool:*
- Latency: < 1s, Freshness: Real-time → Warm (TiDB)

*Transactional system:*
- Latency: < 10ms, ACID required → Hot (Aurora)

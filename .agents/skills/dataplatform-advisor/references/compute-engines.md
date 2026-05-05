# Compute Engines Deep Dive

This document provides detailed information about the compute engines in the data platform: Spark, Trino, and Flink.

## Overview

The platform uses three primary compute engines, each optimized for different workloads:

| Engine | Optimized For | Latency | Execution Model | Primary Use Case |
|--------|---------------|---------|-----------------|------------------|
| **Spark** | Throughput, batch | Minutes-Hours | Distributed batch | ETL, ML, backfills |
| **Trino** | Interactive queries | Seconds-Minutes | MPP SQL | Ad-hoc analysis, BI |
| **Flink** | Stream processing | Real-time | Continuous streaming | CDC, real-time ingestion |

---

## Apache Spark

### Purpose

Distributed batch processing engine for high-throughput ETL, ML, and data transformation workloads.

### Architecture

```
Driver (Job Coordinator)
    ↓
Cluster Manager (YARN / K8s)
    ↓
Executors (Worker Nodes)
    ├→ Task 1 (Partition processing)
    ├→ Task 2
    └→ Task N
    ↓
Data Sources (S3, Iceberg, Kafka, JDBC)
```

**Components:**

1. **Driver**: Coordinates job execution, creates execution plan
2. **Executors**: Process data partitions in parallel
3. **Cluster Manager**: Allocates resources (YARN, Kubernetes, Mesos)

**Execution Model:**

1. **Lazy Evaluation**: Builds DAG (Directed Acyclic Graph) of transformations
2. **Stage Division**: Divides DAG into stages at shuffle boundaries
3. **Task Scheduling**: Schedules tasks to executors (data locality preferred)
4. **Fault Tolerance**: Recomputes lost partitions from lineage

### Key Features

- **Unified API**: SQL, DataFrame, RDD (different abstraction levels)
- **Rich Ecosystem**: 1000+ connectors, integrations
- **Fault Tolerance**: Lineage-based recovery (no checkpointing for batch)
- **Optimization**: Catalyst optimizer (predicate pushdown, column pruning)
- **Streaming**: Structured Streaming (micro-batch)

### Performance Characteristics

- **Latency**: Minutes to hours (optimized for throughput, not latency)
- **Throughput**: 100s GB/sec (depends on cluster size)
- **Scalability**: 1000s of nodes, petabyte-scale data
- **Memory**: In-memory caching (speeds up iterative algorithms)

### APIs

**DataFrame API (Recommended):**
```python
from pyspark.sql import SparkSession
from pyspark.sql.functions import col, sum, avg

spark = SparkSession.builder.appName("ETL").getOrCreate()

# Read from Iceberg
df = spark.read.table("warehouse.payments.transactions")

# Transform
result = df.filter(col("status") == "SUCCESS") \
    .groupBy("merchant_id") \
    .agg(
        sum("amount").alias("total_amount"),
        avg("amount").alias("avg_amount"),
        count("*").alias("txn_count")
    )

# Write to Iceberg
result.writeTo("warehouse.analytics.merchant_stats") \
    .using("iceberg") \
    .createOrReplace()
```

**Spark SQL:**
```sql
-- Create temp view
CREATE OR REPLACE TEMP VIEW payments AS
SELECT * FROM iceberg.warehouse.payments.transactions
WHERE created_date >= CURRENT_DATE - INTERVAL 30 DAYS;

-- Query
SELECT merchant_id, SUM(amount) as total
FROM payments
WHERE status = 'SUCCESS'
GROUP BY merchant_id;
```

**Structured Streaming:**
```python
# Real-time processing from Kafka
stream = spark.readStream \
    .format("kafka") \
    .option("kafka.bootstrap.servers", "kafka:9092") \
    .option("subscribe", "payments") \
    .load()

# Transform
parsed = stream.selectExpr("CAST(value AS STRING)") \
    .select(from_json(col("value"), schema).alias("data")) \
    .select("data.*")

# Write to Iceberg (micro-batch)
query = parsed.writeStream \
    .format("iceberg") \
    .outputMode("append") \
    .option("checkpointLocation", "s3://checkpoints/") \
    .trigger(processingTime="5 minutes") \
    .start()
```

### Use Cases

**Ideal:**
- **ETL pipelines**: Transform and load data between systems
- **Batch processing**: Historical data processing, backfills
- **ML training**: Feature engineering, model training (MLlib, PyTorch)
- **Pre-aggregation**: Prepare data for OLAP systems (denormalize, pre-join)
- **Data quality**: Validation, cleansing, deduplication

**Not Ideal:**
- **Interactive queries**: Use Trino (Spark has high startup overhead)
- **Real-time streaming**: Use Flink (Spark Streaming is micro-batch)
- **User-facing APIs**: Use OLAP (Spark is too slow)

### Deployment Modes

**Ephemeral Clusters (Recommended):**
```bash
# Launch cluster for single job
spark-submit \
  --master yarn \
  --deploy-mode cluster \
  --num-executors 50 \
  --executor-cores 4 \
  --executor-memory 16g \
  --conf spark.dynamicAllocation.enabled=true \
  etl_job.py

# Cluster automatically terminated after job completes
```

**Long-running Clusters:**
```bash
# Shared cluster for Spark Thrift Server (SQL endpoint)
spark-submit \
  --class org.apache.spark.sql.hive.thriftserver.HiveThriftServer2 \
  --master yarn \
  --deploy-mode cluster
```

### Optimizations

**1. Partitioning:**
```python
# Repartition before expensive operations
df.repartition(200, "merchant_id") \
    .groupBy("merchant_id") \
    .agg(...)
```

**2. Caching:**
```python
# Cache intermediate results for reuse
df_cached = df.filter(...).cache()
result1 = df_cached.groupBy(...).agg(...)
result2 = df_cached.groupBy(...).agg(...)
```

**3. Broadcast Joins:**
```python
from pyspark.sql.functions import broadcast

# Broadcast small dimension tables
fact.join(broadcast(dim), fact.dim_id == dim.id)
```

**4. Predicate Pushdown:**
```python
# Filter early (pushed down to storage)
df = spark.read.table("warehouse.transactions") \
    .filter(col("created_date") >= "2024-01-01")  # Pushed to Iceberg
```

**5. Column Pruning:**
```python
# Select only needed columns
df.select("id", "amount", "merchant_id")  # Pushed to Parquet
```

### Monitoring

**Key Metrics:**
- **Job Duration**: Total time to complete
- **Stage Duration**: Time per stage (identify bottlenecks)
- **Shuffle Read/Write**: Data shuffled between executors
- **Task Failures**: Failed tasks (data skew, OOM)
- **GC Time**: Garbage collection overhead

**Spark UI:**
```
http://<driver-host>:4040

Tabs:
- Jobs: View jobs and stages
- Stages: Task-level metrics
- Storage: Cached RDDs/DataFrames
- Executors: Executor metrics (CPU, memory, GC)
- SQL: Query plans, execution times
```

### Cost Model

**EMR (AWS):**
```
Cost = EC2 cost + EMR cost

Example (50-node cluster for 1 hour):
- EC2 (50x r5.xlarge): $0.252/hour × 50 = $12.60
- EMR surcharge: $0.076/hour × 50 = $3.80
Total: $16.40/hour

Ephemeral cluster for 2-hour job: ~$33
```

**Kubernetes:**
```
Cost = Node cost (pay for compute time)

Example (on-demand):
- 50 pods × 4 cores × 16GB RAM × 2 hours
- Cost: ~$30-40 (depends on instance type)
```

### Best Practices

1. **Ephemeral clusters**: Launch per-job (avoid idle cost)
2. **Dynamic allocation**: Enable for automatic scaling
3. **Partitioning**: Align partitions with parallelism (200-1000 partitions)
4. **Avoid shuffles**: Minimize shuffles (expensive network I/O)
5. **Monitor skew**: Detect and fix data skew (uneven partition sizes)
6. **Compression**: Use Snappy for balance (speed vs size)

---

## Trino (formerly PrestoSQL)

### Purpose

Distributed SQL query engine for interactive, ad-hoc analytics across multiple data sources.

### Architecture

```
Client (JDBC, CLI, BI Tool)
    ↓
Coordinator (Query planning, orchestration)
    ↓
Workers (Query execution)
    ↓
Connectors (Iceberg, MySQL, Pinot, S3, etc.)
    ↓
Data Sources
```

**Components:**

1. **Coordinator**: Receives queries, creates execution plan, coordinates workers
2. **Workers**: Execute query fragments, process data
3. **Connectors**: Interface with data sources (Iceberg, MySQL, Kafka, etc.)

**Execution Model:**

1. **Parse & Analyze**: Parse SQL, resolve tables/columns
2. **Plan**: Create distributed execution plan (stages, tasks)
3. **Schedule**: Assign tasks to workers (data locality)
4. **Execute**: Workers execute tasks in pipeline fashion
5. **Return**: Results streamed back to client

### Key Features

- **MPP (Massively Parallel Processing)**: All workers execute in parallel
- **Pipelined Execution**: Results streamed (no intermediate materialization)
- **Cost-Based Optimizer**: Chooses optimal join order, strategies
- **Federation**: Query multiple data sources in single SQL
- **ANSI SQL**: Full SQL support (window functions, CTEs, subqueries)

### Performance Characteristics

- **Latency**: Seconds to minutes (interactive)
- **Throughput**: 100s MB-GB/sec per query
- **Concurrency**: Low-medium (10-50 queries on shared cluster)
- **Memory**: In-memory execution (OOM on large aggregations)

### Connectors

**Iceberg Connector:**
```sql
-- Query Iceberg lakehouse
SELECT merchant_id, SUM(amount)
FROM iceberg.warehouse.payments.transactions
WHERE created_date >= DATE '2024-01-01'
GROUP BY merchant_id;
```

**MySQL Connector:**
```sql
-- Query production database (read-only)
SELECT id, name
FROM mysql.prod.merchants
WHERE status = 'ACTIVE';
```

**Pinot Connector:**
```sql
-- Query OLAP system
SELECT merchant_id, payment_count
FROM pinot.default.merchant_stats
WHERE __timeColumn__ >= NOW() - INTERVAL '1' DAY;
```

**Federation (Join Across Sources):**
```sql
-- Join Iceberg + MySQL
SELECT
    t.transaction_id,
    t.amount,
    m.merchant_name
FROM iceberg.warehouse.transactions t
JOIN mysql.prod.merchants m ON t.merchant_id = m.id
WHERE t.created_date = CURRENT_DATE;
```

### Use Cases

**Ideal:**
- **Ad-hoc exploration**: Data analysts querying lakehouse
- **BI tools**: Metabase, Looker, Tableau (variable queries)
- **Debugging**: Engineers exploring production issues
- **Cross-system queries**: Joining data across systems
- **Data migration validation**: Compare old vs new systems

**Not Ideal:**
- **User-facing APIs**: Unpredictable latency, no SLA
- **High concurrency**: Shared cluster, queuing
- **Heavy transformations**: Use Spark (better fault tolerance)
- **Real-time ingestion**: Use Flink (Trino is read-only)

### Query Optimization

**1. Partition Pruning:**
```sql
-- Pushes filter to Iceberg (reads only relevant partitions)
SELECT * FROM transactions
WHERE created_date = DATE '2024-01-01';
```

**2. Predicate Pushdown:**
```sql
-- Pushes filter to MySQL (reduces data transferred)
SELECT * FROM mysql.prod.merchants
WHERE country = 'IN';
```

**3. Join Optimization:**
```sql
-- Broadcast small tables
SELECT /*+ BROADCAST(dim) */ *
FROM fact JOIN dim ON fact.dim_id = dim.id;
```

**4. Column Pruning:**
```sql
-- Only reads needed columns (Parquet column projection)
SELECT id, amount FROM transactions;
```

### Resource Management

**Resource Groups:**
```json
{
  "rootGroups": [
    {
      "name": "global",
      "softMemoryLimit": "80%",
      "hardConcurrencyLimit": 100,
      "subGroups": [
        {
          "name": "data_analysts",
          "softMemoryLimit": "50%",
          "hardConcurrencyLimit": 50,
          "schedulingPolicy": "fair"
        },
        {
          "name": "dashboards",
          "softMemoryLimit": "30%",
          "hardConcurrencyLimit": 30,
          "schedulingPolicy": "weighted"
        }
      ]
    }
  ]
}
```

### Monitoring

**Key Metrics:**
- **Query Duration**: Total query time
- **Queued Queries**: Queries waiting for resources
- **Failed Queries**: OOM, timeouts, errors
- **Data Scanned**: Amount of data read
- **CPU Time**: Total CPU consumed

**Trino UI:**
```
http://<coordinator-host>:8080

Tabs:
- Query List: Active and completed queries
- Query Details: Execution plan, stages, operators
- Workers: Worker status, memory usage
- Resource Groups: Resource allocation
```

### Cost Model

**On-Demand:**
```
Cost = Node cost × Query duration

Example (10-node cluster):
- Coordinator (1x r5.4xlarge): $1.008/hour
- Workers (10x r5.2xlarge): $0.504/hour × 10 = $5.04/hour
Total: $6/hour

Query running 5 minutes: ~$0.50
```

**Reserved Cluster:**
```
Fixed cost for dedicated cluster

Example (10-node cluster, 24/7):
- Monthly cost: ~$4,500
- Cost per query: Amortized across all queries
```

### Best Practices

1. **Partition pruning**: Always filter on partition columns
2. **Limit concurrency**: Prevent resource exhaustion (resource groups)
3. **Small result sets**: Avoid `SELECT *` (use LIMIT for exploration)
4. **Federation sparingly**: Network overhead is high
5. **Monitor query plans**: Use EXPLAIN to understand execution

---

## Apache Flink

### Purpose

Distributed stream processing engine for real-time data pipelines and event-driven applications.

### Architecture

```
JobManager (Coordination)
    ↓
TaskManagers (Execution)
    ↓
Operators (Map, Filter, Window, Join)
    ↓
State Backend (RocksDB, memory)
    ↓
Sources (Kafka, Kinesis) → Sinks (TiDB, Kafka, S3)
```

**Components:**

1. **JobManager**: Job coordination, checkpointing, recovery
2. **TaskManager**: Execute operators, manage state
3. **State Backend**: Store operator state (RocksDB for large state)
4. **Checkpointing**: Periodic snapshots for fault tolerance

**Execution Model:**

1. **Continuous Processing**: Data flows through operators continuously
2. **Event Time**: Processing based on event timestamps (not processing time)
3. **Watermarks**: Handle out-of-order events
4. **State Management**: Operators maintain state (aggregations, joins)
5. **Fault Tolerance**: Exactly-once via distributed snapshots

### Key Features

- **True Streaming**: Continuous processing (not micro-batch)
- **Event Time Processing**: Handle late and out-of-order events
- **Stateful Computations**: Windows, aggregations, joins
- **Exactly-Once Semantics**: End-to-end consistency guarantees
- **Low Latency**: Milliseconds to seconds
- **High Throughput**: Millions of events/sec

### APIs

**DataStream API:**
```java
StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

// Read from Kafka
DataStream<Transaction> transactions = env
    .addSource(new FlinkKafkaConsumer<>(
        "transactions",
        new TransactionSchema(),
        kafkaProps
    ));

// Transform
DataStream<Enriched> enriched = transactions
    .map(new EnrichmentFunction())
    .filter(txn -> txn.status.equals("SUCCESS"));

// Window aggregation
DataStream<AggregateResult> aggregated = enriched
    .keyBy(txn -> txn.merchantId)
    .window(TumblingEventTimeWindows.of(Time.minutes(5)))
    .aggregate(new SumAggregator());

// Write to TiDB
aggregated.addSink(new JdbcSink<>(
    "INSERT INTO merchant_stats VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE ...",
    (ps, result) -> { /* bind parameters */ },
    jdbcOptions
));

env.execute("Real-time Aggregation");
```

**Table API / SQL:**
```sql
CREATE TABLE kafka_transactions (
    transaction_id BIGINT,
    merchant_id BIGINT,
    amount DECIMAL(18, 2),
    created_at TIMESTAMP(3),
    WATERMARK FOR created_at AS created_at - INTERVAL '5' SECOND
) WITH (
    'connector' = 'kafka',
    'topic' = 'transactions',
    'properties.bootstrap.servers' = 'kafka:9092',
    'format' = 'json'
);

CREATE TABLE tidb_stats (
    merchant_id BIGINT,
    window_start TIMESTAMP(3),
    total_amount DECIMAL(18, 2),
    txn_count BIGINT,
    PRIMARY KEY (merchant_id, window_start) NOT ENFORCED
) WITH (
    'connector' = 'jdbc',
    'url' = 'jdbc:mysql://tidb:4000/analytics',
    'table-name' = 'merchant_stats'
);

INSERT INTO tidb_stats
SELECT
    merchant_id,
    TUMBLE_START(created_at, INTERVAL '5' MINUTE) as window_start,
    SUM(amount) as total_amount,
    COUNT(*) as txn_count
FROM kafka_transactions
WHERE status = 'SUCCESS'
GROUP BY merchant_id, TUMBLE(created_at, INTERVAL '5' MINUTE);
```

### Use Cases

**Ideal:**
- **CDC pipelines**: Replicate database changes to downstream systems
- **Real-time ETL**: Transform and enrich streaming data
- **Real-time aggregations**: Pre-compute for OLAP/dashboards
- **Event-driven applications**: React to events in real-time
- **Real-time alerting**: Detect anomalies, fraud

**Not Ideal:**
- **Batch processing**: Use Spark (better throughput for batch)
- **Ad-hoc queries**: Use Trino (Flink is for continuous processing)
- **Simple CDC**: Use managed CDC tools (Debezium)

### Event Time & Watermarks

**Event Time:**
```java
// Assign timestamps and watermarks
DataStream<Transaction> withTimestamps = transactions
    .assignTimestampsAndWatermarks(
        WatermarkStrategy
            .<Transaction>forBoundedOutOfOrderness(Duration.ofSeconds(5))
            .withTimestampAssigner((txn, timestamp) -> txn.createdAt)
    );
```

**Watermark**: "All events before timestamp T have arrived"
- Allows system to close windows and emit results
- Handles late events (within bounded delay)

### Checkpointing

**Configuration:**
```java
env.enableCheckpointing(60000); // Checkpoint every 60 seconds
env.getCheckpointConfig().setCheckpointingMode(CheckpointingMode.EXACTLY_ONCE);
env.getCheckpointConfig().setMinPauseBetweenCheckpoints(30000);
env.getCheckpointConfig().setCheckpointTimeout(600000);
env.setStateBackend(new RocksDBStateBackend("s3://checkpoints/"));
```

**Fault Tolerance:**
1. Checkpoint triggered every N seconds
2. Snapshot of operator state and Kafka offsets
3. On failure, restore from latest checkpoint
4. Replay from checkpoint (exactly-once)

### Monitoring

**Key Metrics:**
- **Lag**: Offset lag per Kafka partition
- **Throughput**: Records/sec processed
- **Backpressure**: Downstream operators slow (buffer full)
- **Checkpoint Duration**: Time to complete checkpoint
- **State Size**: Size of operator state

**Flink UI:**
```
http://<jobmanager-host>:8081

Tabs:
- Jobs: Running and completed jobs
- Task Metrics: Per-operator metrics
- Checkpoints: Checkpoint history, duration
- Backpressure: Identify bottlenecks
```

### Cost Model

**Kubernetes:**
```
Cost = Node cost × Job duration (continuous)

Example (long-running job):
- JobManager (1x c5.xlarge): $0.17/hour = $122/month
- TaskManagers (3x c5.2xlarge): $0.34/hour × 3 = $735/month
Total: ~$850/month (24/7)
```

### Best Practices

1. **Watermarks**: Configure bounded out-of-order delay
2. **Checkpointing**: Enable for fault tolerance (RocksDB for large state)
3. **Parallelism**: Match Kafka partition count
4. **Backpressure**: Monitor and tune buffer sizes
5. **State TTL**: Configure state expiration (prevent unbounded growth)
6. **Kafka offsets**: Commit offsets to Kafka (for observability)

---

## Compute Engine Comparison

| Feature | Spark | Trino | Flink |
|---------|-------|-------|-------|
| **Execution** | Batch (micro-batch streaming) | Interactive SQL | Continuous streaming |
| **Latency** | Minutes-Hours | Seconds-Minutes | Milliseconds-Seconds |
| **Fault Tolerance** | Lineage (batch), checkpoints (streaming) | None (query fails) | Checkpoints (exactly-once) |
| **Use Case** | ETL, ML | Ad-hoc queries, BI | Real-time pipelines |
| **State Management** | Limited (caching) | None | Rich (windows, joins) |
| **Cost Model** | Ephemeral (per-job) | Reserved (24/7) | Long-running (24/7) |

---

## Choosing the Right Compute Engine

**Decision Factors:**

1. **Latency Requirements**: Real-time vs batch?
2. **Workload Type**: Queries vs transformations vs streaming?
3. **Fault Tolerance**: Can queries be retried, or need exactly-once?
4. **Cost**: Ephemeral vs long-running?

**Examples:**

*Daily ETL to load data into Iceberg:*
- **Spark** (batch, high throughput, ephemeral)

*Analyst exploring historical data:*
- **Trino** (interactive SQL, ad-hoc queries)

*Real-time CDC from Aurora to TiDB:*
- **Flink** (continuous streaming, exactly-once)

*Pre-aggregate data for dashboard (real-time):*
- **Flink** (streaming aggregation) or **Spark Streaming** (micro-batch)

*Cross-system data validation:*
- **Trino** (federation, join Iceberg + MySQL)

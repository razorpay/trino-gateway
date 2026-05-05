# Flink Job Optimization Patterns

Common optimization patterns and anti-patterns for Apache Flink jobs, with specific focus on the jobspec-based runtime.

## Table of Contents

1. [DAG Wiring Issues](#dag-wiring-issues)
2. [Parallelism & Backpressure](#parallelism--backpressure)
3. [Data Skew](#data-skew)
4. [Checkpoint Configuration](#checkpoint-configuration)
5. [Memory Management](#memory-management)
6. [Window Aggregation Optimization](#window-aggregation-optimization)
7. [Operator Chaining](#operator-chaining)

---

## DAG Wiring Issues

### Dead Output Streams

**Problem:** An operator produces an outputStream that is never consumed by any downstream operator or sink.

**Example (YAML):**
```yaml
operators:
  - name: "filter-1"
    type: "FILTER"
    inputStreams: ["kafka-source"]
    outputStream: "filtered-stream"  # ❌ Nothing consumes this

  - name: "filter-2"
    type: "FILTER"
    inputStreams: ["kafka-source"]  # ❌ Ignores filtered-stream
    outputStream: "final-output"
```

**Fix:**
- Remove unused operator, OR
- Wire it to downstream: change `inputStreams: ["filtered-stream"]`

**Detection in Code:**
- Search `Main.java` for the `outputStream` name
- If never referenced in any `inputStreams`, it's dead code

---

### Missing Input Streams

**Problem:** An operator references an inputStream that doesn't exist (no source or operator produces it).

**Example (YAML):**
```yaml
operators:
  - name: "aggregator"
    type: "WINDOW_AGGREGATOR"
    inputStreams: ["non-existent-stream"]  # ❌ Never produced
    outputStream: "aggregated"
```

**Fix:**
- Correct the inputStream name to match an existing outputStream or source name
- Add the missing upstream operator/source

**Detection in Code:**
- `buildOperator()` throws `IllegalArgumentException` if stream not found in `streams` map
- Check logs for "Input stream 'X' not found"

---

## Parallelism & Backpressure

### Low Parallelism for Heavy Operators

**Problem:** Expensive operators (window aggregation, joins, CEP) run with parallelism=1, causing bottlenecks.

**Example (YAML):**
```yaml
parallelism: 1  # ❌ Default for all operators

operators:
  - name: "window-aggregator"
    type: "WINDOW_AGGREGATOR"
    inputStreams: ["filtered-payments"]
    outputStream: "aggregated"
    # No explicit parallelism, uses default=1
```

**Fix:**
```yaml
parallelism: 4  # Better default

operators:
  - name: "window-aggregator"
    type: "WINDOW_AGGREGATOR"
    inputStreams: ["filtered-payments"]
    outputStream: "aggregated"
    parallelism: 8  # ✓ Override for heavy operator
```

**Best Practices:**
- Window aggregators: parallelism = 4-16 (depending on data volume)
- Rule evaluators: parallelism = 2-8
- Filters: can stay at 1-2 if lightweight
- RCA analyzers: parallelism = 4-8 (state-heavy)

**Detection:**
- Check Flink UI for backpressure on specific vertices
- Check vertex parallelism in job graph
- Use metrics: `backPressuredTimeMsPerSecond > 500ms`

---

## Data Skew

### Unbalanced KeyBy Operations

**Problem:** Data is unevenly distributed across parallel instances due to skewed keys.

**Example:**
```yaml
operators:
  - name: "window-aggregator"
    type: "WINDOW_AGGREGATOR"
    config:
      keyField: "merchant_id"  # ⚠ Some merchants have 1000x more events
```

**Symptoms:**
- Some subtasks process much more data than others
- High checkpoint duration for specific subtasks
- Backpressure on some subtasks, idle on others

**Detection:**
- Check per-subtask metrics: `/jobs/:jobid/vertices/:vertexid/subtasks/metrics`
- Compare `numRecordsIn` across subtasks
- Large variance (max/min > 10x) indicates skew

**Fix Options:**

1. **Add salt to key:**
```java
// In code: Add random suffix to distribute hot keys
String saltedKey = merchantId + "-" + (hash(merchantId) % 10);
```

2. **Two-phase aggregation:**
```yaml
# Pre-aggregate with salted key, then final aggregate by original key
operators:
  - name: "pre-aggregator"
    keyField: "salted_merchant_id"  # Distribute load
  - name: "final-aggregator"
    keyField: "merchant_id"  # Accurate result
```

3. **Increase parallelism** (partial mitigation):
- Higher parallelism spreads skew across more instances
- Doesn't fix root cause but reduces impact

---

## Checkpoint Configuration

### Checkpoint Interval Too Frequent

**Problem:** Checkpoints run too often, impacting throughput.

**Example (YAML):**
```yaml
checkpointing:
  enabled: true
  interval: 5000  # ❌ Every 5 seconds - too frequent
```

**Impact:**
- High checkpoint overhead
- Reduced throughput
- Increased I/O to checkpoint storage

**Fix:**
```yaml
checkpointing:
  enabled: true
  interval: 60000  # ✓ Every 60 seconds for most jobs
  # For low-latency jobs: 30000 (30s)
  # For batch-like jobs: 300000 (5 min)
```

**Best Practices:**
- Standard jobs: 30-60 seconds
- Low-latency: 15-30 seconds
- Large state: 2-5 minutes
- Rule of thumb: checkpoint duration should be < 10% of interval

---

### Checkpoint Timeout Too Short

**Problem:** Checkpoints fail due to insufficient timeout.

**Example (YAML):**
```yaml
checkpointing:
  interval: 60000
  checkpointTimeout: 60000  # ❌ Same as interval
```

**Fix:**
```yaml
checkpointing:
  interval: 60000
  checkpointTimeout: 300000  # ✓ 5x the interval
```

**Best Practices:**
- Timeout should be 3-5x the interval
- For large state (GB+): 10+ minutes
- Monitor checkpoint duration in Flink UI

---

## Memory Management

### Insufficient Memory for State

**Problem:** RocksDB state backend runs out of memory, causing OOM or slow performance.

**Example (YAML):**
```yaml
resources:
  memory: "2gb"  # ❌ Too low for large state
  managedMemory: "512mb"  # ❌ Too low for RocksDB
```

**Fix:**
```yaml
resources:
  memory: "8gb"  # ✓ Total task memory
  managedMemory: "2gb"  # ✓ 25-40% for RocksDB

state:
  backend: "ROCKSDB"
  config:
    rocksdb:
      state.backend.rocksdb.block.cache-size: "512mb"
```

**Best Practices:**
- Total memory: 4-16GB per task slot
- Managed memory: 25-40% of total for RocksDB jobs
- Monitor RocksDB metrics: cache hit rate, compaction time
- Use incremental checkpointing for large state

---

### Memory Settings Not Aligned with Cluster

**Problem:** Job requests more memory than available per task manager.

**Example:**
```yaml
resources:
  memory: "16gb"  # ❌ But cluster has only 8GB per TM
```

**Fix:**
- Check cluster capacity: `GET /taskmanagers`
- Align memory requests with available resources
- Or provision larger task managers

---

## Window Aggregation Optimization

### Too Many Aggregation Expressions

**Problem:** Window aggregator has 50+ aggregation expressions, slowing down processing.

**Example (YAML):**
```yaml
operators:
  - name: "window-aggregator"
    type: "WINDOW_AGGREGATOR"
    config:
      aggregations:
        total_payment_upi_intent: "COUNT(...)"
        total_payment_upi_collect: "COUNT(...)"
        # ... 50 more similar expressions ❌
```

**Impact:**
- Slow window processing
- High CPU usage
- Complex state management

**Fix Options:**

1. **Split into multiple operators:**
```yaml
operators:
  - name: "upi-aggregator"
    type: "WINDOW_AGGREGATOR"
    config:
      aggregations:
        # Only UPI metrics

  - name: "card-aggregator"
    type: "WINDOW_AGGREGATOR"
    config:
      aggregations:
        # Only card metrics
```

2. **Simplify expressions:**
- Pre-compute complex fields before aggregation
- Use simpler COUNT/SUM instead of complex CASE WHEN

3. **Reduce window size:**
- Smaller windows = less data per aggregation cycle
- Use sliding windows only when necessary

---

### Repeated CASE WHEN Conditions

**Problem:** Many aggregations repeat the same CASE WHEN logic.

**Example (YAML):**
```yaml
aggregations:
  total_upi: "COUNT(DISTINCT CASE WHEN method_type = 'UPI Intent' THEN id END)"
  success_upi: "COUNT(DISTINCT CASE WHEN method_type = 'UPI Intent' THEN id END)"
  # ❌ Repeated condition
```

**Fix:**
- Pre-filter or pre-compute common conditions
- Use ProcessWindowFunction for complex logic instead of SQL-like expressions

---

## Operator Chaining

### Excessive Operator Chain Breaks

**Problem:** Too many operators with explicit UID/name breaks chaining, reducing performance.

**Example (Java):**
```java
stream
  .filter(...)
  .name("filter-1").uid("filter-1")  // ❌ Breaks chain
  .map(...)
  .name("map-1").uid("map-1")  // ❌ Breaks chain
```

**Impact:**
- More network shuffles
- Higher serialization overhead
- Reduced throughput

**Fix:**
```java
stream
  .filter(...)
  .map(...)
  .name("filter-map-chain")  // ✓ Single name for chain
```

**Best Practices:**
- Let Flink auto-chain compatible operators
- Only break chains when necessary (e.g., before shuffle)
- Use `.disableChaining()` explicitly if needed

---

### Forced Rebalance Without Cause

**Problem:** Unnecessary `rebalance()` calls cause data shuffling.

**Example (Java):**
```java
stream
  .filter(...)
  .rebalance()  // ❌ Unnecessary shuffle
  .map(...)
```

**Fix:**
```java
stream
  .filter(...)
  .map(...)  // ✓ No shuffle needed
```

**When to use rebalance():**
- After data skew to redistribute evenly
- Before expensive operations to balance load
- NOT needed after simple transformations (filter, map)

---

## Summary Checklist

Before deploying a Flink job, verify:

- [ ] No dead output streams (all outputs consumed)
- [ ] No missing input streams (all inputs exist)
- [ ] Parallelism > 1 for heavy operators (window, aggregation, join)
- [ ] Checkpoint interval: 30-60s for most jobs
- [ ] Checkpoint timeout: 3-5x interval
- [ ] Memory: 4GB+ total, 25-40% managed for RocksDB
- [ ] Window aggregations: < 30 expressions per operator
- [ ] Data skew handled via salting or two-phase aggregation
- [ ] Operators properly chained (no unnecessary breaks)
- [ ] No unnecessary rebalance() calls

# Understanding Trino Query Plans

## Getting Query Plans

### EXPLAIN
Shows the logical query plan without executing:

```sql
EXPLAIN
SELECT * FROM table WHERE condition;
```

### EXPLAIN ANALYZE
Shows actual execution statistics (runs the query):

```sql
EXPLAIN ANALYZE
SELECT * FROM table WHERE condition LIMIT 1000;  -- Use LIMIT for safety
```

### EXPLAIN (TYPE DISTRIBUTED)
Shows physical execution plan across cluster:

```sql
EXPLAIN (TYPE DISTRIBUTED)
SELECT * FROM table WHERE condition;
```

## Query Plan Structure

Trino query plans are tree structures with these common operators:

### Data Source Operators

#### TableScan
Reads data from a table.

```
- TableScan[table = hive:schema.table_name, ...]
    Layout: schema.table_name
    Estimates: {rows: 1000000, cpu: 1000.00, memory: 0.00, network: 0.00}
```

**What to look for**:
- Which table/partition is being scanned
- Estimated row count (indicates partition pruning effectiveness)
- Predicates pushed down to scan level

**Cost impact**: Low if partition-pruned, High if full table scan

#### Filter
Applies WHERE conditions not pushed to TableScan.

```
- Filter[condition]
    Estimates: {rows: 100000, cpu: 500.00, memory: 0.00}
```

**What to look for**:
- Conditions that couldn't be pushed to storage layer
- Selectivity (output rows / input rows)

**Cost impact**: Low, but indicates missed pushdown opportunity

### Join Operators

#### InnerJoin
Combines rows from two inputs matching join condition.

```
- InnerJoin[criteria = ("left.id" = "right.id")]
    Estimates: {rows: 500000, cpu: 2000.00, memory: 1000.00}
    Distribution: PARTITIONED
```

**What to look for**:
- Join distribution type (PARTITIONED, REPLICATED, BROADCAST)
- Estimated output size vs input sizes
- Join criteria (equality vs inequality)

**Cost impact**: Medium to High depending on data size

**Distribution types**:
- **PARTITIONED**: Both sides partitioned by join key (good for large-large joins)
- **BROADCAST**: Small table sent to all nodes (good for small-large joins)
- **REPLICATED**: Table replicated to all nodes (automatic for small tables)

#### CrossJoin
Cartesian product (usually a mistake).

```
- CrossJoin
    Estimates: {rows: 1000000000, cpu: 50000.00, memory: 10000.00}
```

**What to look for**:
- Presence of CrossJoin (red flag!)
- Massive row estimate

**Cost impact**: CRITICAL - Usually indicates missing join condition

#### LeftJoin, RightJoin, FullJoin
Outer joins preserving rows from one or both sides.

**Cost impact**: Similar to InnerJoin but may produce more rows

### Aggregation Operators

#### Aggregate
Performs GROUP BY and aggregation functions.

```
- Aggregate[type = FINAL, keys = [user_id], aggregates = [count(*), sum(amount)]]
    Estimates: {rows: 10000, cpu: 1500.00, memory: 500.00}
```

**What to look for**:
- Type: PARTIAL vs FINAL (distributed aggregation)
- Number of grouping keys
- Aggregate functions (COUNT, SUM, AVG, etc.)

**Cost impact**: Medium (higher for many groups or COUNT DISTINCT)

**Types**:
- **PARTIAL**: First phase aggregation on each node
- **FINAL**: Combines partial results
- **SINGLE**: Entire aggregation on one node (can be slow)

### Sort and Window Operators

#### Sort
Orders results (ORDER BY).

```
- Sort[orderBy = [date DESC, amount ASC]]
    Estimates: {rows: 1000000, cpu: 5000.00, memory: 2000.00}
```

**What to look for**:
- Whether LIMIT is present (sorts early-stop if so)
- Size of data being sorted

**Cost impact**: Medium to High (especially without LIMIT)

#### Window
Window functions (ROW_NUMBER, RANK, etc.).

```
- Window[partition by user_id, orderBy = [date], function = row_number()]
    Estimates: {rows: 1000000, cpu: 3000.00, memory: 1500.00}
```

**What to look for**:
- PARTITION BY clause (reduces per-partition size)
- ORDER BY requirements

**Cost impact**: Medium to High

### Output Operators

#### Project
Selects and transforms columns.

```
- Project[expressions = [column1, column2, column1 + column2 as total]]
```

**Cost impact**: Low (cheap operation)

#### Limit
Stops after N rows.

```
- Limit[count = 1000]
```

**Cost impact**: Very Low (optimization)

#### Output
Final output stage.

```
- Output[columnNames = [col1, col2]]
```

**Cost impact**: Minimal

## Reading Cost Estimates

### Estimate Components

```
Estimates: {rows: 1000000, cpu: 5000.00, memory: 2000.00, network: 1000.00}
```

- **rows**: Estimated number of rows processed
- **cpu**: Estimated CPU cost units
- **memory**: Estimated memory cost units
- **network**: Estimated network transfer cost units

### Interpreting Estimates

**Row estimates**:
- Compare estimates at different stages
- Large differences indicate expensive operations
- Massive row counts (billions) suggest cross joins or fan-out

**CPU estimates**:
- Higher values = more processing required
- Compare relative costs of different operators

**Memory estimates**:
- Critical for aggregations, sorts, joins
- Very high values may cause out-of-memory errors

## Common Patterns

### Efficient Pattern: Partition Pruning

```
- TableScan[table = events, filterPredicate = (dt >= '2024-01-01')]
    Estimates: {rows: 1000000}  -- Only scans needed partitions
```

### Inefficient Pattern: No Partition Pruning

```
- TableScan[table = events]
    Estimates: {rows: 100000000}  -- Scans entire table
  - Filter[dt >= '2024-01-01']
      Estimates: {rows: 1000000}  -- Filters after scan (waste)
```

### Efficient Pattern: Broadcast Join

```
- InnerJoin[Distribution: BROADCAST]
  - TableScan[large_table] (1M rows)
  - TableScan[small_dimension] (100 rows, broadcasted)
```

### Inefficient Pattern: Large Broadcast

```
- InnerJoin[Distribution: BROADCAST]
  - TableScan[large_table] (1M rows)
  - TableScan[another_large_table] (1M rows, broadcasted)  -- Bad!
```

### Efficient Pattern: Partial Aggregation

```
- Aggregate[type = FINAL]
  - Exchange[GATHER]
    - Aggregate[type = PARTIAL]  -- Pre-aggregate on each node
      - TableScan[...]
```

### Inefficient Pattern: Single Aggregation

```
- Aggregate[type = SINGLE]  -- Everything aggregated on one node
  - Exchange[GATHER]
    - TableScan[...]
```

## Optimization Indicators

### Good Signs ✓
- Low row estimates relative to table size (partition pruning working)
- PARTITIONED distribution for large joins
- BROADCAST distribution for small-large joins
- PARTIAL aggregations before FINAL
- Filters pushed to TableScan level
- Limit operators present

### Warning Signs ⚠
- Row estimates growing rapidly through plan
- Large broadcasts (>100MB)
- SINGLE aggregation type
- Filters after TableScan instead of in TableScan
- Many exchange operations
- Sort without Limit

### Critical Issues ✗
- CrossJoin operators
- Billions of estimated rows
- Multiple terabytes of estimated memory
- No partition predicates on large tables
- Broadcast of large tables
- Skewed data distribution

## Using EXPLAIN ANALYZE

EXPLAIN ANALYZE provides actual runtime statistics:

```
Fragment 0 [SINGLE]
    CPU: 25.50s, Scheduled: 45.20s, Input: 1000000 rows (100MB), Output: 100 rows
    - Aggregate[FINAL]
        CPU: 20.00s, Input: 1000000 rows, Output: 100 rows
```

**Compare actual vs estimated**:
- Large differences indicate poor statistics
- Actual >> Estimated: Query will be slower than planned
- Actual << Estimated: Statistics need updating

**CPU vs Scheduled time**:
- CPU: Actual processing time
- Scheduled: Wall-clock time (includes waiting)
- Large difference indicates resource contention

## Query Plan Checklist

Before running an expensive query:

- [ ] EXPLAIN shows reasonable row estimates
- [ ] No CrossJoin operators present
- [ ] Partition filters in TableScan predicates
- [ ] Large joins use PARTITIONED distribution
- [ ] Small-large joins use BROADCAST distribution
- [ ] Aggregations use PARTIAL/FINAL pattern
- [ ] No excessive network exchange
- [ ] Memory estimates reasonable (<10GB per node)
- [ ] LIMIT present for testing

## Debugging Slow Queries

1. **Get the plan**: `EXPLAIN ANALYZE` (with LIMIT for safety)
2. **Find expensive operators**: Look for high CPU/memory
3. **Check row estimates**: Massive growth indicates problem
4. **Examine distributions**: Inefficient join distributions
5. **Verify partition pruning**: Check TableScan predicates
6. **Look for anti-patterns**: CrossJoin, SINGLE aggregations
7. **Compare actual vs estimated**: Poor statistics?

## Example Analysis

```sql
EXPLAIN ANALYZE
SELECT
    u.name,
    COUNT(DISTINCT o.order_id) as order_count
FROM users u
JOIN orders o ON u.user_id = o.user_id
WHERE o.dt >= '2024-01-01'
GROUP BY u.name
LIMIT 100;
```

**What to look for**:
1. Is `dt >= '2024-01-01'` in the orders TableScan? (partition pruning)
2. How many rows estimated for orders TableScan?
3. What distribution for the join? (should be PARTITIONED or BROADCAST)
4. Is COUNT(DISTINCT) expensive? (might suggest approx_distinct)
5. Does the Limit help? (should limit output early)

**Ideal plan characteristics**:
- Orders TableScan shows partition filter
- Join uses appropriate distribution
- Aggregate shows PARTIAL/FINAL pattern
- Output limited to 100 rows

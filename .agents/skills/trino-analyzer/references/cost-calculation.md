# Trino Query Cost Calculation

## Cost Components

Trino query cost is calculated based on several resource consumption metrics:

### 1. CPU Time
**Metric**: `cpu_time` (seconds)
**What it measures**: Actual CPU processing time across all workers
**Cost factor**: Primary driver of compute costs

```
CPU Cost = cpu_time_seconds * cpu_rate_per_second
```

### 2. Memory Usage
**Metric**: `peak_mem` (bytes)
**What it measures**: Peak memory usage during query execution
**Cost factor**: Memory allocation costs

```
Memory Cost = peak_mem_gb * duration_seconds * memory_rate_per_gb_second
```

### 3. Data Scanned (S3 Operations)
**Metric**: `input_size` (bytes)
**What it measures**: Amount of data read from S3
**Cost factor**: S3 GET request costs + data transfer

```
S3 Cost = (input_size_gb * s3_data_rate) + (num_files * s3_request_rate)
```

### 4. Data Written
**Metric**: `physical_written_size` (bytes)
**What it measures**: Amount of data written to S3
**Cost factor**: S3 PUT request costs + data transfer

```
Write Cost = (written_size_gb * s3_write_rate) + (num_files * s3_put_request_rate)
```

## Query Cost Formula

Based on the `trino_showback` table structure:

```python
def calculate_query_cost(metrics):
    """
    Calculate Trino query cost from metrics.

    Args:
        metrics: Dictionary with query metrics from trino_showback

    Returns:
        Total cost in USD
    """
    # Extract metrics
    cpu_time = metrics.get('cpu_time', 0)  # seconds
    peak_mem = metrics.get('peak_mem', 0)  # bytes
    input_size = metrics.get('input_size', 0)  # bytes
    execution_time = metrics.get('execution_time', 0)  # seconds
    written_size = metrics.get('physical_written_size', 0)  # bytes

    # Convert units
    peak_mem_gb = peak_mem / (1024**3)
    input_size_gb = input_size / (1024**3)
    written_size_gb = written_size / (1024**3)

    # Cost rates (example - adjust based on actual cluster costs)
    CPU_RATE = 0.01  # $ per CPU-second
    MEMORY_RATE = 0.001  # $ per GB-second
    S3_READ_RATE = 0.0004  # $ per GB read
    S3_WRITE_RATE = 0.0005  # $ per GB written

    # Calculate components
    cpu_cost = cpu_time * CPU_RATE
    memory_cost = peak_mem_gb * execution_time * MEMORY_RATE
    s3_read_cost = input_size_gb * S3_READ_RATE
    s3_write_cost = written_size_gb * S3_WRITE_RATE

    # Total cost
    total_cost = cpu_cost + memory_cost + s3_read_cost + s3_write_cost

    return {
        'total_cost': total_cost,
        'cpu_cost': cpu_cost,
        'memory_cost': memory_cost,
        's3_read_cost': s3_read_cost,
        's3_write_cost': s3_write_cost
    }
```

## Using Historical Data for Cost Prediction

The `trino_showback` table contains pre-calculated `query_cost` and `query_credits` fields:

```sql
SELECT
    query_cost,          -- Pre-calculated cost in USD
    query_credits,       -- Normalized credit units
    cpu_time,
    peak_mem,
    input_size,
    execution_time
FROM hive.dbt_prod_de_metrics.trino_showback
WHERE query_id = 'your_query_id';
```

### Building a Cost Prediction Model

Use historical data to train a regression model:

```python
import pandas as pd
from sklearn.linear_model import LinearRegression

# Query historical data
df = query_trino("""
    SELECT
        cpu_time,
        peak_mem,
        cumulative_user_memory,
        input_size,
        input_rows,
        physical_written_size,
        execution_time,
        query_cost
    FROM hive.dbt_prod_de_metrics.trino_showback
    WHERE state = 'FINISHED'
      AND query_cost IS NOT NULL
      AND dt >= date_add('day', -30, current_date)
    LIMIT 100000
""")

# Prepare features
X = df[[
    'cpu_time',
    'peak_mem',
    'input_size',
    'execution_time',
    'input_rows'
]]

y = df['query_cost']

# Train model
model = LinearRegression()
model.fit(X, y)

# Predict cost for new query
def predict_cost(query_features):
    return model.predict([query_features])[0]
```

## Query Plan Features to Cost Mapping

Estimated cost multipliers based on query plan features:

| Feature | Cost Multiplier | Reason |
|---------|-----------------|--------|
| Cross Join | 10-100x | Cartesian product of tables |
| Window Functions | 2-5x | Requires sorting and partitioning |
| Multiple Aggregations (>5) | 1.5-3x | Multiple passes over data |
| No Partition Filter | 10-50x | Scans entire table instead of partitions |
| ORDER BY without LIMIT | 2-4x | Sorts entire result set |
| COUNT(DISTINCT) | 2-3x | High memory usage |
| Subquery in SELECT | 5-20x | Executes for each row |
| Complex JOINs (>3 tables) | 1.5-3x per join | Intermediate result size growth |

## Cost Estimation from EXPLAIN Output

```python
def estimate_cost_from_plan(plan_features):
    """
    Estimate query cost from plan features before execution.
    """
    base_cost = 0.001  # Minimum cost

    # Apply multipliers
    cost = base_cost

    if plan_features['has_cross_join']:
        cost *= 50

    if plan_features['has_window_functions']:
        cost *= 3

    if plan_features['join_count'] > 0:
        cost *= (1 + 0.5 * plan_features['join_count'])

    if not plan_features['has_partition_filter']:
        cost *= 10

    if plan_features['aggregation_count'] > 5:
        cost *= 2

    return cost
```

## Cost Optimization ROI

Typical savings from common optimizations:

1. **Adding partition filters**: 70-95% cost reduction
2. **Removing SELECT ***: 20-50% cost reduction
3. **Using approx_distinct()**: 40-60% cost reduction
4. **Fixing cross joins**: 90%+ cost reduction
5. **Adding LIMIT during testing**: 90%+ cost reduction
6. **Predicate pushdown**: 30-60% cost reduction
7. **Removing subqueries in SELECT**: 50-80% cost reduction

## Monitoring and Alerts

Set up cost alerts based on thresholds:

```sql
-- Find queries exceeding cost threshold
SELECT
    query_id,
    user,
    query_cost,
    cpu_time,
    peak_mem,
    query
FROM hive.dbt_prod_de_metrics.trino_showback
WHERE dt = current_date
  AND query_cost > 10.0  -- Threshold: $10
ORDER BY query_cost DESC;
```

```sql
-- Daily cost by user
SELECT
    user,
    DATE(query_created_at) as date,
    COUNT(*) as query_count,
    SUM(query_cost) as total_cost,
    AVG(query_cost) as avg_cost
FROM hive.dbt_prod_de_metrics.trino_showback
WHERE dt >= date_add('day', -7, current_date)
GROUP BY user, DATE(query_created_at)
HAVING SUM(query_cost) > 100  -- Daily threshold: $100
ORDER BY total_cost DESC;
```

## References

- CPU time: Total processing time across all workers
- Peak memory: Maximum memory used at any point during execution
- Input size: Total bytes read from storage (S3)
- Execution time: Wall-clock time from start to finish
- Query credits: Normalized cost units (specific to your org)
- Query cost: Actual cost in USD (pre-calculated in trino_showback)

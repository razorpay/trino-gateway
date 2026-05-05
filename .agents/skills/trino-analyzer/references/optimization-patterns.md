# Trino Query Optimization Patterns

## Common Anti-Patterns

### 1. SELECT * Usage
**Problem**: Retrieves all columns even if only a few are needed.
**Impact**: Increases data transfer, memory usage, and processing time.
**Fix**: Select only required columns.

```sql
-- Bad
SELECT * FROM large_table WHERE dt = '2024-01-01';

-- Good
SELECT user_id, transaction_amount, created_at
FROM large_table
WHERE dt = '2024-01-01';
```

### 2. Missing Partition Filters
**Problem**: Scans entire table instead of specific partitions.
**Impact**: 10-100x more data scanned from S3.
**Fix**: Always filter on partition columns (dt, date, year, month).

```sql
-- Bad
SELECT * FROM events WHERE user_id = 12345;

-- Good
SELECT * FROM events
WHERE dt BETWEEN '2024-01-01' AND '2024-01-31'
  AND user_id = 12345;
```

### 3. Cross Joins
**Problem**: Creates cartesian product of tables.
**Impact**: Can increase result size from thousands to millions/billions of rows.
**Fix**: Add proper JOIN conditions.

```sql
-- Bad
SELECT * FROM table1 CROSS JOIN table2;

-- Good
SELECT * FROM table1
INNER JOIN table2 ON table1.id = table2.table1_id;
```

### 4. Subqueries in SELECT Clause
**Problem**: Executes subquery for each row in result.
**Impact**: N+1 query pattern causing extreme slowdown.
**Fix**: Move to JOIN or use window functions.

```sql
-- Bad
SELECT
    user_id,
    (SELECT COUNT(*) FROM orders WHERE user_id = u.user_id) as order_count
FROM users u;

-- Good
SELECT
    u.user_id,
    COUNT(o.order_id) as order_count
FROM users u
LEFT JOIN orders o ON u.user_id = o.user_id
GROUP BY u.user_id;
```

### 5. Multiple OR Conditions
**Problem**: Prevents efficient index/partition usage.
**Impact**: Full table scans instead of partition pruning.
**Fix**: Use IN clause or UNION.

```sql
-- Bad
WHERE status = 'pending' OR status = 'active' OR status = 'completed';

-- Good
WHERE status IN ('pending', 'active', 'completed');
```

### 6. COUNT(DISTINCT) on High Cardinality Columns
**Problem**: Requires maintaining full distinct set in memory.
**Impact**: High memory usage and slow performance.
**Fix**: Use approx_distinct() for approximate counts.

```sql
-- Bad (exact but slow)
SELECT COUNT(DISTINCT user_id) FROM large_table;

-- Good (2-3% error, 60% faster)
SELECT approx_distinct(user_id) FROM large_table;
```

## Optimization Techniques

### 1. Partition Pruning
Filter on partition columns to reduce data scanned.

```sql
-- Only scans data from specified date range
SELECT * FROM events
WHERE dt >= '2024-01-01' AND dt <= '2024-01-31';
```

### 2. Predicate Pushdown
Apply filters early to reduce data size before joins.

```sql
-- Good: Filter before join
WITH filtered_events AS (
    SELECT * FROM events
    WHERE dt = '2024-01-01' AND event_type = 'click'
)
SELECT e.*, u.name
FROM filtered_events e
JOIN users u ON e.user_id = u.id;
```

### 3. Join Order Optimization
Join smaller tables first, then larger tables.

```sql
-- Good: Start with smallest table
SELECT *
FROM small_dimension_table d
JOIN medium_table m ON d.id = m.dim_id
JOIN large_fact_table f ON m.id = f.medium_id;
```

### 4. Use CTEs for Readability and Reusability
Common Table Expressions (WITH clauses) improve clarity.

```sql
WITH daily_totals AS (
    SELECT dt, SUM(amount) as total
    FROM transactions
    WHERE dt >= '2024-01-01'
    GROUP BY dt
)
SELECT * FROM daily_totals WHERE total > 1000;
```

### 5. Approximate Functions
Use approximate aggregations for acceptable accuracy with better performance.

```sql
-- Approximate distinct count (2-3% error)
SELECT approx_distinct(user_id) FROM events;

-- Approximate percentiles
SELECT approx_percentile(response_time, 0.95) FROM requests;

-- Approximate set membership
SELECT approx_set(user_id) FROM events;
```

### 6. Limit During Development
Always use LIMIT when testing complex queries.

```sql
-- Test with small result set first
SELECT * FROM complex_query
LIMIT 100;
```

### 7. Materialize Intermediate Results
For complex multi-step queries, create temporary tables.

```sql
CREATE TABLE temp.intermediate_results AS
SELECT * FROM complex_calculation;

-- Then use intermediate results
SELECT * FROM temp.intermediate_results
WHERE condition;
```

### 8. Use EXPLAIN to Understand Query Plans
Always check query plan before running expensive queries.

```sql
EXPLAIN
SELECT * FROM large_table WHERE complex_condition;

-- Even better: EXPLAIN ANALYZE (actually runs the query)
EXPLAIN ANALYZE
SELECT * FROM table LIMIT 1000;  -- Use LIMIT for testing
```

## Cost Factors

### High Cost Operations (Avoid if Possible)
1. **Cross joins** - Cartesian products
2. **Window functions without PARTITION BY** - Process entire dataset
3. **ORDER BY without LIMIT** - Sort entire result
4. **DISTINCT** on high cardinality - Memory intensive
5. **Multiple aggregations** - Multiple passes over data

### Medium Cost Operations (Use Wisely)
1. **GROUP BY** - Requires aggregation
2. **Joins** - Depends on data size
3. **Window functions with PARTITION BY** - Scoped processing
4. **Subqueries** - Depends on complexity

### Low Cost Operations (Generally Safe)
1. **Filters on partitions** - Prunes data early
2. **Projections** (column selection) - Reduces data transfer
3. **LIMIT** - Stops early
4. **Simple predicates** - Filter efficiently

## Performance Checklist

Before running a query, verify:

- [ ] Partition filters are present (dt, date, etc.)
- [ ] Only necessary columns are selected (no SELECT *)
- [ ] No cross joins or unintentional cartesian products
- [ ] Filters are applied before joins
- [ ] Appropriate aggregations (exact vs approximate)
- [ ] LIMIT clause for testing
- [ ] Query plan reviewed with EXPLAIN

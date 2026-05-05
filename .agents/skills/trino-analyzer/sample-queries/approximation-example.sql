-- Example: Using Approximate Aggregations
-- Compare exact vs approximate functions for performance

-- EXPENSIVE: Exact distinct count
SELECT
    dt,
    COUNT(DISTINCT user_id) as exact_unique_users,
    COUNT(DISTINCT session_id) as exact_unique_sessions,
    COUNT(DISTINCT event_id) as exact_unique_events
FROM events
WHERE dt >= '2024-01-01' AND dt <= '2024-01-31'
GROUP BY dt;
-- Estimated Cost: $5-10

-- OPTIMIZED: Approximate distinct count (2-3% error)
SELECT
    dt,
    approx_distinct(user_id) as approx_unique_users,
    approx_distinct(session_id) as approx_unique_sessions,
    approx_distinct(event_id) as approx_unique_events
FROM events
WHERE dt >= '2024-01-01' AND dt <= '2024-01-31'
GROUP BY dt;
-- Estimated Cost: $2-4 (40-60% reduction)

-- Other approximate functions:
-- approx_percentile(value, 0.95) - 95th percentile
-- approx_set(column) - HyperLogLog set for set operations
-- approx_most_frequent(column, 10) - Top 10 most common values

-- Try with Claude:
-- "Compare the cost of exact vs approximate aggregations in this query"

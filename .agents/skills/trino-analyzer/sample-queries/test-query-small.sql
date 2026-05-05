-- Example: Safe Test Query
-- Always start with small queries during development

SELECT
    dt,
    event_type,
    COUNT(*) as event_count,
    COUNT(DISTINCT user_id) as unique_users
FROM events
WHERE dt = '2024-01-31'  -- Single day only
  AND event_type = 'click'
GROUP BY dt, event_type
LIMIT 100  -- Small result set for testing

-- Why this is a good test query:
-- 1. ✅ Single partition (dt = '2024-01-31')
-- 2. ✅ Additional filter (event_type)
-- 3. ✅ LIMIT to restrict results
-- 4. ✅ Simple aggregation for validation

-- Estimated Cost: $0.01-0.05

-- Try with Claude:
-- "Predict the cost of this test query"

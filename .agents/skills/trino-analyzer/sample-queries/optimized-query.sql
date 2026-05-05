-- Example: Optimized Version of expensive-query.sql
-- This shows the corrected version with all optimizations applied

SELECT
    e.event_id,
    e.event_type,
    e.event_timestamp,
    u.user_id,
    u.user_name,
    u.country
FROM events e
INNER JOIN users u ON e.user_id = u.id  -- Proper JOIN instead of CROSS JOIN
WHERE e.dt >= '2024-01-01' AND e.dt <= '2024-01-31'  -- Partition filter added
  AND e.event_type IN ('click', 'view', 'purchase')  -- Already optimal
  AND u.status = 'active'
  AND u.country IN ('US', 'UK', 'CA')  -- IN clause instead of multiple ORs
LIMIT 10000  -- Limit for safety during development

-- Optimizations Applied:
-- 1. ✅ SELECT specific columns only (not *)
-- 2. ✅ INNER JOIN instead of CROSS JOIN
-- 3. ✅ Partition filter added (dt range)
-- 4. ✅ IN clause for country filter
-- 5. ✅ LIMIT added for testing

-- Estimated Cost: $0.50-2.00 (85-95% reduction)

-- Try with Claude:
-- "Compare the cost of expensive-query.sql vs optimized-query.sql"

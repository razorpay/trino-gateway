-- Example: Expensive Query with Multiple Issues
-- This query demonstrates common anti-patterns that make queries expensive

SELECT *
FROM events e
CROSS JOIN users u
WHERE e.user_id = u.id  -- Should use INNER JOIN instead
  AND e.event_type IN ('click', 'view', 'purchase')
  AND u.status = 'active'
  AND (u.country = 'US' OR u.country = 'UK' OR u.country = 'CA')  -- Should use IN

-- Issues:
-- 1. SELECT * - retrieves all columns (unnecessary data)
-- 2. CROSS JOIN - creates cartesian product before filtering
-- 3. No partition filter - scans entire events table
-- 4. Multiple OR conditions - could use IN clause
-- 5. No LIMIT - returns all matching rows

-- Estimated Cost: $10-50 depending on table size

-- Try with Claude:
-- "Can you optimize this query and estimate the cost savings?"

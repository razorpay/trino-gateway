# Example 1: Top 10 Expensive Queries from Specific Cluster

## Scenario
You want to find the most expensive queries that ran on the `trino-tableau` cluster in the last 7 days to identify optimization opportunities.

## How to Ask Claude

Simply ask:
```
Give me top 10 expensive queries from trino-tableau cluster in last 7 days
```

Or be more specific:
```
Find the top 10 most expensive queries by CPU time from the trino-tableau
cluster in the last 7 days. Show me the query text, user, and resource usage.
```

## What Claude Will Do

Claude will:
1. Query the `iceberg.de_metrics.trino_query_analyzer_metrics` table
2. Filter by cluster_label = 'trino-tableau'
3. Filter by date range (last 7 days)
4. Order by CPU time (descending)
5. Return top 10 results with detailed analysis

## Sample Output

```
====================================================================================================
TOP 10 EXPENSIVE QUERIES - trino-tableau CLUSTER (LAST 7 DAYS)
====================================================================================================
Date Range: 2026-01-25 to 2026-02-01

📊 Summary:
   🖥️  Cluster: trino-tableau
   ⏱️  Combined CPU time: 142.3 hours
   💾 Combined data scanned: 8.4 TB
   📝 Total queries analyzed: 10
   ❌ Failed queries: 5 (50.0% failure rate)
   ⚠️  Wasted CPU time (failures): 64.2 hours

====================================================================================================

====================================================================================================
#1 - 20260131_082417_00234_abcde
====================================================================================================
👤 User: pushplata.s@razorpay.com
📅 Time: 2026-01-31 08:24:17
🏷️  Type: SELECT
🖥️  Cluster: trino-tableau
❌ Status: FAILED (USER_CANCELED)

📈 RESOURCES:
   ⏱️  CPU Time:        39.80 hours  (143,280s)
   💾 Memory:           245.67 GB
   📊 Data Scanned:     2,847.32 GB  (2.78 TB)
   📝 Rows:            8,234,567,890
   ⏰ Execution Time:   1,234 seconds

📝 Query Preview:
   SELECT
     o.id, o.name, concat('org_', cast(o.id as varchar)) as org_key,
     COUNT(DISTINCT u.id) as user_count,
     COUNT(DISTINCT t.id) as transaction_count
   FROM organizations o
   LEFT JOIN users u ON concat('org_', cast(o.id as varchar)) = u.org_key
   LEFT JOIN transactions t ON u.id = t.user_id
   WHERE o.merchant_id = 'ABC123'
   GROUP BY 1,2,3
   ... (truncated)

⚠️  ISSUES IDENTIFIED:
   ❌ Function on JOIN column: concat('org_', cast(o.id as varchar))
   ❌ Multiple COUNT(DISTINCT) - use approx_distinct()
   ❌ Multiple LEFT JOINs without filters
   ⚠️  No partition filter detected

💡 OPTIMIZATION POTENTIAL: 60-80% cost reduction

====================================================================================================
#2 - 20260130_143522_00456_fghij
====================================================================================================
👤 User: analytics.team@razorpay.com
📅 Time: 2026-01-30 14:35:22
🏷️  Type: SELECT
🖥️  Cluster: trino-tableau
✅ Status: FINISHED

📈 RESOURCES:
   ⏱️  CPU Time:        28.45 hours  (102,420s)
   💾 Memory:           189.23 GB
   📊 Data Scanned:     1,456.78 GB  (1.42 TB)
   📝 Rows:            5,678,901,234
   ⏰ Execution Time:   876 seconds

📝 Query Preview:
   SELECT date_trunc('day', created_at) as day,
          merchant_id,
          COUNT(*) as total_transactions,
          SUM(amount) as total_amount
   FROM transactions
   WHERE created_at >= current_date - interval '90' day
   GROUP BY 1, 2
   ORDER BY 1 DESC, 2
   ... (truncated)

⚠️  ISSUES IDENTIFIED:
   ⚠️  Large date range (90 days) - consider partition pruning
   ⚠️  Full table scan likely
   ✅ Simple aggregation - already optimized

💡 OPTIMIZATION POTENTIAL: 20-30% cost reduction

====================================================================================================
... (queries #3-#10 omitted for brevity)
====================================================================================================

====================================================================================================
📊 ANALYTICS & INSIGHTS
====================================================================================================

👥 Top Users by CPU Time:
   pushplata.s@razorpay.com           64.5 hours   (45.3%)
   analytics.team@razorpay.com        38.2 hours   (26.8%)
   data.viz@razorpay.com              23.1 hours   (16.2%)
   reporting@razorpay.com             16.5 hours   (11.6%)

📈 Query Type Breakdown:
   SELECT:      10 queries  (100%)
   INSERT:       0 queries
   DELETE:       0 queries

⚠️  Failure Analysis:
   Total failures: 5 out of 10 (50.0%)
   Wasted CPU: 64.2 hours
   Common reasons:
     - USER_CANCELED: 3 queries
     - EXCEEDED_TIME_LIMIT: 2 queries

🎯 Top Issues Found:
   1. Functions on JOIN columns (5 queries) - prevents index usage
   2. Missing partition filters (7 queries) - full table scans
   3. COUNT(DISTINCT) on large datasets (4 queries) - use approx_distinct()
   4. Multiple LEFT JOINs (6 queries) - creates large intermediate results

💡 OPTIMIZATION OPPORTUNITIES:
   If 30-50% optimization achieved on successful queries: 23.4 - 39.1 hours saved
   If failures prevented: 64.2 hours saved
   Total potential savings: 87.6 - 103.3 hours per week

✅ RECOMMENDED ACTIONS:
   1. Contact pushplata.s@razorpay.com about failed queries
      - 3 failed queries consuming 64.2 CPU-hours
      - Add partition filters and fix JOIN conditions

   2. Implement query guardrails:
      - Enforce partition filters on large tables
      - Set time limits for Tableau queries
      - Require LIMIT clause for development queries

   3. Create optimized versions of top 3 queries:
      - Replace COUNT(DISTINCT) with approx_distinct()
      - Pre-compute JOINs with functions
      - Add appropriate partition filters

   4. Review Tableau dashboard efficiency:
      - 50% failure rate indicates dashboards timing out
      - Consider pre-aggregated tables or materialized views

====================================================================================================
```

## Understanding the Results

### Key Metrics to Look At

1. **CPU Time**: Total compute time across all workers
   - High CPU time = expensive query
   - In multi-worker systems, CPU time can exceed wall-clock time

2. **Data Scanned**: Amount of data read from storage
   - High data scanned usually means missing partition filters
   - Ideally, should only scan necessary partitions

3. **Failure Rate**: Percentage of queries that failed
   - High failure rate wastes resources
   - Common causes: timeouts, user cancellations, memory limits

4. **User Analysis**: Which users are running expensive queries
   - Target user education and query review
   - May indicate dashboard issues or inefficient reports

### Red Flags to Watch For

- ❌ **Functions on JOIN columns**: `concat()`, `cast()` in JOIN conditions prevent optimization
- ❌ **No partition filters**: Queries without `created_date` filters scan entire tables
- ❌ **Multiple COUNT(DISTINCT)**: Very expensive, use `approx_distinct()` instead
- ❌ **High failure rate**: Indicates queries timing out or being canceled
- ❌ **SELECT ***: Reading unnecessary columns wastes I/O and memory

### Next Steps After Getting Results

1. **Prioritize by impact**: Focus on top 3 queries first (usually 40-60% of total cost)

2. **Analyze failed queries**: Failed queries waste resources and frustrate users
   ```
   Can you analyze query ID 20260131_082417_00234_abcde and suggest optimizations?
   ```

3. **Contact heavy users**: Work with users running expensive queries
   ```
   Show me all queries by pushplata.s@razorpay.com in the last 7 days
   ```

4. **Get query plan**: Understand execution strategy
   ```
   Can you analyze the query plan for this query and explain the bottlenecks?
   ```

5. **Compare alternatives**: Test optimized versions
   ```
   Compare the cost of my current query vs this optimized version
   ```

## Common Follow-up Questions

**Q: How do I optimize a specific query?**
```
Can you analyze query ID 20260131_082417_00234_abcde and provide an optimized version?
```

**Q: Why are these queries failing?**
```
Show me the failure reasons for queries by pushplata.s@razorpay.com on trino-tableau cluster
```

**Q: What's the cost trend over time?**
```
Show me the daily total CPU usage for trino-tableau cluster over the last 30 days
```

**Q: Which Tableau dashboards are most expensive?**
```
Group expensive queries by source/dashboard on trino-tableau cluster
```

## Tips for Best Results

1. **Be specific about the cluster**: Different clusters have different workloads
   - `trino-tableau` - BI/visualization queries
   - `trino-adhoc` - Ad-hoc analysis queries
   - `trino-scheduler` - Scheduled batch jobs

2. **Choose the right time range**:
   - Last 7 days: Recent trends and current issues
   - Last 30 days: Monthly patterns and recurring problems
   - Last 24 hours: Immediate troubleshooting

3. **Focus on actionable insights**:
   - Don't just collect data - act on it
   - Prioritize high-impact optimizations
   - Work with query authors to fix issues

4. **Monitor regularly**:
   - Weekly review of top expensive queries
   - Monthly cost analysis and trending
   - Set up alerts for unusual spikes

## Related Examples

- **Example 2**: Analyze a specific query for optimization opportunities
- **Example 3**: Compare query costs before and after optimization
- **Example 4**: Find queries missing partition filters
- **Example 5**: Identify users with highest query costs

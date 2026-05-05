# Example 3: Cluster Insights and Analytics

## Scenario
You want to understand query patterns, resource usage, and optimization opportunities across the querybook trino cluster over the last 7 days.

## How to Ask Claude

Ask for comprehensive insights:
```
Analyze last 7 days querybook trino cluster queries and give insights
```

Or be more specific:
```
Give me insights and analytics for the trino-querybook cluster for the last 7 days.
Include query patterns, top users, failure analysis, and optimization opportunities.
```

Other variations:
```
What's happening on the querybook cluster? Show me patterns and issues from last week.
```

```
Analyze trino-querybook cluster health - show me query trends, failures, and cost drivers
```

## Claude's Analysis Output

```
====================================================================================================
TRINO-QUERYBOOK CLUSTER ANALYSIS - LAST 7 DAYS
====================================================================================================
Date Range: 2026-01-25 to 2026-02-01
Cluster: trino-querybook
Analysis Time: 2026-02-01 14:30:00

====================================================================================================
📊 EXECUTIVE SUMMARY
====================================================================================================

Total Queries Analyzed: 12,847
├─ Successful: 9,234 (71.9%)
├─ Failed: 3,613 (28.1%)
└─ Still Running: 0

Resource Consumption:
├─ Total CPU Time: 3,847.2 hours (160.3 days)
├─ Total Data Scanned: 248.7 TB
├─ Average Query CPU: 18.0 minutes
└─ Peak Memory Usage: 2.4 TB (concurrent queries)

Cost Breakdown:
├─ Successful Queries: ~2,765 CPU-hours (71.9%)
├─ Failed Queries (wasted): ~1,082 CPU-hours (28.1%)
└─ Optimization Potential: 30-50% (~1,154 - 1,924 CPU-hours)

Top Issues:
1. High failure rate (28.1%) - 3,613 failed queries
2. Missing partition filters (67% of queries)
3. Expensive SELECT * patterns (45% of queries)
4. COUNT(DISTINCT) overuse (892 queries)

====================================================================================================
📈 QUERY VOLUME TRENDS
====================================================================================================

Daily Query Counts:
┌────────────┬──────────┬───────────┬─────────┬──────────────┐
│ Date       │ Total    │ Success   │ Failed  │ Failure Rate │
├────────────┼──────────┼───────────┼─────────┼──────────────┤
│ 2026-01-25 │ 1,834    │ 1,312     │ 522     │ 28.5%        │
│ 2026-01-26 │ 1,756    │ 1,245     │ 511     │ 29.1%        │
│ 2026-01-27 │ 2,012    │ 1,456     │ 556     │ 27.6%        │
│ 2026-01-28 │ 1,923    │ 1,389     │ 534     │ 27.8%        │
│ 2026-01-29 │ 1,845    │ 1,334     │ 511     │ 27.7%        │
│ 2026-01-30 │ 1,678    │ 1,201     │ 477     │ 28.4%        │
│ 2026-01-31 │ 1,799    │ 1,297     │ 502     │ 27.9%        │
└────────────┴──────────┴───────────┴─────────┴──────────────┘

📊 Observations:
   - Relatively stable query volume (~1,700-2,000 queries/day)
   - Consistent ~28% failure rate (concerning!)
   - Monday (Jan 27) had highest volume (2,012 queries)
   - Weekend (Jan 26) had lowest volume (1,756 queries)

====================================================================================================
👥 TOP USERS BY QUERY COUNT
====================================================================================================

┌────┬─────────────────────────────────┬──────────┬───────────┬─────────┬──────────┐
│ #  │ User                            │ Queries  │ Success   │ Failed  │ Fail %   │
├────┼─────────────────────────────────┼──────────┼───────────┼─────────┼──────────┤
│ 1  │ analytics.team@razorpay.com     │ 2,456    │ 1,987     │ 469     │ 19.1%    │
│ 2  │ data.science@razorpay.com       │ 1,834    │ 1,245     │ 589     │ 32.1%    │
│ 3  │ prasad.pawar@razorpay.com       │ 1,567    │ 823       │ 744     │ 47.5% ⚠️  │
│ 4  │ business.intel@razorpay.com     │ 1,234    │ 1,012     │ 222     │ 18.0%    │
│ 5  │ reporting@razorpay.com          │ 987      │ 856       │ 131     │ 13.3%    │
│ 6  │ adhoc.analysis@razorpay.com     │ 845      │ 567       │ 278     │ 32.9%    │
│ 7  │ finance.team@razorpay.com       │ 723      │ 623       │ 100     │ 13.8%    │
│ 8  │ product.analytics@razorpay.com  │ 656      │ 534       │ 122     │ 18.6%    │
│ 9  │ risk.analytics@razorpay.com     │ 589      │ 478       │ 111     │ 18.8%    │
│ 10 │ ops.team@razorpay.com           │ 456      │ 389       │ 67      │ 14.7%    │
└────┴─────────────────────────────────┴──────────┴───────────┴─────────┴──────────┘

🚨 Alert: prasad.pawar@razorpay.com has 47.5% failure rate!
   - 744 failed queries out of 1,567 total
   - Recommend user training or query review

====================================================================================================
⏱️  TOP USERS BY CPU TIME (COST)
====================================================================================================

┌────┬─────────────────────────────────┬─────────────┬──────────────┬─────────────┐
│ #  │ User                            │ CPU Hours   │ % of Total   │ Avg/Query   │
├────┼─────────────────────────────────┼─────────────┼──────────────┼─────────────┤
│ 1  │ data.science@razorpay.com       │ 1,234.5     │ 32.1%        │ 40.4 min    │
│ 2  │ prasad.pawar@razorpay.com       │ 892.3       │ 23.2%        │ 34.2 min ⚠️  │
│ 3  │ analytics.team@razorpay.com     │ 678.9       │ 17.6%        │ 16.6 min    │
│ 4  │ business.intel@razorpay.com     │ 345.6       │ 9.0%         │ 16.8 min    │
│ 5  │ adhoc.analysis@razorpay.com     │ 234.7       │ 6.1%         │ 16.7 min    │
│ 6  │ product.analytics@razorpay.com  │ 167.8       │ 4.4%         │ 15.3 min    │
│ 7  │ reporting@razorpay.com          │ 123.4       │ 3.2%         │ 7.5 min     │
│ 8  │ finance.team@razorpay.com       │ 89.2        │ 2.3%         │ 7.4 min     │
│ 9  │ risk.analytics@razorpay.com     │ 56.7        │ 1.5%         │ 5.8 min     │
│ 10 │ ops.team@razorpay.com           │ 24.1        │ 0.6%         │ 3.2 min     │
└────┴─────────────────────────────────┴─────────────┴──────────────┴─────────────┘

🎯 Cost Concentration:
   - Top 3 users: 2,805.7 CPU-hours (72.9% of total!)
   - Top 5 users: 3,385.9 CPU-hours (88.0% of total)
   - Long tail: 461.3 CPU-hours (12.0% across 100+ other users)

⚠️  Note: prasad.pawar has high cost + high failure rate
   - 892.3 CPU-hours total
   - 47.5% failure rate = ~424 CPU-hours WASTED
   - Urgent optimization needed

====================================================================================================
🔍 QUERY TYPE BREAKDOWN
====================================================================================================

By Statement Type:
┌─────────────────┬──────────┬──────────────┬────────────────┐
│ Query Type      │ Count    │ % of Total   │ Avg CPU Time   │
├─────────────────┼──────────┼──────────────┼────────────────┤
│ SELECT          │ 11,234   │ 87.4%        │ 16.8 min       │
│ INSERT          │ 892      │ 6.9%         │ 45.2 min       │
│ CREATE TABLE    │ 456      │ 3.5%         │ 34.5 min       │
│ DELETE          │ 178      │ 1.4%         │ 23.1 min       │
│ UPDATE          │ 67       │ 0.5%         │ 28.7 min       │
│ DROP            │ 20       │ 0.2%         │ 1.2 min        │
└─────────────────┴──────────┴──────────────┴────────────────┘

By Complexity:
┌──────────────────────────────┬──────────┬──────────────┐
│ Complexity Level             │ Count    │ % of Total   │
├──────────────────────────────┼──────────┼──────────────┤
│ Simple (1 table, no JOINs)   │ 4,567    │ 35.5%        │
│ Medium (2-3 tables/JOINs)    │ 5,234    │ 40.7%        │
│ Complex (4+ tables/JOINs)    │ 2,345    │ 18.2%        │
│ Very Complex (CTEs, Window)  │ 701      │ 5.5%         │
└──────────────────────────────┴──────────┴──────────────┘

====================================================================================================
❌ FAILURE ANALYSIS
====================================================================================================

Total Failed Queries: 3,613 (28.1%)
Wasted CPU Time: 1,082.3 hours

Failure Reasons:
┌─────────────────────────────────────┬──────────┬──────────────┬─────────────┐
│ Failure Reason                      │ Count    │ % of Failures│ CPU Wasted  │
├─────────────────────────────────────┼──────────┼──────────────┼─────────────┤
│ EXCEEDED_TIME_LIMIT                 │ 1,234    │ 34.2%        │ 456.7 hrs   │
│ USER_CANCELED                       │ 987      │ 27.3%        │ 234.5 hrs   │
│ EXCEEDED_MEMORY_LIMIT               │ 678      │ 18.8%        │ 189.2 hrs   │
│ QUERY_REJECTED (missing filter)     │ 445      │ 12.3%        │ 112.3 hrs   │
│ SYNTAX_ERROR                        │ 156      │ 4.3%         │ 23.4 hrs    │
│ PERMISSION_DENIED                   │ 78       │ 2.2%         │ 12.1 hrs    │
│ RESOURCE_EXHAUSTED                  │ 35       │ 1.0%         │ 54.1 hrs    │
└─────────────────────────────────────┴──────────┴──────────────┴─────────────┘

💡 Key Insights:

1. **EXCEEDED_TIME_LIMIT (34.2%)**: Queries running too long
   - Common cause: Missing partition filters, full table scans
   - Solution: Enforce partition filters, add query time limits

2. **USER_CANCELED (27.3%)**: Users manually stopping queries
   - Common cause: Queries taking longer than expected
   - Solution: Optimize slow queries, set realistic expectations

3. **EXCEEDED_MEMORY_LIMIT (18.8%)**: Queries using too much memory
   - Common cause: Large JOINs, window functions, aggregations
   - Solution: Optimize JOINs, use approximate functions

4. **QUERY_REJECTED (12.3%)**: Missing required partition filters
   - Common cause: Partition filter policy enforcement
   - Solution: User education, query templates

====================================================================================================
🔥 ANTI-PATTERNS DETECTED
====================================================================================================

1. ❌ Missing Partition Filters: 8,607 queries (67.0%)

   Impact:
   - Full table scans instead of partition pruning
   - 70-90% unnecessary data scanned
   - Primary cause of timeouts and high costs

   Example:
   ```sql
   -- BAD: No partition filter
   SELECT * FROM payments WHERE merchant_id = 'ABC123'

   -- GOOD: With partition filter
   SELECT * FROM payments
   WHERE created_date >= '2026-01-25'
     AND merchant_id = 'ABC123'
   ```

   Estimated waste: ~2,690 CPU-hours (70% of total)

2. ❌ SELECT * Pattern: 5,781 queries (45.0%)

   Impact:
   - Reading unnecessary columns
   - Increased I/O and memory usage
   - 20-50% overhead

   Example:
   ```sql
   -- BAD: SELECT * reading all 50 columns
   SELECT * FROM transactions WHERE id = '123'

   -- GOOD: Select only needed columns
   SELECT id, amount, status, created_at
   FROM transactions WHERE id = '123'
   ```

   Estimated waste: ~385 CPU-hours (10% of total)

3. ❌ COUNT(DISTINCT) on Large Datasets: 892 queries (6.9%)

   Impact:
   - Very expensive aggregation
   - High memory usage
   - 40-60% slower than approximate functions

   Example:
   ```sql
   -- BAD: Exact count, very expensive
   SELECT COUNT(DISTINCT user_id) FROM events

   -- GOOD: Approximate count, 40-60% faster
   SELECT approx_distinct(user_id) FROM events
   ```

   Estimated waste: ~178 CPU-hours (4.6% of total)

4. ⚠️  Functions on JOIN Columns: 567 queries (4.4%)

   Impact:
   - Prevents index usage
   - Prevents predicate pushdown
   - 30-50% slower JOINs

   Example:
   ```sql
   -- BAD: Function on JOIN column
   SELECT * FROM users u
   JOIN transactions t ON concat('user_', u.id) = t.user_key

   -- GOOD: Pre-compute or use direct column
   SELECT * FROM users u
   JOIN transactions t ON u.user_key = t.user_key
   ```

   Estimated waste: ~85 CPU-hours (2.2% of total)

5. ⚠️  Multiple LEFT JOINs Without Filters: 434 queries (3.4%)

   Impact:
   - Large intermediate result sets
   - High memory usage
   - Cartesian products risk

   Example:
   ```sql
   -- BAD: Multiple LEFT JOINs, no filters
   SELECT * FROM orders o
   LEFT JOIN customers c ON o.customer_id = c.id
   LEFT JOIN products p ON o.product_id = p.id
   LEFT JOIN shipping s ON o.id = s.order_id

   -- GOOD: Add filters, consider INNER JOIN if appropriate
   SELECT o.id, c.name, p.title, s.status
   FROM orders o
   INNER JOIN customers c ON o.customer_id = c.id
   INNER JOIN products p ON o.product_id = p.id
   LEFT JOIN shipping s ON o.id = s.order_id
   WHERE o.created_date >= '2026-01-01'
   ```

   Estimated waste: ~130 CPU-hours (3.4% of total)

====================================================================================================
📊 DATA SCAN ANALYSIS
====================================================================================================

Total Data Scanned: 248.7 TB

By Table:
┌─────────────────────────────────────┬──────────────┬──────────────┬─────────────┐
│ Table                               │ Data Scanned │ % of Total   │ Query Count │
├─────────────────────────────────────┼──────────────┼──────────────┼─────────────┤
│ events.lumberjack                   │ 98.4 TB      │ 39.6%        │ 2,345       │
│ payments.transactions               │ 67.2 TB      │ 27.0%        │ 3,456       │
│ merchants.merchant_details          │ 34.5 TB      │ 13.9%        │ 1,789       │
│ users.user_activity                 │ 23.1 TB      │ 9.3%         │ 1,234       │
│ payouts.payout_attempts             │ 12.3 TB      │ 4.9%         │ 892         │
│ others                              │ 13.2 TB      │ 5.3%         │ 3,131       │
└─────────────────────────────────────┴──────────────┴──────────────┴─────────────┘

🔍 Insights:
   - events.lumberjack: 98.4 TB (39.6%)
     └─ Most frequently queried table
     └─ High scan volume suggests missing partition filters
     └─ Recommend: Enforce created_date partition filter

   - payments.transactions: 67.2 TB (27.0%)
     └─ Second highest scan volume
     └─ Likely candidate for optimization
     └─ Recommend: Review query patterns

====================================================================================================
⏰ QUERY DURATION DISTRIBUTION
====================================================================================================

Execution Time Breakdown:
┌─────────────────────┬──────────┬──────────────┬─────────────────┐
│ Duration Range      │ Count    │ % of Total   │ Avg CPU Time    │
├─────────────────────┼──────────┼──────────────┼─────────────────┤
│ < 10 seconds        │ 5,234    │ 40.7%        │ 2.3 min         │
│ 10s - 1 min         │ 3,456    │ 26.9%        │ 8.7 min         │
│ 1 min - 5 min       │ 2,345    │ 18.2%        │ 23.4 min        │
│ 5 min - 15 min      │ 1,234    │ 9.6%         │ 67.8 min        │
│ 15 min - 1 hour     │ 456      │ 3.5%         │ 156.2 min       │
│ > 1 hour            │ 122      │ 0.9%         │ 342.5 min       │
└─────────────────────┴──────────┴──────────────┴─────────────────┘

🎯 Optimization Targets:
   - Queries > 15 minutes: 578 queries (4.5%)
     └─ Consuming disproportionate resources
     └─ 892.3 CPU-hours (23.2% of total!)
     └─ High optimization potential

   - Queries > 1 hour: 122 queries (0.9%)
     └─ Consuming 418.7 CPU-hours (10.9% of total!)
     └─ URGENT optimization needed

====================================================================================================
💡 OPTIMIZATION OPPORTUNITIES
====================================================================================================

Priority 1: Missing Partition Filters (67% of queries)
├─ Affected queries: 8,607
├─ Current waste: ~2,690 CPU-hours
├─ Potential savings: 70-90% reduction → ~1,881 - 2,421 CPU-hours
└─ Action: Enforce partition filter policy

Priority 2: Reduce Query Failures (28.1% failure rate)
├─ Failed queries: 3,613
├─ Wasted CPU: 1,082 CPU-hours
├─ Potential savings: 50-80% reduction → ~541 - 866 CPU-hours
└─ Actions:
    - Fix queries exceeding time limits
    - Optimize memory-intensive queries
    - User training on query best practices

Priority 3: Optimize SELECT * Patterns (45% of queries)
├─ Affected queries: 5,781
├─ Current waste: ~385 CPU-hours
├─ Potential savings: 20-30% reduction → ~77 - 116 CPU-hours
└─ Action: Educate users to select only needed columns

Priority 4: Replace COUNT(DISTINCT) (892 queries)
├─ Affected queries: 892
├─ Current waste: ~178 CPU-hours
├─ Potential savings: 40-60% reduction → ~71 - 107 CPU-hours
└─ Action: Use approx_distinct() for large datasets

Priority 5: Fix JOIN Anti-patterns (567 queries)
├─ Affected queries: 567
├─ Current waste: ~85 CPU-hours
├─ Potential savings: 30-50% reduction → ~26 - 43 CPU-hours
└─ Action: Remove functions from JOIN conditions

====================================================================================================
📈 PROJECTED SAVINGS
====================================================================================================

Current Weekly Cost: 3,847.2 CPU-hours

Optimistic Scenario (50% improvement across all categories):
├─ Partition filters: 1,345 CPU-hours saved
├─ Failure reduction: 541 CPU-hours saved
├─ SELECT * optimization: 77 CPU-hours saved
├─ COUNT(DISTINCT) replacement: 71 CPU-hours saved
├─ JOIN optimization: 26 CPU-hours saved
└─ Total savings: ~2,060 CPU-hours/week (53.5% reduction)

Conservative Scenario (30% improvement):
├─ Partition filters: 807 CPU-hours saved
├─ Failure reduction: 325 CPU-hours saved
├─ SELECT * optimization: 46 CPU-hours saved
├─ COUNT(DISTINCT) replacement: 43 CPU-hours saved
├─ JOIN optimization: 16 CPU-hours saved
└─ Total savings: ~1,237 CPU-hours/week (32.2% reduction)

Annual Projection (Conservative):
└─ 1,237 CPU-hours/week × 52 weeks = 64,324 CPU-hours/year saved

====================================================================================================
✅ RECOMMENDED ACTIONS
====================================================================================================

Immediate (This Week):

1. 🚨 Contact prasad.pawar@razorpay.com
   - 47.5% failure rate (744 failed queries)
   - 892 CPU-hours consumed, ~424 hours wasted
   - Action: Query review session, identify failing patterns

2. 📋 Enforce Partition Filter Policy
   - 67% of queries missing partition filters
   - Implement QUERY_REJECTED for queries without date filters
   - Create documentation and examples

3. 🔍 Review Top 10 Most Expensive Queries
   - Get query IDs for top 10 by CPU time
   - Analyze for optimization opportunities
   - Target 50% cost reduction

Short-term (This Month):

4. 📚 User Education Program
   - Create query best practices guide
   - Conduct workshops for top 10 users
   - Share optimization examples

5. 🛠️  Create Query Templates
   - Common query patterns with best practices
   - Pre-optimized templates for frequent use cases
   - Include partition filters by default

6. 📊 Set Up Monitoring
   - Daily failure rate alerts (> 30%)
   - Weekly cost reports by user
   - Query duration alerts (> 30 minutes)

Long-term (This Quarter):

7. 🏗️  Optimize Hot Tables
   - events.lumberjack (98.4 TB scanned)
   - payments.transactions (67.2 TB scanned)
   - Consider materialized views for common aggregations

8. 🤖 Automated Query Validation
   - Pre-execution query validation
   - Estimated cost warnings
   - Suggest optimizations before running

9. 📈 Cost Attribution and Budgets
   - Team-level cost tracking
   - Monthly cost budgets per team
   - Cost optimization incentives

====================================================================================================
```

## Understanding Cluster Insights

### What This Analysis Provides

1. **Health Overview**: Overall cluster health, resource consumption, query success rate
2. **User Patterns**: Who's using the cluster, how much, and how efficiently
3. **Failure Analysis**: Why queries fail and what's being wasted
4. **Anti-patterns**: Common mistakes and their impact
5. **Optimization Roadmap**: Prioritized actions with estimated savings

### Key Metrics to Watch

- **Failure Rate**: > 20% indicates systemic issues
- **Cost Concentration**: If top 3 users consume > 70%, focus optimization there
- **Partition Filter Coverage**: Should be > 90% for partitioned tables
- **Query Duration**: > 5% of queries taking > 15 min indicates problems

### Red Flags

- 🚨 **High failure rate** (> 30%): User training or query validation needed
- 🚨 **Individual user with high failures**: Immediate intervention required
- 🚨 **Missing partition filters**: Most common cost driver
- 🚨 **Growing query duration**: May indicate data growth without optimization

## Common Follow-up Questions

**Q: Show me the actual failing queries for a specific user**
```
Show me all failed queries for prasad.pawar@razorpay.com in the last 7 days
with failure reasons and query text
```

**Q: Which queries are missing partition filters?**
```
Find all queries from querybook cluster in last 7 days that are missing
partition filters on created_date
```

**Q: What are the top 10 most expensive queries?**
```
Give me top 10 expensive queries from trino-querybook cluster in last 7 days
```

**Q: Show me trends over time**
```
Show me daily CPU usage and failure rate trends for querybook cluster
over the last 30 days
```

**Q: Compare this week to last week**
```
Compare querybook cluster metrics for this week vs last week -
show improvements or regressions
```

## Tips for Cluster Analysis

1. **Regular Reviews**: Weekly cluster analysis helps catch issues early

2. **Focus on Failures**: High failure rate = wasted resources + user frustration

3. **User Education**: Most issues stem from lack of knowledge, not malice

4. **Enforce Policies**: Partition filters should be required, not optional

5. **Track Progress**: Compare week-over-week to measure improvement

6. **Celebrate Wins**: Share optimization success stories with teams

## Related Examples

- **Example 1**: Find top 10 expensive queries by cluster
- **Example 2**: Analyze and optimize a specific query
- **Example 4**: Get detailed query plan analysis
- **Example 5**: Compare query performance before/after optimization

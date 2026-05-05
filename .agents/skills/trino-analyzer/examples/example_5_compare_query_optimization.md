# Example 5: Compare Query Performance Before/After Optimization

## Scenario
You optimized a query and want to verify the improvement by comparing resource usage before and after the optimization.

## How to Ask Claude

Provide both query IDs or the queries themselves:

```
Compare the performance of these two queries:
Before: query ID 20260131_082417_00234_abcde
After: query ID 20260131_143522_00456_fghij
```

Or with actual queries:
```
I optimized this query. Can you compare the performance?

Original query:
[paste original query]

Optimized query:
[paste optimized query]

Show me the resource improvement.
```

Or just describe what you did:
```
I added a partition filter to my query. The original query ID is
20260131_082417_00234_abcde and the new one is 20260131_143522_00456_fghij.
How much did I improve it?
```

## Example Comparison Request

```
Compare these two query versions:

BEFORE (Query ID: 20260130_104523_00789_xyzab):
SELECT
  merchant_id,
  COUNT(*) as total_transactions,
  SUM(amount) as total_amount
FROM payments.transactions
WHERE status = 'captured'
  AND FROM_UNIXTIME(created_at) >= CURRENT_DATE - INTERVAL '7' DAY
GROUP BY merchant_id;

AFTER (Query ID: 20260131_153045_01234_pqrst):
SELECT
  merchant_id,
  COUNT(*) as total_transactions,
  SUM(amount) as total_amount
FROM payments.transactions
WHERE created_date >= CAST(CURRENT_DATE - INTERVAL '7' DAY AS VARCHAR)
  AND FROM_UNIXTIME(created_at) >= CURRENT_DATE - INTERVAL '7' DAY
  AND status = 'captured'
GROUP BY merchant_id;

Show me the improvement metrics.
```

## Claude's Comparison Output

```
====================================================================================================
QUERY PERFORMANCE COMPARISON - BEFORE vs AFTER OPTIMIZATION
====================================================================================================

📊 OPTIMIZATION SUMMARY:
   What changed: Added partition filter on created_date column
   Expected impact: 70-90% reduction in data scanned and CPU time

====================================================================================================
📈 RESOURCE METRICS COMPARISON
====================================================================================================

┌─────────────────────────┬──────────────────┬──────────────────┬──────────────────┐
│ Metric                  │ BEFORE           │ AFTER            │ Improvement      │
├─────────────────────────┼──────────────────┼──────────────────┼──────────────────┤
│ Query ID                │ 20260130_104523  │ 20260131_153045  │ -                │
│ Status                  │ FINISHED         │ FINISHED         │ ✅ Both success  │
│                         │                  │                  │                  │
│ CPU Time                │ 456.7 hours      │ 45.2 hours       │ ⬇️ 90.1% faster  │
│                         │ (1,644,120 sec)  │ (162,720 sec)    │ (411.5 hrs saved)│
│                         │                  │                  │                  │
│ Peak Memory             │ 234.5 GB         │ 28.3 GB          │ ⬇️ 87.9% less    │
│                         │                  │                  │ (206.2 GB saved) │
│                         │                  │                  │                  │
│ Data Scanned            │ 45.6 TB          │ 4.2 TB           │ ⬇️ 90.8% less    │
│                         │ (46,694 GB)      │ (4,301 GB)       │ (41.4 TB saved)  │
│                         │                  │                  │                  │
│ Input Rows              │ 8,234,567,890    │ 756,234,123      │ ⬇️ 90.8% less    │
│                         │                  │                  │                  │
│ Execution Time          │ 1,845 seconds    │ 234 seconds      │ ⬇️ 87.3% faster  │
│                         │ (30.8 minutes)   │ (3.9 minutes)    │ (26.9 min saved) │
│                         │                  │                  │                  │
│ Output Rows             │ 12,345           │ 12,345           │ ✅ Identical     │
│                         │                  │                  │                  │
│ Query Type              │ SELECT           │ SELECT           │ -                │
│ User                    │ analytics.team@  │ analytics.team@  │ -                │
│ Cluster                 │ trino-querybook  │ trino-querybook  │ -                │
└─────────────────────────┴──────────────────┴──────────────────┴──────────────────┘

====================================================================================================
🎯 KEY IMPROVEMENTS
====================================================================================================

1. ⬇️ CPU Time: 90.1% reduction (456.7h → 45.2h)
   ├─ Saved: 411.5 CPU-hours per execution
   ├─ Impact: If query runs daily: ~12,345 CPU-hours/month saved
   └─ Rating: ⭐⭐⭐⭐⭐ EXCELLENT improvement

2. ⬇️ Data Scanned: 90.8% reduction (45.6 TB → 4.2 TB)
   ├─ Saved: 41.4 TB per execution
   ├─ Impact: Partition filter working perfectly
   └─ Rating: ⭐⭐⭐⭐⭐ EXCELLENT improvement

3. ⬇️ Peak Memory: 87.9% reduction (234.5 GB → 28.3 GB)
   ├─ Saved: 206.2 GB
   ├─ Impact: Much lower risk of memory limit failures
   └─ Rating: ⭐⭐⭐⭐⭐ EXCELLENT improvement

4. ⬇️ Execution Time: 87.3% reduction (30.8 min → 3.9 min)
   ├─ Saved: 26.9 minutes of waiting
   ├─ Impact: 7.9x faster user experience
   └─ Rating: ⭐⭐⭐⭐⭐ EXCELLENT improvement

5. ✅ Output Correctness: Identical (12,345 rows)
   ├─ Same results in fraction of the time
   ├─ Confirms optimization didn't change logic
   └─ Rating: ✅ VERIFIED CORRECT

====================================================================================================
📊 VISUAL COMPARISON
====================================================================================================

CPU Time (hours):
BEFORE: ████████████████████████████████████████████████  456.7 hours
AFTER:  ████                                               45.2 hours
        └─ 90.1% reduction

Data Scanned (TB):
BEFORE: ████████████████████████████████████████████████  45.6 TB
AFTER:  ████                                               4.2 TB
        └─ 90.8% reduction

Execution Time (minutes):
BEFORE: ████████████████████████████████████████████████  30.8 min
AFTER:  ████                                                3.9 min
        └─ 87.3% reduction

Peak Memory (GB):
BEFORE: ████████████████████████████████████████████████  234.5 GB
AFTER:  ███                                                28.3 GB
        └─ 87.9% reduction

====================================================================================================
🔍 WHAT CAUSED THE IMPROVEMENT?
====================================================================================================

Change Made: Added partition filter
└─ BEFORE: WHERE FROM_UNIXTIME(created_at) >= CURRENT_DATE - INTERVAL '7' DAY
└─ AFTER:  WHERE created_date >= CAST(CURRENT_DATE - INTERVAL '7' DAY AS VARCHAR)

Why This Helped:

1. ✅ Partition Pruning Enabled
   - BEFORE: Full table scan across all partitions (500+ days of data)
   - AFTER: Only scans 7 partitions (7 days of data)
   - Result: 90.8% less data scanned (45.6 TB → 4.2 TB)

2. ✅ Reduced Data = Reduced Processing
   - Less data → Less CPU needed to process
   - Less data → Less memory needed to hold
   - Less data → Faster execution
   - Result: All metrics improved proportionally

3. ✅ Filter Pushdown to Storage
   - Partition filter applied at storage layer (S3/HDFS)
   - Only relevant files read into memory
   - Trino doesn't waste time reading irrelevant data
   - Result: Massive I/O savings

====================================================================================================
💰 COST IMPACT ANALYSIS
====================================================================================================

Per Execution Savings:
├─ CPU Time saved: 411.5 hours
├─ Data I/O saved: 41.4 TB
├─ Memory saved: 206.2 GB-hours
└─ Time saved: 26.9 minutes

If Query Runs Daily:
├─ Weekly savings: 2,880 CPU-hours (411.5 × 7)
├─ Monthly savings: 12,345 CPU-hours (411.5 × 30)
└─ Annual savings: 150,198 CPU-hours (411.5 × 365)

If Query Runs Hourly:
├─ Daily savings: 9,876 CPU-hours (411.5 × 24)
├─ Weekly savings: 69,132 CPU-hours
└─ Monthly savings: 296,280 CPU-hours

User Experience:
├─ BEFORE: Wait 30.8 minutes for results
├─ AFTER: Wait 3.9 minutes for results
└─ 7.9x faster response time!

====================================================================================================
✅ OPTIMIZATION RATING
====================================================================================================

Overall Grade: ⭐⭐⭐⭐⭐ A+ (EXCELLENT)

┌─────────────────────────┬─────────┬────────────────────────────────┐
│ Criteria                │ Rating  │ Comments                       │
├─────────────────────────┼─────────┼────────────────────────────────┤
│ CPU Reduction           │ ⭐⭐⭐⭐⭐ │ 90.1% - Outstanding            │
│ Memory Reduction        │ ⭐⭐⭐⭐⭐ │ 87.9% - Excellent              │
│ I/O Reduction           │ ⭐⭐⭐⭐⭐ │ 90.8% - Outstanding            │
│ Execution Speed         │ ⭐⭐⭐⭐⭐ │ 87.3% - Excellent              │
│ Result Correctness      │ ✅      │ Identical output - Verified    │
│ Implementation Effort   │ ⭐⭐⭐⭐⭐ │ Simple 1-line change           │
│ Maintainability         │ ⭐⭐⭐⭐⭐ │ Clearer intent, best practice  │
└─────────────────────────┴─────────┴────────────────────────────────┘

Benchmarks:
├─ 10-30% improvement: ⭐⭐ Good
├─ 30-50% improvement: ⭐⭐⭐ Great
├─ 50-70% improvement: ⭐⭐⭐⭐ Excellent
├─ 70-90% improvement: ⭐⭐⭐⭐⭐ Outstanding
└─ > 90% improvement: 🏆 Exceptional

Your 90% improvement = ⭐⭐⭐⭐⭐ Outstanding!

====================================================================================================
📝 QUERY PLAN COMPARISON (if available)
====================================================================================================

BEFORE - Query Plan Highlights:
├─ TableScan: payments.transactions
│  └─ Partitions scanned: 547 partitions (all data)
│  └─ Estimated rows: 8.2 billion
├─ Filter: status = 'captured'
│  └─ Applied AFTER reading all data
├─ Aggregate: GROUP BY merchant_id
└─ Exchange: HASH partitioning on merchant_id

AFTER - Query Plan Highlights:
├─ TableScan: payments.transactions
│  └─ Partitions scanned: 7 partitions (last 7 days only) ✅
│  └─ Estimated rows: 756 million (90.8% reduction) ✅
│  └─ created_date >= '2026-01-25' :: [[2026-01-25], [2026-01-26], ...]
├─ Filter: status = 'captured'
│  └─ Applied to much smaller dataset
├─ Aggregate: GROUP BY merchant_id
└─ Exchange: HASH partitioning on merchant_id

Key Difference:
✅ Partition pruning: 547 partitions → 7 partitions (98.7% reduction)

====================================================================================================
🎓 LESSONS LEARNED
====================================================================================================

1. ✅ Always Use Partition Filters
   - Single most impactful optimization
   - 70-90% cost reduction typical
   - Simple to implement (1 line change)

2. ✅ Verify Results Match
   - Output rows identical (12,345 = 12,345) ✅
   - Critical to confirm optimization correctness
   - Always check before deploying to production

3. ✅ Measure Impact
   - Before: 456.7 CPU-hours
   - After: 45.2 CPU-hours
   - Quantify savings to justify time spent optimizing

4. ✅ Document Success
   - Share optimization results with team
   - Encourage similar improvements
   - Build best practices library

====================================================================================================
🚀 NEXT STEPS
====================================================================================================

1. ✅ Deploy to Production
   - Results verified, safe to deploy
   - Update any dashboards or reports using this query
   - Consider this the new standard query

2. 📚 Share the Win
   - Document this optimization as a case study
   - Share with team: "Added partition filter, saved 90% resources"
   - Encourage others to review their queries

3. 🔍 Find Similar Queries
   - Look for other queries missing partition filters
   - Apply same optimization pattern
   - Potential for 10x-100x impact if pattern is common

   Ask Claude:
   ```
   Find all queries from last 7 days that are missing partition
   filters on created_date and have CPU time > 10 hours
   ```

4. 📊 Monitor Ongoing Performance
   - Track this query's performance over time
   - Ensure optimization remains effective
   - Watch for data growth requiring further optimization

====================================================================================================
💡 COMMON COMPARISON PATTERNS
====================================================================================================

Pattern 1: Partition Filter (This Example)
├─ Before: No partition filter → full table scan
├─ After: Added partition filter → partition pruning
└─ Expected: 70-90% improvement ✅ Achieved: 90.1%

Pattern 2: SELECT * Optimization
├─ Before: SELECT * (50 columns)
├─ After: SELECT specific columns (5 columns)
└─ Expected: 20-40% improvement

Pattern 3: COUNT(DISTINCT) → approx_distinct()
├─ Before: COUNT(DISTINCT user_id)
├─ After: approx_distinct(user_id)
└─ Expected: 40-60% improvement

Pattern 4: Remove Function from JOIN
├─ Before: JOIN ON concat('user_', u.id) = t.user_key
├─ After: JOIN ON u.user_key = t.user_key
└─ Expected: 30-50% improvement

Pattern 5: Add LIMIT for Testing
├─ Before: Full query (millions of rows)
├─ After: Same query + LIMIT 1000
└─ Expected: 80-95% improvement (for testing only)

====================================================================================================
⚠️  WHEN OPTIMIZATION DOESN'T WORK
====================================================================================================

If your "after" query shows worse or similar performance:

1. ❌ Partition Filter Not Applied
   - Check query plan: Are partitions still being scanned?
   - Verify partition column name (created_date vs date vs dt)
   - Ensure partition key is VARCHAR, not DATE

2. ❌ Results Don't Match
   - Output rows different? (12,345 vs 9,876)
   - Logic error in optimization
   - Need to fix before deploying

3. ❌ Minimal Improvement (< 10%)
   - Optimization not addressing root cause
   - Try different approach
   - Get query plan analysis to identify real bottleneck

4. ❌ Performance Regression (slower after!)
   - Added inefficiency (e.g., unnecessary DISTINCT)
   - Data distribution changed
   - Revert and re-analyze

Ask Claude for help:
```
My optimization didn't work as expected. Here are the metrics:
[paste comparison]

What went wrong?
```

====================================================================================================
```

## Understanding the Comparison

### What Claude Does

1. **Retrieves Metrics**: Fetches actual execution metrics from both query IDs
2. **Compares Resources**: CPU time, memory, data scanned, execution time
3. **Verifies Correctness**: Checks output rows match (critical!)
4. **Calculates Impact**: Shows % improvement and absolute savings
5. **Explains Why**: Identifies what changed and why it helped
6. **Rates Success**: Grades optimization (⭐⭐⭐⭐⭐ scale)
7. **Provides Context**: Projects savings if query runs frequently

### Key Metrics to Compare

**Must Match:**
- ✅ Output rows (12,345 = 12,345) - Proves correctness
- ✅ Query logic - Same business logic

**Should Improve:**
- ⬇️ CPU time - Lower is better
- ⬇️ Data scanned - Lower is better
- ⬇️ Peak memory - Lower is better
- ⬇️ Execution time - Faster is better

### Success Criteria

| Improvement | Rating | Assessment |
|-------------|--------|------------|
| < 10% | ⭐ | Minimal - probably not worth it |
| 10-30% | ⭐⭐ | Good - worthwhile for frequent queries |
| 30-50% | ⭐⭐⭐ | Great - definitely worth it |
| 50-70% | ⭐⭐⭐⭐ | Excellent - significant impact |
| 70-90% | ⭐⭐⭐⭐⭐ | Outstanding - major improvement |
| > 90% | 🏆 | Exceptional - transformative |

## Common Follow-up Questions

**Q: The results don't match - what do I do?**
```
My optimized query returns different results:
Before: 12,345 rows
After: 9,876 rows

What went wrong? Here are the queries:
[paste both queries]
```

**Q: Why didn't I see the expected improvement?**
```
I added a partition filter but only saw 15% improvement instead of 70%.
Query IDs: [before] and [after]
What's wrong?
```

**Q: Can you show me the query plans for both?**
```
Compare the query plans for these two queries:
Before: [query ID]
After: [query ID]

Show me what changed in the execution strategy.
```

**Q: How do I verify this in production?**
```
I want to A/B test this optimization safely. How should I:
1. Verify results match
2. Measure real-world improvement
3. Roll out gradually
```

## Tips for Effective Comparisons

1. **Always Verify Correctness First**
   - Check output row count matches
   - Spot-check actual data (sample rows)
   - Run both queries on same date range

2. **Use Same Cluster**
   - Run both on same cluster for fair comparison
   - Similar time of day (avoid peak vs off-peak)
   - Same data (don't compare queries run weeks apart)

3. **Document the Change**
   - What optimization was applied
   - Why you expected improvement
   - Actual vs expected results

4. **Consider Frequency**
   - One-time query: 50% improvement may not matter
   - Hourly query: Even 10% improvement = big impact
   - Calculate cumulative savings

5. **Share Success Stories**
   - Document big wins (> 50% improvement)
   - Create internal case studies
   - Encourage team to optimize

## Red Flags

⚠️ **Results don't match**
- Output rows different
- Data values different
- Logic error - DO NOT DEPLOY

⚠️ **Performance regression**
- After query slower than before
- Added inefficiency
- Revert and re-analyze

⚠️ **Minimal improvement**
- < 10% improvement
- Not worth the complexity
- Try different optimization

⚠️ **Inconsistent improvements**
- CPU improved but memory increased significantly
- May indicate different execution path
- Review query plan

## Related Examples

- **Example 1**: Find top 10 expensive queries to identify optimization candidates
- **Example 2**: Analyze and optimize a specific query
- **Example 3**: Cluster insights to find patterns
- **Example 4**: Understanding cost metrics and calculations

# Example 4: Understanding Query Cost and Resource Metrics

## Important: How Query "Cost" is Measured

### What We Actually Track

The Trino query analyzer skill tracks **resource consumption**, not dollar costs:

1. **CPU Time** (seconds/hours) - Primary cost indicator
2. **Peak Memory** (GB) - Memory consumption
3. **Data Scanned** (GB/TB) - I/O volume
4. **Execution Time** (seconds) - Wall-clock duration
5. **Query State** (FINISHED, FAILED, etc.)

### We DON'T Calculate Dollar Costs

**Important:** The skill does NOT use the `query_cost_dollars` column or estimate monetary costs because:

1. **Cost formulas vary by organization** - Different companies have different cloud pricing, reserved capacity, spot instances, etc.
2. **CPU Time ≠ Execution Time** - This is critical to understand (explained below)
3. **Cost attribution is complex** - Involves cluster overhead, network, storage tiers, etc.

Instead, we focus on **resource metrics** which are:
- ✅ Accurate and measurable
- ✅ Directly actionable for optimization
- ✅ Consistent across environments

---

## Critical Concept: CPU Time vs Execution Time

### ⚠️ Common Misconception

When you see **CPU Time = 1,066 hours**, it does NOT mean:
- ❌ The query ran for 1,066 hours (44 days)
- ❌ You waited 1,066 hours for results

### ✅ What It Actually Means

**CPU Time is the SUM of compute time across ALL workers and threads**

Example:
```
Query runs on: 32 workers × 64 threads each = 2,048 parallel threads
Execution time: 30 minutes (wall-clock time you waited)
CPU Time: 30 min × 2,048 threads = 61,440 minutes = 1,024 hours

So you waited 30 minutes, but consumed 1,024 CPU-hours of compute!
```

### Real-World Example from Historical Data

From the actual query analysis earlier:

```
Query ID: 20221226_231002_19281_iqc9y
├─ CPU Time: 1,066.08 hours (3,837,888 seconds)
├─ Execution Time: ~30-60 minutes (estimated based on pattern)
├─ Parallelism: ~32-128 workers × 64 threads
└─ Interpretation: High parallelism on large dataset
```

This is why **CPU Time is the best cost indicator** - it represents total compute resources consumed, regardless of parallelism.

---

## How the Skill Analyzes Queries

### Step 1: Query Historical Metrics Table

```sql
SELECT
    query_id,
    user,
    query,
    cpu_time,              -- Total CPU seconds across all workers
    peak_mem,              -- Peak memory used (bytes)
    input_size,            -- Data read from storage (bytes)
    input_rows,            -- Number of rows processed
    execution_time,        -- Wall-clock execution time
    elapsed_time,          -- Total elapsed time
    query_date_ts,         -- When query was created
    query_type,            -- SELECT, INSERT, etc.
    cluster_label,         -- Which cluster (trino-tableau, etc.)
    state                  -- FINISHED, FAILED, etc.
FROM iceberg.de_metrics.trino_query_analyzer_metrics
WHERE cpu_time IS NOT NULL
  AND cpu_time > 0
  AND query_date_ts >= CURRENT_TIMESTAMP - INTERVAL '7' DAY
ORDER BY cpu_time DESC
LIMIT 10
```

### Step 2: Analyze Resource Metrics

For each query, we report:

```
CPU Time: 1,066.08 hours (primary cost indicator)
├─ This is total compute across all workers
├─ Higher = more expensive
└─ Use this to prioritize optimization

Peak Memory: 490.00 GB
├─ Maximum memory used during execution
├─ High memory = potential for EXCEEDED_MEMORY_LIMIT failures
└─ Optimize: Reduce JOIN sizes, use approximate functions

Data Scanned: 42.605 TB
├─ Total data read from storage (S3, HDFS, etc.)
├─ High scan = likely missing partition filters
└─ Optimize: Add partition filters, select specific columns

Execution Time: 1,234 seconds (wall-clock)
├─ How long you waited for results
├─ May be much shorter than CPU time due to parallelism
└─ User experience metric
```

### Step 3: Identify Optimization Opportunities

Based on patterns in the metrics:

```python
# High data scanned + high CPU time = likely missing partition filter
if data_scanned > 1_TB and cpu_time > 100_hours:
    issue = "Missing partition filter - full table scan"
    potential_saving = "70-90% reduction"

# High CPU time + many rows = likely aggregation or JOIN issue
if cpu_time > 50_hours and input_rows > 1_billion:
    issue = "Large aggregation or inefficient JOIN"
    potential_saving = "40-60% reduction"

# High peak memory + failure = memory optimization needed
if peak_mem > 500_GB and state == 'FAILED':
    issue = "Memory-intensive query exceeding limits"
    potential_saving = "Prevent failure + 30-50% reduction"
```

---

## What "Cost" Really Means in Different Contexts

### Context 1: "Top 10 Expensive Queries"

**Meaning:** Queries consuming most CPU time

```
Query #1: 1,066 CPU-hours
├─ Interpretation: Consumed 1,066 hours of worker compute time
├─ Impact: Highest resource consumer
└─ Priority: Optimize first (highest impact)
```

**NOT:** Queries with highest dollar cost (we don't calculate this)

### Context 2: "Estimated Cost Reduction"

**Meaning:** Expected CPU time reduction from optimization

```
Current: 1,066 CPU-hours
Optimized (estimated): 320 CPU-hours
Reduction: 746 CPU-hours (70% improvement)

Interpretation:
├─ If this query runs daily: 746 CPU-hours/day saved
├─ If runs hourly: 17,904 CPU-hours/day saved
└─ Actual dollar savings: Depends on your cloud pricing
```

**NOT:** Dollar cost savings (varies by org)

### Context 3: "Query Cost Analysis"

**Meaning:** Resource consumption breakdown

```
CPU Cost: 1,066 hours of compute
Memory Cost: 490 GB peak usage for 1,234 seconds
I/O Cost: 42.6 TB data scanned
Network Cost: Data shuffled across workers (from query plan)

Interpretation:
├─ Primary cost driver: CPU time (1,066 hours)
├─ Secondary driver: Data scanned (42.6 TB)
└─ Optimization target: Reduce both with partition filter
```

---

## How to Interpret Resource Metrics

### CPU Time Benchmarks

```
< 1 hour:        Normal for most queries
1-10 hours:      Moderate - worth reviewing if frequent
10-100 hours:    High - optimization recommended
100-1000 hours:  Very high - urgent optimization needed
> 1000 hours:    Extremely high - likely missing partition filter
```

### Data Scanned Benchmarks

```
< 1 GB:          Small query
1-100 GB:        Medium query
100 GB - 1 TB:   Large query - verify partition filter present
1-10 TB:         Very large - likely missing partition filter
> 10 TB:         Extremely large - almost certainly full table scan
```

### Failure Cost (Wasted Resources)

```
Failed Query Example:
├─ CPU Time: 335 hours
├─ State: FAILED
├─ Reason: EXCEEDED_TIME_LIMIT
└─ Wasted: 335 CPU-hours produced NO results

This is pure waste - resources consumed with no output.
Reducing failures has immediate impact.
```

---

## Example: How We Report "Cost" in Analysis

### Sample Output from Skill

```
====================================================================================================
#1 - MOST EXPENSIVE QUERY
====================================================================================================
Query ID: 20221226_231002_19281_iqc9y
User: shorya.saini@razorpay.com

📈 RESOURCES CONSUMED:
   ⏱️  CPU Time:      1,066.08 hours  (3,837,888 seconds)
   💾 Memory:          490.00 GB peak
   📊 Data Scanned:    42.605 TB
   📝 Rows:          148,915,744,372
   ⏰ Execution:       ~30-60 minutes (estimated wall-clock)

🔍 COST ANALYSIS:
   Primary Cost Driver: CPU Time (1,066 hours)
   ├─ With 128 parallel workers, query completed in ~30-60 minutes
   ├─ But consumed 1,066 compute-hours across all workers
   └─ This is the "cost" - total compute resources used

   Secondary Cost Driver: Data Scanned (42.6 TB)
   ├─ Large data scan indicates possible full table scan
   ├─ Likely missing partition filter
   └─ Could be reduced 70-90% with proper filtering

💡 OPTIMIZATION IMPACT:
   Current CPU Time: 1,066 hours
   Estimated Optimized: 107-320 hours (70-90% reduction)

   Potential Savings: 746-959 CPU-hours per execution

   If this query runs:
   ├─ Daily: 746-959 CPU-hours/day saved
   ├─ Weekly: 5,222-6,713 CPU-hours/week saved
   └─ Monthly: 22,380-28,770 CPU-hours/month saved

   Your dollar cost savings: Depends on your cloud pricing model
   (For reference: AWS Athena charges ~$5/TB scanned, EC2 varies widely)
```

### What We're Actually Showing

1. **Resource metrics** - Actual measured values from Trino
2. **Relative cost** - CPU hours as proxy for expense
3. **Optimization potential** - Expected % reduction based on patterns
4. **Frequency impact** - Multiplier if query runs regularly

We do NOT show:
- ❌ Dollar amounts (unless you provide pricing)
- ❌ ROI calculations
- ❌ Total cost of ownership

---

## Why This Approach Works

### Benefits of Resource-Based Analysis

1. **Universal** - Works across any cloud provider or on-prem
2. **Accurate** - Based on actual Trino metrics, not estimates
3. **Actionable** - "Reduce CPU time" is clear, "save $X" requires context
4. **Measurable** - Can verify improvements by re-running queries

### How to Convert to Dollar Costs (If Needed)

If you need dollar costs, apply your org's pricing:

```python
# Example conversion (your rates will differ)
cpu_hours = 1066
memory_gb_hours = 490 * (execution_time_seconds / 3600)
data_scanned_tb = 42.6

# Your org's pricing (example rates)
cpu_rate_per_hour = 0.10        # $0.10/CPU-hour
memory_rate_per_gb_hour = 0.01  # $0.01/GB-hour
s3_scan_rate_per_tb = 5.00      # $5/TB scanned

estimated_cost = (
    cpu_hours * cpu_rate_per_hour +
    memory_gb_hours * memory_rate_per_gb_hour +
    data_scanned_tb * s3_scan_rate_per_tb
)

# Example result:
# $106.60 (CPU) + $8.17 (memory) + $213 (S3) = $327.77
```

But this varies WIDELY based on:
- Reserved instances vs on-demand
- Spot instances discounts
- Enterprise agreements
- Regional pricing
- Storage tier (S3 Standard vs Glacier)
- Network transfer costs
- Etc.

**That's why we focus on resources, not dollars.**

---

## Common Questions

### Q: Why not just show me dollar costs?

**A:** Dollar costs vary by:
- Cloud provider (AWS, GCP, Azure, on-prem)
- Pricing model (on-demand, reserved, spot)
- Enterprise agreements
- Regional pricing
- Time of day (spot pricing)

Resource metrics (CPU hours, data scanned) are universal and accurate.

### Q: How do I know if a query is "expensive"?

**A:** Compare CPU time to cluster average:
```
Cluster average: 18 minutes CPU time/query
Your query: 1,066 hours CPU time

Your query is 3,553x more expensive than average!
This is definitely expensive and worth optimizing.
```

### Q: What about queries that fail - do they cost anything?

**A:** Yes! Failed queries still consume resources:
```
Failed Query:
├─ CPU Time: 335 hours (resources consumed)
├─ Result: FAILED (no output produced)
└─ This is pure waste - 335 CPU-hours with no value

Successful Query:
├─ CPU Time: 335 hours (resources consumed)
├─ Result: Data produced
└─ Resources consumed, but value delivered
```

Reducing failures has immediate ROI.

### Q: Can you estimate how much I'd save in dollars?

**A:** Only if you provide your pricing:
```
Please provide your organization's Trino/Presto pricing:
- Cost per CPU-hour: $X
- Cost per GB-hour memory: $Y
- Cost per TB data scanned: $Z

Then I can estimate dollar savings.
```

Otherwise, we report % reduction in resources.

---

## Summary: What "Cost" Means in This Skill

| Term | What It Means | What We Report |
|------|---------------|----------------|
| "Expensive query" | High resource consumption | CPU time in hours |
| "Query cost" | Resources consumed | CPU, memory, data scanned |
| "Cost reduction" | Resource savings | % reduction in CPU time |
| "Wasted cost" | Failed query resources | CPU hours with no results |
| "Top 10 by cost" | Top 10 by CPU time | Sorted by CPU hours DESC |

**Key Principle:** We measure and optimize **resources** (CPU, memory, I/O), which are accurate and universal. You convert to **dollars** using your org's pricing if needed.

---

## Related Examples

- **Example 1**: Find top expensive queries (sorted by CPU time)
- **Example 2**: Analyze and optimize query (reduce CPU time)
- **Example 3**: Cluster insights (resource consumption patterns)
- **Example 5**: Compare before/after optimization (resource reduction)

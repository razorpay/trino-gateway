---
name: trino-query-cost-analyzer
description: "Analyze Trino query costs by examining query plans, estimating CPU/memory/S3 requirements, and using historical data from delta.dbt_prod_de_metrics.trino_showback for cost prediction. Use when Claude needs to: (1) Predict query cost before execution, (2) Analyze expensive queries and suggest optimizations, (3) Understand query execution plans, (4) Identify cost-saving opportunities, or (5) Compare historical vs predicted costs. Triggers include \"estimate query cost\", \"optimize expensive query\", \"analyze trino cost\", \"predict cost\", \"why is this query expensive\"."
disable-model-invocation: true
---

# Trino Query Cost Analyzer

Comprehensive toolkit for analyzing Trino query costs, optimizing expensive queries, and predicting costs before execution.

## Quick Start

### Predict Query Cost

```bash
# Estimate cost before running
python scripts/predict_query_cost.py --query "SELECT * FROM table WHERE dt >= CURRENT_DATE - INTERVAL '7' DAY"

# From file
python scripts/predict_query_cost.py --query-file query.sql
```

### Analyze Expensive Query

```bash
# Get optimization suggestions
python scripts/identify_optimizations.py --query "SELECT * FROM large_table"

# Analyze query plan
python scripts/analyze_query_plan.py --query "SELECT ..." --estimate-cost
```

### Query Historical Data

```bash
# Get most resource-intensive queries (last 7 days)
python scripts/query_historical_data.py --expensive --days 7 --limit 20

# Get resource usage statistics
python scripts/query_historical_data.py --stats --days 30

# Get specific query metrics
python scripts/query_historical_data.py --query-id "YYYYMMDD_HHMMSS_NNNNN_XXXXX"
```

## Examples

### Example 1: Predicting Cost Before Running a New Query

**Scenario**: You have a new analytics query and want to know if it will be expensive before running it.

**Step 1**: Save your query to a file:
```bash
cat > new_query.sql << 'EOF'
SELECT
    user_id,
    COUNT(*) as transaction_count,
    SUM(amount) as total_amount
FROM hive.payments.transactions
WHERE created_at >= CURRENT_DATE - INTERVAL '90' DAY
GROUP BY user_id
EOF
```

**Step 2**: Predict the cost:
```bash
python scripts/predict_query_cost.py --query-file new_query.sql
```

**Expected Output**:
```json
{
  "estimated_cost_usd": 0.45,
  "confidence": "medium",
  "complexity_score": 6.5,
  "cost_breakdown": {
    "cpu_cost": 0.30,
    "memory_cost": 0.10,
    "io_cost": 0.05
  },
  "plan_features": {
    "has_aggregation": true,
    "has_join": false,
    "table_scans": 1,
    "partition_filters": ["created_at"]
  },
  "recommendations": [
    "Query cost is moderate ($0.45)",
    "Good: Using partition filter on created_at",
    "Consider: Adding LIMIT for testing"
  ]
}
```

**Step 3**: Decide whether to proceed or optimize:
- Cost < $0.10: Safe to run
- Cost $0.10-$1.00: Review recommendations
- Cost > $1.00: Optimize before running

### Example 2: Analyzing and Optimizing an Expensive Query

**Scenario**: A query is running slow and consuming too many resources. You need to identify issues and fix them.

**Step 1**: Get the query metrics from history:
```bash
python scripts/query_historical_data.py --query-id "20250203_145623_00042_xyz89"
```

**Expected Output**:
```json
{
  "query_id": "20250203_145623_00042_xyz89",
  "resource_usage": {
    "cpu_time_seconds": 1847.3,
    "peak_memory_bytes": 4294967296,
    "input_size_bytes": 53687091200,
    "execution_time_seconds": 145.2
  },
  "query_text": "SELECT * FROM large_table WHERE status = 'active'"
}
```

**Analysis**: Query used 1847 CPU seconds and scanned 50GB of data - very expensive!

**Step 2**: Identify optimization opportunities:
```bash
python scripts/identify_optimizations.py --query "SELECT * FROM large_table WHERE status = 'active'"
```

**Expected Output**:
```json
{
  "anti_patterns": [
    {
      "severity": "HIGH",
      "pattern": "SELECT_STAR",
      "message": "Using SELECT * reads all columns unnecessarily",
      "estimated_savings": "30-50%",
      "fix": "Select only needed columns: SELECT id, name, status FROM..."
    },
    {
      "severity": "CRITICAL",
      "pattern": "MISSING_PARTITION_FILTER",
      "message": "No partition filter detected - scanning entire table",
      "estimated_savings": "80-95%",
      "fix": "Add partition filter: WHERE dt >= 'YYYY-MM-DD' AND status = 'active'"
    }
  ],
  "recommendations": [
    "Add partition filter on 'dt' column (if table is partitioned)",
    "Replace SELECT * with specific columns",
    "Consider adding LIMIT during testing"
  ],
  "potential_total_savings": "85-95%"
}
```

**Step 3**: Apply the optimizations:
```bash
cat > optimized_query.sql << 'EOF'
SELECT
    id,
    name,
    status,
    created_at
FROM large_table
WHERE dt >= CURRENT_DATE - INTERVAL '7' DAY
  AND status = 'active'
EOF
```

**Step 4**: Verify the improvements:
```bash
python scripts/predict_query_cost.py --query-file optimized_query.sql
```

**Expected Output**:
```json
{
  "estimated_cost_usd": 0.03,
  "confidence": "high",
  "complexity_score": 2.1,
  "recommendations": [
    "Query cost is low ($0.03)",
    "Good: Using partition filter",
    "Good: Selecting specific columns",
    "Estimated savings vs original: ~93%"
  ]
}
```

**Result**: Cost reduced from ~$2.50 to $0.03 (93% savings)!

### Example 3: Finding and Fixing Top Resource Consumers

**Scenario**: You want to identify the most expensive queries in your cluster and optimize them to reduce overall costs.

**Step 1**: Find the top resource-intensive queries from the last week:
```bash
python scripts/query_historical_data.py --expensive --days 7 --limit 10
```

**Expected Output**:
```json
{
  "expensive_queries": [
    {
      "query_id": "20250201_083015_00123_abc45",
      "cpu_time_seconds": 3421.5,
      "input_size_gb": 127.3,
      "execution_count": 47,
      "total_cost_estimate": "$8.45"
    },
    {
      "query_id": "20250202_141520_00089_def67",
      "cpu_time_seconds": 2156.8,
      "input_size_gb": 89.2,
      "execution_count": 23,
      "total_cost_estimate": "$5.20"
    }
  ]
}
```

**Step 2**: Get details on the most expensive query:
```bash
python scripts/query_historical_data.py --query-id "20250201_083015_00123_abc45"
```

**Step 3**: Analyze it for optimizations:
```bash
# Use the query_text from the output above
python scripts/identify_optimizations.py --query "SELECT user_id, DATE(timestamp) as date, COUNT(DISTINCT session_id) FROM events GROUP BY user_id, DATE(timestamp)"
```

**Expected Output**:
```json
{
  "anti_patterns": [
    {
      "severity": "MEDIUM",
      "pattern": "EXPENSIVE_DISTINCT",
      "message": "COUNT(DISTINCT) is expensive for large datasets",
      "estimated_savings": "40-60%",
      "fix": "Use approx_distinct() if approximate count is acceptable: COUNT(approx_distinct(session_id))"
    },
    {
      "severity": "HIGH",
      "pattern": "MISSING_PARTITION_FILTER",
      "message": "No partition filter - scanning all historical data",
      "estimated_savings": "70-90%",
      "fix": "Add time filter: WHERE dt >= CURRENT_DATE - INTERVAL '30' DAY"
    }
  ],
  "impact": "Query runs 47 times per week - high optimization ROI"
}
```

**Step 4**: Apply fixes and measure impact:
```sql
-- Optimized version
SELECT
    user_id,
    DATE(timestamp) as date,
    approx_distinct(session_id) as session_count
FROM events
WHERE dt >= CURRENT_DATE - INTERVAL '30' DAY
GROUP BY user_id, DATE(timestamp)
```

**Step 5**: Compare predicted costs:
```bash
python scripts/predict_query_cost.py --query-file optimized_query.sql
```

**Result**: Reduced per-execution cost from $0.18 to $0.02, saving ~$7.50/week (47 executions × $0.16 savings).

## Workflows

### Workflow 1: Cost Prediction Before Execution

Use this when you want to know if a query will be expensive before running it.

1. Get query text from user
2. Run cost prediction:
   ```bash
   python scripts/predict_query_cost.py --query "USER_QUERY"
   ```
3. Review output:
   - Estimated cost in USD
   - Confidence level
   - Complexity score
   - Cost breakdown
   - Recommendations
4. If cost is high (>$1), suggest optimizations before running
5. If user wants optimizations, proceed to Workflow 2

### Workflow 2: Optimize Expensive Query

Use this when a query is too expensive or slow.

1. Analyze query for anti-patterns:
   ```bash
   python scripts/identify_optimizations.py --query "EXPENSIVE_QUERY"
   ```
2. Review findings:
   - Anti-patterns detected (severity levels)
   - Optimization suggestions with examples
   - Potential savings estimates
   - Priority level
3. For each high-severity issue:
   - Explain the problem clearly
   - Show the specific problematic pattern in their query
   - Provide concrete fix with code example
4. If relevant, show query plan analysis:
   ```bash
   python scripts/analyze_query_plan.py --query "EXPENSIVE_QUERY"
   ```
5. Explain key plan features affecting cost (joins, aggregations, scans)

### Workflow 3: Historical Query Analysis

Use this to understand past query resource usage and patterns.

1. Get resource-intensive queries:
   ```bash
   python scripts/query_historical_data.py --expensive --days 7 --limit 10
   ```
2. For interesting queries, get full metrics:
   ```bash
   python scripts/query_historical_data.py --query-id "QUERY_ID"
   ```
3. Compare with prediction to validate model:
   ```bash
   python scripts/predict_query_cost.py --compare "QUERY_ID"
   ```
4. Analyze patterns:
   - Which users/teams have highest resource usage?
   - What query types use most resources?
   - Are there recurring anti-patterns?

### Workflow 4: Resource Usage Trend Analysis

Use this to understand resource consumption patterns over time.

1. Get resource usage statistics for different periods:
   ```bash
   # Last week
   python scripts/query_historical_data.py --stats --days 7

   # Last month
   python scripts/query_historical_data.py --stats --days 30
   ```
2. Filter by user or team:
   ```bash
   python scripts/query_historical_data.py --stats --user "username" --days 30
   ```
3. Identify trends:
   - Is resource usage increasing?
   - Which users consume most resources?
   - What's the average resource usage per query?

## Reference Documentation

For deeper understanding of specific topics, read:

- **Optimization Patterns** - See [optimization-patterns.md](references/optimization-patterns.md) for:
  - Complete list of anti-patterns and fixes
  - Optimization techniques with examples
  - Performance checklist
  - Cost factors for different operations

- **Cost Calculation** - See [cost-calculation.md](references/cost-calculation.md) for:
  - Detailed cost formula components
  - How to build prediction models from historical data
  - Cost optimization ROI estimates
  - Monitoring and alert queries

- **Query Plan Interpretation** - See [query-plan-interpretation.md](references/query-plan-interpretation.md) for:
  - Understanding EXPLAIN output
  - Query plan operators and what they mean
  - How to identify expensive operations
  - Examples of efficient vs inefficient patterns

## Scripts Reference

### predict_query_cost.py

**Purpose**: Predict cost by combining query plan analysis with historical data.

**Usage**:
```bash
# Basic prediction
python scripts/predict_query_cost.py --query "SELECT ..."

# Prediction without historical data (less accurate)
python scripts/predict_query_cost.py --query "SELECT ..." --no-historical

# Compare with actual historical query
python scripts/predict_query_cost.py --compare "query_id"
```

**Output**:
- Estimated cost in USD
- Confidence level (high/medium/low)
- Complexity score
- Cost breakdown by component
- Plan features (joins, scans, aggregations)
- Recommendations for cost reduction
- Historical context (if using historical data)

### identify_optimizations.py

**Purpose**: Identify anti-patterns and suggest optimizations.

**Usage**:
```bash
# Full analysis (gets query plan)
python scripts/identify_optimizations.py --query "SELECT ..."

# Quick analysis (no query plan, faster)
python scripts/identify_optimizations.py --query "SELECT ..." --no-plan

# From file
python scripts/identify_optimizations.py --query-file query.sql
```

**Output**:
- Anti-patterns with severity levels
- Specific recommendations with examples
- Potential savings estimates
- Overall priority level

### analyze_query_plan.py

**Purpose**: Get and parse EXPLAIN output.

**Usage**:
```bash
# Get explain output
python scripts/analyze_query_plan.py --query "SELECT ..."

# Get explain analyze (actually runs query!)
python scripts/analyze_query_plan.py --query "SELECT ..." --analyze

# Parse and estimate cost
python scripts/analyze_query_plan.py --query "SELECT ..." --estimate-cost
```

**Output**:
- EXPLAIN output (raw)
- Parsed plan features
- Cost estimate (if --estimate-cost)

**Warning**: --analyze flag executes the query. Use LIMIT for safety.

### query_historical_data.py

**Purpose**: Query trino_showback table for historical resource usage metrics.

**Usage**:
```bash
# Get resource-intensive queries (sorted by CPU time and data scanned)
python scripts/query_historical_data.py --expensive --days 7 --limit 20

# Get resource usage statistics
python scripts/query_historical_data.py --stats --days 30 --user "username"

# Get specific query metrics
python scripts/query_historical_data.py --query-id "YYYYMMDD_HHMMSS_NNNNN_XXXXX"
```

**Output**: JSON with resource usage metrics (CPU time, memory, data scanned)

**Note**: This script does NOT use the query_cost column from trino_showback. It focuses on actual resource usage metrics like CPU time, memory, and data scanned.

## Connection Configuration

All scripts connect to Trino using these settings (hardcoded):

```python
TRINO_CONFIG = {
    'host': 'trino-querybook-coordinator.de.razorpay.com',
    'port': 443,
    'user': 'root',
    'catalog': 'hive',
    'schema': 'dbt_prod_de_metrics',
    'http_scheme': 'https'
}
```

Historical data is in: `hive.dbt_prod_de_metrics.trino_showback`

## Common Patterns

### Before Running Expensive Query

```bash
# 1. Predict cost
python scripts/predict_query_cost.py --query-file production_query.sql

# 2. If cost is high, analyze for issues
python scripts/identify_optimizations.py --query-file production_query.sql

# 3. Review recommendations and fix issues

# 4. Re-predict after fixes
python scripts/predict_query_cost.py --query-file optimized_query.sql
```

### Investigating Slow Query

```bash
# 1. Get actual resource usage metrics if already ran
python scripts/query_historical_data.py --query-id "QUERY_ID"

# 2. Analyze for optimizations
python scripts/identify_optimizations.py --query "QUERY_TEXT"

# 3. Understand query plan
python scripts/analyze_query_plan.py --query "QUERY_TEXT"

# 4. Compare predicted cost with actual resource usage
python scripts/predict_query_cost.py --compare "QUERY_ID"
```

### Resource Optimization Sprint

```bash
# 1. Find most resource-intensive queries
python scripts/query_historical_data.py --expensive --days 30 --limit 50

# 2. For each resource-intensive query:
#    a. Get query text and metrics
python scripts/query_historical_data.py --query-id "QUERY_ID"

#    b. Analyze for optimizations
python scripts/identify_optimizations.py --query "QUERY_TEXT"

#    c. Estimate savings potential based on resource reduction
#    d. Prioritize by resource usage * frequency

# 3. Fix high-priority queries
# 4. Validate resource usage reduction
```

## Tips

### Cost Thresholds

Use these as guidelines:
- **< $0.01**: Cheap query, safe to run frequently
- **$0.01 - $0.10**: Moderate cost, acceptable for regular use
- **$0.10 - $1.00**: Expensive, review before running
- **> $1.00**: Very expensive, must optimize or verify necessity

### High-Impact Optimizations (Priority Order)

1. **Add partition filters** (70-95% savings) - Always check first
2. **Fix cross joins** (90%+ savings) - Critical if present
3. **Remove SELECT *** (20-50% savings) - Easy win
4. **Use approx_distinct()** (40-60% savings) - When exactness not needed
5. **Add LIMIT for testing** (90%+ savings) - Always during development

### When to Use Each Script

**predict_query_cost.py**:
- Before running new queries
- Comparing alternatives
- Validating optimizations

**identify_optimizations.py**:
- Query is slow or expensive
- Systematic cost reduction
- Code review for cost efficiency

**analyze_query_plan.py**:
- Understanding execution strategy
- Debugging unexpected slowness
- Learning Trino internals

**query_historical_data.py**:
- Resource usage reporting and monitoring
- Identifying top resource consumers
- Validating prediction reasonableness
- Trend analysis of resource consumption

## Limitations

1. **Prediction accuracy**: Cost estimates are based on query plan heuristics and historical resource usage patterns. Actual cost may vary by ±50% depending on data distribution and cluster load.

2. **No actual cost data**: This skill does NOT use the query_cost column from trino_showback. All cost predictions are estimates based on resource usage (CPU, memory, data scanned) and do not reflect actual billing.

3. **Historical data dependency**: Resource usage statistics require sufficient historical data in `trino_showback` table.

4. **Data-dependent**: Query plan analysis doesn't know actual data sizes or distributions. Partition cardinality and data skew can significantly affect actual cost.

5. **Model simplicity**: Cost prediction uses simple heuristics. A machine learning model trained on historical resource usage patterns would be more accurate.

## Troubleshooting

**Script fails with connection error**:
- Verify Trino coordinator is accessible
- Check network connectivity
- Ensure credentials are valid

**No historical data returned**:
- Verify table exists: `SELECT * FROM hive.dbt_prod_de_metrics.trino_showback LIMIT 1`
- Check date range filter
- Ensure resource metrics (cpu_time, peak_mem, input_size) are populated

**Prediction seems inaccurate**:
- Compare with actual resource usage using `--compare` flag
- Check if query has unusual characteristics
- Consider data distribution and skew
- Note: Cost predictions are estimates based on query patterns, not actual billing data

**EXPLAIN ANALYZE fails**:
- Query may have syntax errors
- May not have permissions
- Use LIMIT to prevent expensive execution

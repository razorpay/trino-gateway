# Trino Query Cost Analyzer

A comprehensive toolkit for analyzing Trino query costs, optimizing expensive queries, and predicting costs before execution.

## Features

- **Cost Prediction**: Estimate query costs before execution using query plan analysis and historical data
- **Query Optimization**: Identify anti-patterns and get actionable optimization suggestions
- **Query Plan Analysis**: Parse and understand Trino EXPLAIN output
- **Historical Analysis**: Query resource usage patterns from historical data

## Installation

### From Source

```bash
git clone https://github.com/yourusername/trino-query-cost-analyzer.git
cd trino-query-cost-analyzer
pip install -e .
```

### From PyPI (when published)

```bash
pip install trino-query-cost-analyzer
```

## Quick Start

### Predict Query Cost

```bash
# Estimate cost before running
python scripts/predict_query_cost.py --query "SELECT * FROM table WHERE dt = '2024-01-01'"

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
```

## Configuration

Update the Trino connection settings in each script:

```python
TRINO_CONFIG = {
    'host': 'your-trino-coordinator.example.com',
    'port': 443,
    'user': 'your-username',
    'catalog': 'hive',
    'schema': 'your_schema',
    'http_scheme': 'https'
}
```

## Documentation

- [Complete Usage Guide](SKILL.md) - Comprehensive workflows and examples
- [Optimization Patterns](references/optimization-patterns.md) - Anti-patterns and fixes
- [Cost Calculation](references/cost-calculation.md) - Cost formula and prediction models
- [Query Plan Interpretation](references/query-plan-interpretation.md) - Understanding EXPLAIN output

## Cost Thresholds

Use these as guidelines:
- **< $0.01**: Cheap query, safe to run frequently
- **$0.01 - $0.10**: Moderate cost, acceptable for regular use
- **$0.10 - $1.00**: Expensive, review before running
- **> $1.00**: Very expensive, must optimize or verify necessity

## Common Workflows

### Before Running an Expensive Query

```bash
# 1. Predict cost
python scripts/predict_query_cost.py --query-file production_query.sql

# 2. If cost is high, analyze for issues
python scripts/identify_optimizations.py --query-file production_query.sql

# 3. Review recommendations and fix issues

# 4. Re-predict after fixes
python scripts/predict_query_cost.py --query-file optimized_query.sql
```

### Investigating a Slow Query

```bash
# 1. Get actual resource usage if already ran
python scripts/query_historical_data.py --query-id "QUERY_ID"

# 2. Analyze for optimizations
python scripts/identify_optimizations.py --query "QUERY_TEXT"

# 3. Understand query plan
python scripts/analyze_query_plan.py --query "QUERY_TEXT"
```

## Requirements

- Python 3.8+
- Trino connection access
- Historical data table (optional, for better predictions)

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please open an issue on GitHub.

# PromQL Patterns Reference

## Counter Metrics

Counters only go up. Always use `rate()` or `increase()`.

### Basic Rate

```promql
rate(http_requests_total{job=~"$job"}[$__rate_interval])
```

### Aggregated Rate

```promql
sum(rate(http_requests_total{job=~"$job"}[$__rate_interval])) by (method, status)
```

### Total Increase Over Time Range

```promql
increase(http_requests_total{job=~"$job"}[$__range])
```

## Gauge Metrics

Gauges can go up or down. Use directly or with aggregations.

### Direct Value

```promql
process_resident_memory_bytes{job=~"$job"}
```

### Aggregated Value

```promql
avg(process_resident_memory_bytes{job=~"$job"}) by (instance)
```

### Time Since (for timestamp gauges)

```promql
time() - process_start_time_seconds{job=~"$job"}
```

## Histogram Metrics

Histograms have `_bucket`, `_sum`, and `_count` suffixes.

### Heatmap Query

```promql
sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le)
```

### Percentile (Quantile) Calculation

```promql
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le))
```

### Average Duration

```promql
sum(rate(http_request_duration_seconds_sum{job=~"$job"}[$__rate_interval])) / sum(rate(http_request_duration_seconds_count{job=~"$job"}[$__rate_interval]))
```

### Request Rate from Histogram

```promql
sum(rate(http_request_duration_seconds_count{job=~"$job"}[$__rate_interval]))
```

## Summary Metrics

Summaries have pre-calculated quantiles in a `quantile` label.

### Direct Quantiles

```promql
go_gc_duration_seconds{job=~"$job"}
```

### Aggregated by Quantile

```promql
sum(go_gc_duration_seconds{job=~"$job"}) by (quantile)
```

## Info Metrics

Info metrics expose metadata as labels with value 1.

### Table Display

```promql
build_info{job=~"$job"}
```

Use with `format: "table"` and `instant: true`.

## Common Aggregation Functions

| Function | Description |
|----------|-------------|
| `sum()` | Total across all series |
| `avg()` | Average across all series |
| `max()` | Maximum value |
| `min()` | Minimum value |
| `count()` | Number of series |
| `group()` | Group labels only (returns 1) |
| `topk(n, ...)` | Top N series by value |
| `bottomk(n, ...)` | Bottom N series by value |

## Variable Syntax

| Syntax | Description |
|--------|-------------|
| `$variable` | Simple substitution |
| `${variable}` | Explicit variable reference |
| `$__rate_interval` | Dynamic rate interval (recommended) |
| `$__interval` | Dashboard interval |
| `$__range` | Dashboard time range |
| `job=~"$job"` | Regex match for multi-select variables |

## Label Filtering

```promql
# Exact match
metric{label="value"}

# Regex match (for variables with All option)
metric{label=~"$variable"}

# Negative match
metric{label!="value"}

# Regex negative match
metric{label!~"error|warn"}
```

## Legend Format Examples

| Legend Format | Result |
|---------------|--------|
| `{{method}}` | GET |
| `{{method}} {{status}}` | GET 200 |
| `{{instance}} - {{method}}` | localhost:9090 - GET |
| `__auto` | Automatic labels |

## Rate Interval Best Practice

Always use `$__rate_interval` instead of fixed intervals:

```promql
# Good
rate(metric[$__rate_interval])

# Avoid (may miss data points)
rate(metric[5m])
```

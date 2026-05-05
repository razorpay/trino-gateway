# Prometheus Queries Reference

This document details the Prometheus queries used to fetch CPU and memory usage metrics.

## CPU Usage Query

### Query for P95 CPU per pod (5-minute rate)

```promql
sum(rate(container_cpu_usage_seconds_total{
  namespace="{namespace}",
  container!="POD"
}[2m])) by (pod)
```

**Explanation:**
- `container_cpu_usage_seconds_total` — Cumulative CPU time in seconds
- `rate(...[2m])` — Convert to rate per second (CPU cores used)
- `container!="POD"` — Exclude pod init container
- `by (pod)` — Group by pod name
- Result: CPU cores used per pod (e.g., 0.5 = 500m)

**Alternative (5m rate for smoother data):**
```promql
sum(rate(container_cpu_usage_seconds_total{
  namespace="{namespace}",
  container!=""
}[5m])) by (pod)
```

### Query for per-container breakdown (optional)

```promql
sum(rate(container_cpu_usage_seconds_total{
  namespace="{namespace}",
  container!="POD"
}[2m])) by (pod, container)
```

---

## Memory Usage Query

### Query for working set memory per pod

```promql
sum(container_memory_working_set_bytes{
  namespace="{namespace}"
}) by (pod)
```

**Explanation:**
- `container_memory_working_set_bytes` — Memory in use (similar to RSS in Linux)
- Preferred over `rss` because it's the actual memory the kernel tracks
- Result: Memory in bytes per pod

### Convert bytes to human-readable format

In Python, when fetching results:
```python
memory_bytes = 1073741824
memory_gb = memory_bytes / (1024 ** 3)  # 1.0 GB
memory_mi = memory_bytes / (1024 ** 2)  # 1024 Mi
```

---

## Alternative Memory Metrics

### Memory limit usage (if needed)

```promql
sum(container_spec_memory_limit_bytes{
  namespace="{namespace}"
}) by (pod)
```

### Memory reservation/request

```promql
sum(kube_pod_container_resource_requests{
  namespace="{namespace}",
  resource="memory"
}) by (pod)
```

---

## Data Retrieval Parameters

### Time ranges for Grafana MCP

**For 48-hour history:**
```
start_time: 2024-01-15T10:00:00Z
end_time: 2024-01-17T10:00:00Z
step: 300 (5 minutes)
```

**For 72-hour history:**
```
start_time: 2024-01-14T10:00:00Z
end_time: 2024-01-17T10:00:00Z
step: 300 (5 minutes)
```

---

## Query Execution via Grafana MCP

Example call:

```python
# Fetch CPU metrics
promql_query = """
sum(rate(container_cpu_usage_seconds_total{
  namespace="settlements",
  container!="POD"
}[2m])) by (pod)
"""

# Call Grafana MCP with:
# - datasource_uid: "6ZssswRnk" (primary Prometheus)
# - query: promql_query
# - start: 48 hours ago
# - end: now
# - step: 300 (5 minutes)

response = grafana_mcp.range_query(
    datasource_uid="6ZssswRnk",
    query=promql_query,
    start="now-48h",
    end="now",
    step=300
)
```

---

## Expected Response Format

Grafana MCP returns time-series data:

```json
{
  "status": "success",
  "data": {
    "resultType": "matrix",
    "result": [
      {
        "metric": {
          "pod": "payments-api-live-1"
        },
        "values": [
          [1705329600, "0.5"],
          [1705329900, "0.52"],
          [1705330200, "0.48"],
          ...
        ]
      },
      {
        "metric": {
          "pod": "payments-api-live-2"
        },
        "values": [
          ...
        ]
      }
    ]
  }
}
```

**Parsing:**
- Each entry in `values` is `[unix_timestamp, "value_string"]`
- Convert string to float: `float("0.5")` = 0.5 cores

---

## Calculating P95 Percentile

Once you have all data points for a pod:

```python
import numpy as np

cpu_values = [0.48, 0.50, 0.52, 0.49, 0.75, 0.80, 1.2, ...]
p95_cpu = np.percentile(cpu_values, 95)
# Result: 0.95 cores = 950m
```

Or without numpy (for simpler implementation):

```python
def percentile(data, p):
    """Calculate pth percentile"""
    sorted_data = sorted(data)
    index = int(len(sorted_data) * p / 100)
    return sorted_data[min(index, len(sorted_data) - 1)]

p95 = percentile(cpu_values, 95)
```

---

## Handling Missing Data

**If namespace has no metrics:**
- Return NULL for usage values
- Recommend user to check if pods are running and metrics are being scraped
- Show only requested/limit values from pod spec

**If pod has <1 hour of data:**
- Exclude from analysis (too new to be reliable)
- Note in report: "Pod X created recently, skipping usage analysis"

**If time range returns no results:**
- Retry with shorter time range (e.g., 24h instead of 48h)
- Or use average of available data

---

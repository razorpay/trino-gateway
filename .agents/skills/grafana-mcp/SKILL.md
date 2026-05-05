---
name: grafana-mcp
description: Query Prometheus metrics via Grafana MCP across Razorpay cells (IN, SG, US) with seamless SAML SSO. Use when users ask to "query prometheus", "check grafana metrics", "run promql", "list metrics", "compare metrics across cells", "investigate service health", "check latency/error rate/throughput", "pull dashboard metrics", or need live Prometheus data from any Grafana cell.
---

# Grafana MCP — Live Prometheus Query Skill

Query live Prometheus metrics across Razorpay's three Grafana cells using the `grafana` MCP server tools. Authentication is seamless — browser opens for SAML SSO automatically.

## Cells

| Cell | Grafana Instance | Use for |
|------|-----------------|---------|
| `in` | vajra.razorpay.com | India production |
| `sg` | grafana-sg.razorpay.com | Singapore production |
| `us` | grafana-us.razorpay.com | US production |

## Prerequisites

The `grafana` MCP server must be registered in `~/.claude.json`:

```json
{
  "mcpServers": {
    "grafana": {
      "type": "stdio",
      "command": "python",
      "args": ["/path/to/mcp-grafana/server.py"],
      "env": {}
    }
  }
}
```

Dependencies: `pip install mcp requests playwright && python -m playwright install chromium`

## Workflow

ALWAYS follow this sequence when handling a metrics query:

### Step 1: Determine cell and validate auth

- Default to `in` cell unless user specifies otherwise
- If user provides a Grafana URL, extract the cell:
  - `vajra.razorpay.com` → `in`
  - `grafana-sg.razorpay.com` → `sg`
  - `grafana-us.razorpay.com` → `us`
- Auth is automatic — if session is missing or expired, browser opens for SAML. No action needed.

### Step 2: Discover datasource ID

Before any PromQL query, you MUST get the datasource ID:

```
→ call list_datasources(cell="in")
← returns: [{"id": 1, "uid": "abc", "name": "Prometheus", "type": "prometheus"}, ...]
```

Pick the correct datasource based on user context. If multiple Prometheus datasources exist, ask which one. Cache the `id` for subsequent queries in the same conversation.

### Step 3: Run queries

Use the appropriate tool based on what the user needs:

| Need | Tool | Key params |
|------|------|-----------|
| Current value | `query_instant` | `expr`, optional `time` |
| Time series / trends | `query_range` | `expr`, `start`, `end`, `step` |
| What metrics exist | `list_metrics` | optional `match` filter |
| Find a dashboard | `search_dashboards` | `query` search term |
| Read dashboard queries | `get_dashboard` | `uid` from URL or search |

### Step 4: Interpret and present results

- Format numbers: use K/M/B suffixes, round to 2 decimal places
- For time series: summarize trend (increasing, stable, spiky) and call out min/max/avg
- For comparisons: present as a table with cell/metric columns
- Flag anomalies: sudden spikes, values at zero, NaN results

## PromQL Patterns

Use these standard patterns. NEVER query raw counters — always wrap with `rate()` or `increase()`.

### Request Rate (throughput)
```promql
sum(rate(http_requests_total{job="<service>"}[5m])) by (status_code)
```

### Error Rate
```promql
sum(rate(http_requests_total{job="<service>",status_code=~"5.."}[5m]))
/ sum(rate(http_requests_total{job="<service>"}[5m])) * 100
```

### P99 Latency
```promql
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job="<service>"}[5m])) by (le))
```

### P50 / P90 / P99 Comparison
```promql
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket{job="<service>"}[5m])) by (le))
histogram_quantile(0.90, sum(rate(http_request_duration_seconds_bucket{job="<service>"}[5m])) by (le))
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job="<service>"}[5m])) by (le))
```

### CPU Usage
```promql
100 - (avg(rate(node_cpu_seconds_total{mode="idle",instance=~"<host>"}[5m])) * 100)
```

### Memory Usage %
```promql
(1 - node_memory_MemAvailable_bytes{instance=~"<host>"} / node_memory_MemTotal_bytes{instance=~"<host>"}) * 100
```

### Goroutine Count (Go services)
```promql
go_goroutines{job="<service>"}
```

### Kafka Consumer Lag
```promql
sum(kafka_consumer_group_lag{group="<consumer-group>"}) by (topic)
```

### Pod Restart Count
```promql
increase(kube_pod_container_status_restarts_total{namespace="<ns>",pod=~"<service>.*"}[1h])
```

## Step Selection for Range Queries

Match `step` to the time range to get ~100-200 data points:

| Time range | Step |
|-----------|------|
| Last 15m | `15s` |
| Last 1h | `30s` |
| Last 6h | `3m` |
| Last 24h | `15m` |
| Last 7d | `1h` |
| Last 30d | `6h` |

## Common Workflows

### "Check health of service X"

1. `list_datasources(cell="in")` → get datasource_id
2. Run three queries in parallel:
   - Rate: `sum(rate(http_requests_total{job="X"}[5m]))`
   - Errors: `sum(rate(http_requests_total{job="X",status_code=~"5.."}[5m]))`
   - P99: `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job="X"}[5m])) by (le))`
3. Present as RED summary table

### "Compare service X across IN and SG"

1. `list_datasources(cell="in")` → get IN datasource_id
2. `list_datasources(cell="sg")` → get SG datasource_id
3. Run same PromQL on both cells
4. Present side-by-side comparison table

### "What happened during incident at TIME?"

1. `list_datasources(cell)` → get datasource_id
2. Run `query_range` with tight window around incident time:
   - Error rate with `step=15s`
   - Latency percentiles with `step=15s`
   - Throughput with `step=15s`
3. Identify the spike/drop timestamp and magnitude

### "Pull metrics from this dashboard URL"

Extract dashboard UID from URL (the part after `/d/`):
- `vajra.razorpay.com/d/yLloK0Kik/prod-shield` → uid = `yLloK0Kik`, cell = `in`
- `grafana-sg.razorpay.com/d/TbaMXnWGz/kafka-connect` → uid = `TbaMXnWGz`, cell = `sg`

1. `get_dashboard(cell, uid)` → get all panel queries
2. Run each panel's PromQL expression via `query_range`
3. Present results panel by panel

### "What metrics does service X expose?"

1. `list_datasources(cell)` → get datasource_id
2. `list_metrics(cell, datasource_id, match='{job="X"}')` → all metric names for that job
3. Group by naming convention:
   - `*_total` → counters (use with `rate()`)
   - `*_bucket` → histograms (use with `histogram_quantile()`)
   - `*_bytes`, `*_seconds` → gauges with units

## Response Format

When presenting query results:

**For instant queries:**
```
Metric: http_requests_total
Cell: IN | Value: 1,234.56 req/s
Labels: {job="api-server", status="200"}
```

**For range queries:**
```
Metric: error_rate (last 1h, step=30s)
Cell: IN
  Avg: 0.12% | Max: 2.34% (at 14:23 UTC) | Min: 0.01%
  Trend: stable with spike at 14:23
```

**For cross-cell comparison:**
```
| Metric          | IN      | SG      | US      |
|-----------------|---------|---------|---------|
| Request Rate    | 12.3K/s | 3.1K/s  | 8.7K/s  |
| Error Rate      | 0.12%   | 0.08%   | 0.15%   |
| P99 Latency     | 245ms   | 312ms   | 198ms   |
```

## Error Handling

| Error | Meaning | Action |
|-------|---------|--------|
| 401 Unauthorized | Session expired | Auto-relogin triggers. Wait for browser. |
| 403 Forbidden | No datasource access | Tell user to check Grafana permissions |
| 422 / bad_data | Invalid PromQL | Fix the expression syntax |
| No data | Metric doesn't exist or wrong labels | Try `list_metrics` to discover correct names |
| Timeout | Query too expensive | Add more label filters or reduce time range |

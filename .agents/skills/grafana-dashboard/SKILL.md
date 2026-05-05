---
name: grafana-dashboard
description: Generate production-ready Grafana dashboards from Prometheus metrics. Outputs JSON that can be directly pasted into Grafana's import menu. Use when users ask to "create grafana dashboard", "generate dashboard for metrics", "visualize prometheus metrics", "build grafana panels", or need to convert Prometheus metrics into Grafana visualizations.
---

# Grafana Dashboard Creator

Generate production-ready Grafana dashboard JSON from Prometheus metrics using industry-standard observability methodologies (USE and RED).

## Observability Methodologies

This skill follows two complementary monitoring methodologies that together provide complete system observability:

### RED Method (Services/Requests)

For every **service**, monitor these request-scoped metrics:

| Metric | Description | Focus |
|--------|-------------|-------|
| **R**ate | Requests per second | Throughput |
| **E**rrors | Failed requests per second | Reliability |
| **D**uration | Latency distribution (p50, p90, p99) | Performance |

**When to use:** Microservices, APIs, web applications - anywhere you handle requests. RED answers "How are my users experiencing my service?"

### USE Method (Resources/Infrastructure)

For every **resource**, monitor these system-scoped metrics:

| Metric | Description | Focus |
|--------|-------------|-------|
| **U**tilization | % time resource is busy | Capacity |
| **S**aturation | Queue depth / backlog | Bottlenecks |
| **E**rrors | Error count | Health |

**When to use:** CPUs, memory, storage, network, databases - infrastructure components. USE answers "How are my resources handling the workload?"

### Choosing a Methodology

| Dashboard Type | Primary Method | Secondary |
|----------------|----------------|-----------|
| API/Service | RED | USE for dependencies |
| Infrastructure | USE | - |
| Full Stack | Both | Separate rows |
| Database | USE + RED | Query rate + resource usage |

## Workflow

### 1. Understand Request

Determine the dashboard type and methodology:

| Method | Description |
|--------|-------------|
| Metrics endpoint | User provides a `/metrics` URL to fetch and parse |
| Direct list | User lists metric names, types, and labels |
| Service dashboard | Use RED method - focus on request metrics |
| Infrastructure dashboard | Use USE method - focus on resource metrics |
| Full-stack dashboard | Combine both methodologies |

### 2. Collect Metrics

For each metric, gather:
- **Name**: The metric name (e.g., `http_requests_total`)
- **Type**: counter, gauge, histogram, summary, or info
- **Labels**: Available labels for filtering/grouping

If type is not specified, infer from naming:
- `*_total` suffix → counter
- `*_info`, `*_labels` suffix → info
- `*_bucket` suffix → histogram
- Has `quantile` label → summary
- `*_time`, `*_timestamp`, `*_started` → gauge (time-based)
- Otherwise → gauge

### 3. Configure Panels

Ask user about preferences:
- **Methodology**: RED (services), USE (infrastructure), or both
- Aggregation function (sum, avg, max, min, count)
- Grouping dimensions (which labels to `by()`)
- Row organization (how to group panels)

### 4. Organize by Methodology

Structure dashboards using collapsible rows based on the chosen methodology:

**RED Dashboard Layout:**
```
Row: Rate (Request Throughput)
  - Requests/sec by endpoint
  - Requests/sec by method
Row: Errors
  - Error rate by endpoint
  - Error rate by status code
  - Error ratio (errors/total)
Row: Duration (Latency)
  - P50/P90/P99 latency
  - Latency heatmap
  - Average response time
```

**USE Dashboard Layout:**
```
Row: CPU
  - Utilization %
  - Saturation (load average, run queue)
  - Errors (if available)
Row: Memory
  - Utilization (used/total)
  - Saturation (swap usage, OOM events)
Row: Storage
  - Disk utilization %
  - I/O saturation (queue depth)
  - I/O errors
Row: Network
  - Bandwidth utilization
  - Saturation (dropped packets)
  - Network errors
```

### 5. Generate Dashboard

Build the dashboard JSON using the structures defined in `references/dashboard-structure.md`.

### 6. Output

Display the complete dashboard JSON to stdout. User can copy/paste or redirect to file.

## Metric Type to Panel Mapping

| Metric Type | Panel Type | PromQL Pattern |
|-------------|------------|----------------|
| counter | Time Series | `rate(metric{selector}[$__rate_interval])` |
| gauge | Time Series | `metric{selector}` or `time() - metric` for timestamps |
| histogram | Heatmap | `sum(rate(metric_bucket{sel}[$__rate_interval])) by (le)` |
| summary | Time Series | `sum(metric{selector}) by (quantile)` |
| info | Table | `metric{selector}` (instant query) |

## Panel Defaults

```
Width: 8 (1-12 grid units)
Height: 5 (1-12 grid units)
LineWidth: 1 (0-10)
Fill: 1 (0-100 opacity)
Stack: false
Scale: "linear" (linear, log2, log10)
LegendPosition: "bottom" (bottom, right, hide)
LegendCalcs: ["max", "avg", "last"]
```

## Unit Detection

Auto-detect units from metric name suffix:

| Pattern | Grafana Unit |
|---------|--------------|
| `_seconds` | `s` |
| `_bytes` | `decbytes` |
| `_volt` | `volt` |
| `_ampere` | `amp` |
| `_hertz` | `rothz` |
| `_celsius` | `celsius` |

## Example Interactions

### RED Dashboard (Service Monitoring)

```
User: Create a dashboard for my API service

Claude: I'll create a RED dashboard for your API. This will show:
- Rate: Request throughput
- Errors: Failed requests
- Duration: Latency distribution

What metrics do you have? Common patterns:
- http_requests_total, http_request_duration_seconds
- Or provide a /metrics endpoint URL

User: I have these metrics:
- http_requests_total (counter) with labels: method, status, endpoint
- http_request_duration_seconds (histogram)

Claude: I'll organize this as a RED dashboard:

**Row: Rate**
- Request Rate: `sum(rate(http_requests_total{job=~"$job"}[$__rate_interval])) by (endpoint)`
- Request Rate by Method: `sum(rate(http_requests_total{job=~"$job"}[$__rate_interval])) by (method)`

**Row: Errors**
- Error Rate: `sum(rate(http_requests_total{job=~"$job",status=~"5.."}[$__rate_interval]))`
- Error Ratio: `sum(rate(http_requests_total{status=~"5.."}[$__rate_interval])) / sum(rate(http_requests_total[$__rate_interval])) * 100`

**Row: Duration**
- P99 Latency: `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le))`
- Latency Heatmap: `sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le)`

Generate dashboard JSON?

User: Yes

Claude: [Outputs complete dashboard JSON]
```

### USE Dashboard (Infrastructure Monitoring)

```
User: Create a dashboard for my server resources

Claude: I'll create a USE dashboard for infrastructure monitoring:
- Utilization: How busy are resources?
- Saturation: Is work queuing up?
- Errors: Any resource errors?

What resources to monitor? (CPU, Memory, Disk, Network)

User: All of them - I'm using node_exporter

Claude: I'll create a USE dashboard with node_exporter metrics:

**Row: CPU**
- Utilization: `100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[$__rate_interval])) * 100)`
- Saturation: `node_load1` (1-min load average)

**Row: Memory**
- Utilization: `(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100`
- Saturation: `node_memory_SwapFree_bytes / node_memory_SwapTotal_bytes`

**Row: Disk**
- Utilization: `rate(node_disk_io_time_seconds_total[$__rate_interval]) * 100`
- Saturation: `node_disk_io_time_weighted_seconds_total`

**Row: Network**
- Utilization: `rate(node_network_receive_bytes_total[$__rate_interval])`
- Saturation: `rate(node_network_receive_drop_total[$__rate_interval])`

Generate dashboard JSON?
```

## Resources

- `references/dashboard-structure.md` - Complete JSON templates for dashboards, panels, and variables
- `references/promql-patterns.md` - PromQL query patterns for each metric type
- `references/use-red-methods.md` - USE and RED methodology reference with common metrics

## References

- [The USE Method](https://www.brendangregg.com/usemethod.html) - Brendan Gregg's original methodology
- [The RED Method](https://grafana.com/blog/2018/08/02/the-red-method-how-to-instrument-your-services/) - Tom Wilkie's service monitoring approach
- [RED and USE Metrics Guide](https://betterstack.com/community/guides/monitoring/red-use-metrics/) - Practical implementation guide

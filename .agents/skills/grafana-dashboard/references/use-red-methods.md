# USE and RED Methods Reference

## Overview

Two complementary methodologies for comprehensive observability:

| Method | Focus | Answers |
|--------|-------|---------|
| RED | Services/Requests | "How are users experiencing my service?" |
| USE | Resources/Infrastructure | "How are resources handling the workload?" |

---

## RED Method (Rate, Errors, Duration)

### When to Use

Apply RED to any service that handles requests:
- HTTP APIs and web services
- gRPC services
- Message queue consumers
- Database query layers
- Microservices

### Core Metrics

#### Rate (R) - Request Throughput

| Metric | PromQL Pattern |
|--------|----------------|
| Total request rate | `sum(rate(http_requests_total{job=~"$job"}[$__rate_interval]))` |
| Rate by endpoint | `sum(rate(http_requests_total{job=~"$job"}[$__rate_interval])) by (endpoint)` |
| Rate by method | `sum(rate(http_requests_total{job=~"$job"}[$__rate_interval])) by (method)` |
| Rate by status | `sum(rate(http_requests_total{job=~"$job"}[$__rate_interval])) by (status)` |

**Panel:** Time Series, unit: `reqps`

#### Errors (E) - Failed Requests

| Metric | PromQL Pattern |
|--------|----------------|
| Error rate (5xx) | `sum(rate(http_requests_total{job=~"$job",status=~"5.."}[$__rate_interval]))` |
| Error rate (4xx) | `sum(rate(http_requests_total{job=~"$job",status=~"4.."}[$__rate_interval]))` |
| Error ratio % | `sum(rate(http_requests_total{status=~"5.."}[$__rate_interval])) / sum(rate(http_requests_total[$__rate_interval])) * 100` |
| Errors by type | `sum(rate(http_requests_total{job=~"$job",status=~"[45].."}[$__rate_interval])) by (status)` |

**Panel:** Time Series with thresholds (green < 1%, yellow < 5%, red >= 5%)

#### Duration (D) - Latency Distribution

| Metric | PromQL Pattern |
|--------|----------------|
| P50 latency | `histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le))` |
| P90 latency | `histogram_quantile(0.90, sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le))` |
| P99 latency | `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le))` |
| Average latency | `sum(rate(http_request_duration_seconds_sum[$__rate_interval])) / sum(rate(http_request_duration_seconds_count[$__rate_interval]))` |
| Latency heatmap | `sum(rate(http_request_duration_seconds_bucket{job=~"$job"}[$__rate_interval])) by (le)` |

**Panel:** Time Series for percentiles (unit: `s`), Heatmap for distribution

### RED Dashboard Structure

```
Row: Rate
  Panel: Request Rate (time series, stacked by endpoint)
  Panel: Request Rate by Method (time series)
  Panel: Current RPS (stat panel, sparkline)

Row: Errors
  Panel: Error Rate (time series, 4xx vs 5xx)
  Panel: Error Ratio % (time series with threshold lines)
  Panel: Current Error Rate (stat panel with thresholds)

Row: Duration
  Panel: Latency Percentiles (time series, p50/p90/p99 lines)
  Panel: Latency Heatmap (heatmap)
  Panel: Average Response Time (stat panel)
```

### Common RED Metric Names

| Framework/Tool | Request Counter | Duration Histogram |
|----------------|-----------------|-------------------|
| Go net/http | `http_requests_total` | `http_request_duration_seconds` |
| Prometheus client | `http_requests_total` | `http_request_duration_seconds` |
| Spring Boot | `http_server_requests_seconds_count` | `http_server_requests_seconds` |
| Express.js | `http_request_duration_seconds` | `http_request_duration_seconds` |
| Django | `django_http_requests_total` | `django_http_request_duration_seconds` |
| gRPC | `grpc_server_handled_total` | `grpc_server_handling_seconds` |

---

## USE Method (Utilization, Saturation, Errors)

### When to Use

Apply USE to infrastructure resources:
- CPUs and cores
- Memory
- Storage devices
- Network interfaces
- Database connections
- Thread pools

### Resource Checklist

#### CPU

| Metric Type | What to Measure | PromQL (node_exporter) |
|-------------|-----------------|------------------------|
| Utilization | % time busy | `100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[$__rate_interval])) * 100)` |
| Saturation | Run queue length | `node_load1` or `node_load5` |
| Saturation | Scheduler latency | `rate(node_schedstat_waiting_seconds_total[$__rate_interval])` |
| Errors | CPU errors | Hardware-specific (rare in metrics) |

#### Memory

| Metric Type | What to Measure | PromQL (node_exporter) |
|-------------|-----------------|------------------------|
| Utilization | % used | `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100` |
| Saturation | Swap usage | `(1 - (node_memory_SwapFree_bytes / node_memory_SwapTotal_bytes)) * 100` |
| Saturation | Page faults | `rate(node_vmstat_pgmajfault[$__rate_interval])` |
| Errors | OOM events | `increase(node_vmstat_oom_kill[$__rate_interval])` |

#### Storage (Disk I/O)

| Metric Type | What to Measure | PromQL (node_exporter) |
|-------------|-----------------|------------------------|
| Utilization | % time busy | `rate(node_disk_io_time_seconds_total{device!~"dm-.*"}[$__rate_interval]) * 100` |
| Saturation | Queue depth | `rate(node_disk_io_time_weighted_seconds_total[$__rate_interval])` |
| Saturation | Wait time | `rate(node_disk_read_time_seconds_total[$__rate_interval]) / rate(node_disk_reads_completed_total[$__rate_interval])` |
| Errors | I/O errors | `rate(node_disk_io_errors_total[$__rate_interval])` (if available) |

#### Storage (Capacity)

| Metric Type | What to Measure | PromQL (node_exporter) |
|-------------|-----------------|------------------------|
| Utilization | % used | `(1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)) * 100` |
| Saturation | Inode usage | `(1 - (node_filesystem_files_free / node_filesystem_files)) * 100` |

#### Network

| Metric Type | What to Measure | PromQL (node_exporter) |
|-------------|-----------------|------------------------|
| Utilization | Bandwidth used | `rate(node_network_receive_bytes_total{device!~"lo|veth.*"}[$__rate_interval]) * 8` |
| Saturation | Dropped packets | `rate(node_network_receive_drop_total[$__rate_interval])` |
| Saturation | Queue overruns | `rate(node_network_receive_fifo_total[$__rate_interval])` |
| Errors | Receive errors | `rate(node_network_receive_errs_total[$__rate_interval])` |
| Errors | Transmit errors | `rate(node_network_transmit_errs_total[$__rate_interval])` |

### USE Dashboard Structure

```
Row: CPU
  Panel: CPU Utilization % (time series, by core or avg)
  Panel: Load Average (time series, 1m/5m/15m lines)
  Panel: Current CPU % (stat with thresholds)

Row: Memory
  Panel: Memory Utilization % (time series)
  Panel: Swap Usage (time series)
  Panel: Current Memory % (stat with thresholds)
  Panel: OOM Events (stat, show if > 0)

Row: Disk I/O
  Panel: Disk Utilization % (time series, by device)
  Panel: I/O Queue Depth (time series)
  Panel: Disk Throughput (time series, read/write)

Row: Disk Space
  Panel: Filesystem Usage % (time series, by mount)
  Panel: Inode Usage % (time series)

Row: Network
  Panel: Network Traffic (time series, rx/tx)
  Panel: Dropped Packets (time series)
  Panel: Network Errors (time series)
```

### Interpretation Guidelines

| Condition | Meaning | Action |
|-----------|---------|--------|
| High utilization (>70%) | Resource is busy | May hide bursts; investigate further |
| 100% utilization | Bottleneck | Immediate investigation needed |
| Any saturation (>0) | Work is queuing | Resource cannot keep up |
| Increasing errors | Health degradation | Investigate root cause |

---

## Combined Dashboard (Full-Stack)

For complete visibility, combine both methods:

```
=== Service Layer (RED) ===
Row: Request Rate
Row: Error Rate
Row: Latency

=== Infrastructure Layer (USE) ===
Row: CPU (Utilization, Saturation)
Row: Memory (Utilization, Saturation)
Row: Disk (Utilization, Saturation, Errors)
Row: Network (Utilization, Saturation, Errors)
```

### Correlation Tips

1. **High latency + High CPU utilization** → CPU bottleneck
2. **High latency + High disk saturation** → I/O bottleneck
3. **Errors increasing + Memory saturation** → OOM or memory pressure
4. **Rate dropping + Network saturation** → Network bottleneck
5. **Errors increasing + No resource issues** → Application bug

---

## Quick Reference Card

### RED (for Services)

```
Rate:     sum(rate(requests_total[$__rate_interval])) by (endpoint)
Errors:   sum(rate(requests_total{status=~"5.."}[$__rate_interval])) / sum(rate(requests_total[$__rate_interval]))
Duration: histogram_quantile(0.99, sum(rate(request_duration_bucket[$__rate_interval])) by (le))
```

### USE (for Resources)

```
CPU Utilization:    100 - avg(rate(cpu_seconds_total{mode="idle"}[$__rate_interval])) * 100
CPU Saturation:     node_load1 / count(node_cpu_seconds_total{mode="idle"})
Memory Utilization: (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100
Memory Saturation:  rate(node_vmstat_pgmajfault[$__rate_interval])
Disk Utilization:   rate(node_disk_io_time_seconds_total[$__rate_interval]) * 100
Disk Saturation:    rate(node_disk_io_time_weighted_seconds_total[$__rate_interval])
```

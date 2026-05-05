# Grafana Query Templates

This document provides PromQL query templates for monitoring deployments.

## Datasource Configuration

**Required Datasource:**
- **Name:** Victoria Metrics (`vm-new-v1.61.1`)
- **UID:** `ALvd9Tgnz`
- **Type:** Prometheus-compatible

**Important:** All queries in this document should be executed against the Victoria Metrics datasource.

## Metric Selection by Service Type

**For Razorpay Foundation Framework Services (gRPC):**
- Use `grpc_server_*` metrics for incoming requests
- Use `grpc_code` label for status codes (OK, InvalidArgument, Internal, etc.)
- The `http_requests_total` metric tracks **outgoing** HTTP calls to external services

**For HTTP/REST Services:**
- Use `http_requests_total` for incoming HTTP requests
- Use `status` label for HTTP status codes (200, 404, 500, etc.)

## gRPC Server Metrics (Razorpay Foundation Framework)

### Error Rate

**Current error rate (all errors):**
```promql
sum by (grpc_method, grpc_code) (rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>",
  grpc_code!="OK"
}[5m]))
```

**5xx equivalent errors (server-side failures):**
```promql
sum by (grpc_method) (rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>",
  grpc_code=~"Internal|Unavailable|DataLoss|Unknown"
}[5m]))
```

**4xx equivalent errors (client errors):**
```promql
sum by (grpc_method, grpc_code) (rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>",
  grpc_code=~"InvalidArgument|NotFound|AlreadyExists|PermissionDenied|FailedPrecondition|OutOfRange|Unauthenticated"
}[5m]))
```

**Error percentage:**
```promql
(sum(rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>",
  grpc_code!="OK"
}[5m])) /
sum(rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>"
}[5m]))) * 100
```

### Latency Metrics

**P95 latency:**
```promql
histogram_quantile(0.95,
  sum by (le, grpc_method) (rate(grpc_server_handled_duration_seconds_bucket{
    kubernetes_namespace="<namespace>"
  }[5m]))
)
```

**P99 latency:**
```promql
histogram_quantile(0.99,
  sum by (le, grpc_method) (rate(grpc_server_handled_duration_seconds_bucket{
    kubernetes_namespace="<namespace>"
  }[5m]))
)
```

**Average latency:**
```promql
avg by (grpc_method) (
  rate(grpc_server_handled_duration_seconds_sum{
    kubernetes_namespace="<namespace>"
  }[5m]) /
  rate(grpc_server_handled_duration_seconds_count{
    kubernetes_namespace="<namespace>"
  }[5m])
)
```

### Request Rate

**Requests per second (total):**
```promql
sum(rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>"
}[5m]))
```

**Requests per second by method:**
```promql
sum by (grpc_method) (rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>"
}[5m]))
```

**Successful requests per second:**
```promql
sum by (grpc_method) (rate(grpc_server_handled_total{
  kubernetes_namespace="<namespace>",
  grpc_code="OK"
}[5m]))
```

## HTTP Request Metrics

### 5xx Error Rate

**Current error rate (last 5 minutes):**
```promql
sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m])) by (route)
```

**Error rate comparison (now vs 24h ago):**
```promql
# Current
sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m])) by (route)

# 24 hours ago
sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m] offset 24h)) by (route)
```

**Error percentage:**
```promql
(sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m])) by (route) /
 sum(rate(http_requests_total{route="<route>"}[5m])) by (route)) * 100
```

### Latency Metrics

**P95 latency:**
```promql
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m])) by (le, route)
)
```

**P99 latency:**
```promql
histogram_quantile(0.99,
  sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m])) by (le, route)
)
```

**Average latency:**
```promql
avg(rate(http_request_duration_seconds_sum{route="<route>"}[5m]) /
    rate(http_request_duration_seconds_count{route="<route>"}[5m])) by (route)
```

**Latency comparison (now vs 24h ago):**
```promql
# Current P95
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m])) by (le, route)
)

# P95 24 hours ago
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m] offset 24h)) by (le, route)
)
```

### Request Rate

**Requests per second:**
```promql
sum(rate(http_requests_total{route="<route>"}[5m])) by (route)
```

**Request rate change:**
```promql
# Current rate
sum(rate(http_requests_total{route="<route>"}[5m])) by (route)

# Rate 24h ago
sum(rate(http_requests_total{route="<route>"}[5m] offset 24h)) by (route)
```

## Service-Specific Metrics

### Razorpay Services (Common Patterns)

**Error rate by service:**
```promql
sum(rate(grpc_server_handled_total{grpc_code!="OK", grpc_service="<service>"}[5m])) by (grpc_method)
```

**gRPC latency:**
```promql
histogram_quantile(0.95,
  sum(rate(grpc_server_handling_seconds_bucket{grpc_service="<service>"}[5m])) by (le, grpc_method)
)
```

**Database query latency:**
```promql
histogram_quantile(0.95,
  sum(rate(db_query_duration_seconds_bucket{query_name="<query>"}[5m])) by (le)
)
```

## Kubernetes Metrics

### Pod Health

**Pod restarts (last 30 minutes):**
```promql
changes(kube_pod_container_status_restarts_total{namespace="<namespace>", pod=~"<deployment>.*"}[30m])
```

**CPU usage:**
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="<namespace>", pod=~"<deployment>.*"}[5m])) by (pod)
```

**Memory usage:**
```promql
sum(container_memory_usage_bytes{namespace="<namespace>", pod=~"<deployment>.*"}) by (pod)
```

## Alerting Queries

### Threshold-Based Alerts

**5xx error spike (>20% increase):**
```promql
(
  sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m])) by (route)
  /
  sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m] offset 24h)) by (route)
) > 1.2
```

**Latency degradation (>30% increase):**
```promql
(
  histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m])) by (le, route))
  /
  histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{route="<route>"}[5m] offset 24h)) by (le, route))
) > 1.3
```

## Time Windows

**Short-term (deployment monitoring):**
- Use `[5m]` for rate calculations
- Compare against 24h ago for baseline

**Medium-term (trend analysis):**
- Use `[15m]` or `[30m]` for smoothing
- Compare against 7d ago for weekly patterns

**Long-term (capacity planning):**
- Use `[1h]` or `[6h]` for aggregation
- Compare against 30d ago for monthly trends

## Dashboard Integration

### Using Grafana API

**Query metrics programmatically:**

```python
# Example: Query 5xx errors via Grafana API
GET /api/datasources/proxy/:datasourceId/api/v1/query

{
  "query": "sum(rate(http_requests_total{status=~\"5..\", route=\"/v1/offers\"}[5m]))",
  "time": "<timestamp>"
}
```

**Get dashboard panel data:**

```python
# Use grafana MCP server
get_dashboard_panel_queries:
  dashboard_uid: "<dashboard-uid>"

# Then execute each query to get current values
query_prometheus:
  query: "<extracted-query>"
  time: "now"
```

## Common Metric Names

| Service Type | Error Metric | Latency Metric | Notes |
|-------------|--------------|----------------|-------|
| **gRPC (Razorpay Foundation)** | `grpc_server_handled_total{grpc_code!="OK"}` | `grpc_server_handled_duration_seconds` | **Primary metric for incoming requests** |
| HTTP API | `http_requests_total{status=~"5.."}` | `http_request_duration_seconds` | For HTTP/REST services |
| HTTP External Calls | `affordability_http_requests_count` | `affordability_http_request_duration_ms_hist` | **Outgoing** HTTP calls to external services |
| Database | `db_errors_total` | `db_query_duration_seconds` | Database operations |
| Cache | `cache_misses_total` | `cache_operation_duration_seconds` | Cache operations |

**Important:**
- For Razorpay Foundation framework services, use `grpc_server_*` metrics for incoming requests
- The `affordability_http_requests_count` metric tracks **outgoing** HTTP calls to external services, NOT incoming requests

## Example: Complete Route Check

For route `POST /v1/offers/create`:

```promql
# 1. Current 5xx error rate
sum(rate(http_requests_total{status=~"5..", route="/v1/offers/create", method="POST"}[5m]))

# 2. Baseline error rate (24h ago)
sum(rate(http_requests_total{status=~"5..", route="/v1/offers/create", method="POST"}[5m] offset 24h))

# 3. Current P95 latency
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{route="/v1/offers/create", method="POST"}[5m])) by (le)
)

# 4. Baseline P95 latency (24h ago)
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{route="/v1/offers/create", method="POST"}[5m] offset 24h)) by (le)
)

# 5. Request rate
sum(rate(http_requests_total{route="/v1/offers/create", method="POST"}[5m]))
```

**Interpretation:**
- Error rate: 0.05 vs 0.03 → 67% increase ⚠️
- P95 latency: 150ms vs 120ms → 25% increase ⚠️
- Request rate: 50 req/s vs 48 req/s → 4% increase ✅

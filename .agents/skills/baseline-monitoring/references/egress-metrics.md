# Egress Metrics

Read this file when the service calls external APIs, downstream services, or other network dependencies.

Dependency visibility is part of the baseline observability story for services that make outbound calls.

## Metric Families

| Metric family | What it should measure | Canonical metric name | Key labels | Notes |
|---|---|---|---|---|
| Egress request rate per host | Outbound call volume by dependency | `httpclient_http_requests_count` | `kubernetes_namespace`, `host`, `code`, `method`, `url` | Tracks dependency usage and drop-offs |
| Egress 5xx errors | Downstream server-side failures | `httpclient_http_requests_count` | `kubernetes_namespace`, `host`, `code` | Filter `code=~"5.."` |
| Egress 4xx errors | Downstream client-side failures | `httpclient_http_requests_count` | `kubernetes_namespace`, `host`, `code` | Filter `code=~"4.."` |
| Egress non-2xx errors | Overall non-success outbound responses | `httpclient_http_requests_count` | `kubernetes_namespace`, `host`, `code` | Filter `code!~"2.."` |
| Egress latency | Downstream response time distribution | `httpclient_http_request_duration_ms_hist_bucket` | `kubernetes_namespace`, `host`, `le` | Histogram; derive p90/p95/p99 per host |

## Baseline Expectation

If a service owns outbound dependencies, egress traffic, error, and latency coverage should be treated as baseline, not optional.

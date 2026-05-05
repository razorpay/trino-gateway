# Edge Metrics

Read this file when the service depends on an edge or API gateway layer (Kong) and those signals are part of the observability story.

## Metric Families

| Metric family | Canonical metric name | Key labels | Description |
|---|---|---|---|
| Edge request rate | `kong_request_total` | `service`, `code`, `source` | Request volume through Kong by status code and source |
| Edge request rate by route | `kong_http_requests_total` | `service`, `code`, `route` | Request volume with route-level granularity; used for top-N 4xx/5xx and rate-limiter panels |
| Edge request latency | `kong_request_latency_ms_bucket` | `service` | Total request latency histogram (client perspective) |
| Edge upstream latency | `kong_upstream_latency_ms_bucket` | `service` | Upstream (backend) latency histogram |
| Edge internal latency | `kong_kong_latency_ms_bucket` | `service` | Kong's own processing overhead histogram |

## Derived Views

- Top-N 5xx routes derived from `kong_http_requests_total` filtered by `code=~"5.."`
- Top-N 4xx routes derived from `kong_http_requests_total` filtered by `code=~"4.."`
- p99 latency by route derived from `kong_request_latency_ms_bucket`
- Rate-limiter rejections derived from `kong_http_requests_total` filtered by `code="429"`

## Baseline Expectation

When the service has an edge deployment through Kong, these metrics provide the perimeter-layer visibility. Load this file alongside `references/infra/traefik-metrics.md` if traffic passes through both layers.

# HTTP Metrics

Read this file when the service exposes HTTP or REST endpoints.

These metrics cover the standard HTTP ingress surface. They represent the expected request, latency, and outcome visibility for HTTP services.

## Metric Families

| Metric family | What it should measure | Canonical metric name | Code-owned labels |
|---|---|---|---|
| HTTP request rate | Incoming HTTP request volume | `http_requests_total` | `kubernetes_namespace`, `code` |
| HTTP status distribution | Response volume by status code family or exact status | `http_responses_total` | `kubernetes_namespace`, `k8s_pod`, `code` |
| HTTP latency (service-level) | Response time distribution for all requests | `http_durations_ms_histogram_bucket` | `kubernetes_namespace`, `le` |
| HTTP latency (per-route) | Response time distribution per route; used for liveness and health-check paths | `http_request_duration_ms_bucket` | `route`, `le` |

## Derived Views

- 5xx error rate should be derived from `http_responses_total` filtered by `code=~"5.."`
- 4xx error rate should be derived from `http_responses_total` filtered by `code=~"4.."`
- non-2xx error rate should be derived from `http_responses_total` filtered by `code!~"2.."`
- server-error count for availability calculation should be derived from `http_requests_total` filtered by `code=~"5.."`
- SLA breach count should be derived from `http_durations_ms_histogram_bucket` using bucket math: total rate minus rate at the SLA threshold bucket (`le="$service_sla"`)

## Baseline Expectation

For HTTP services, request volume, status distribution, latency, and outcome/error coverage are baseline metrics and should not be omitted.

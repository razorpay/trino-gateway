# Uptime Metrics

Read this file when uptime, SLA, reachability, or health-probe coverage is relevant to the service.

These metrics describe the standard availability view.

## Metric Families

| Metric family | Metric or form | Description |
|---|---|---|
| Service uptime | `status_cake_test_uptime_percent` | External uptime percentage for the service |
| External response time | External uptime provider response-time metric | Response-time view from the uptime monitor |
| Availability at edge or ingress | Availability metric from the relevant edge or ingress layer | Availability of the user-facing path |
| Health check status | Health check probe metrics | Ping and response status of health surfaces |

## Composite Availability Panels

The standard dashboard includes three derived availability panels that combine uptime percentage with request quality. Each panel produces a composite SLA-aware availability score.

| Panel | Formula | Condition | Metrics used |
|---|---|---|---|
| Availability at Traefik | `uptime% × (non-5xx requests − SLA-breaching requests) / total requests` | `traefikServices` configured | `status_cake_test_uptime_percent`, `traefik_service_requests_total`, `traefik_service_request_duration_seconds_bucket`, `traefik_service_request_duration_seconds_count` |
| Availability at Service | `(uptime/100) × (1 − (5xx + SLA-breach) / total) × 100` | Always present | `status_cake_test_uptime_percent`, `http_requests_total`, `http_durations_ms_histogram_bucket` |
| Availability at Edge | Same formula as Service but using Kong metrics | `edgeDeployment` configured | `status_cake_test_uptime_percent`, `kong_request_total`, `kong_http_requests_total`, `kong_upstream_latency_ms_bucket` |

### SLA Breach Component

The SLA breach count is derived from histogram bucket math. Requests exceeding `$service_sla` threshold are counted as SLA-breaching:
- At Traefik: threshold in seconds (uses `traefik_service_request_duration_seconds_bucket`)
- At Service: threshold in milliseconds (uses `http_durations_ms_histogram_bucket`)
- At Edge: threshold in milliseconds (uses `kong_upstream_latency_ms_bucket`)

## StatusCake Requirement

- when external uptime monitoring is part of the service baseline, the service should have a StatusCake test configured
- the service configuration should carry the StatusCake test identifier needed to bind uptime panels and alerts to the correct monitor
- `status_cake_test_uptime_percent` is the canonical external uptime metric used for that monitor
- if SLA tracking depends on ingress-proxy latency, load `references/infra/traefik-metrics.md` for the Traefik request-latency metrics

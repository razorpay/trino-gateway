# Traefik Metrics

Read this file when the service is exposed through Traefik or when ingress proxy behavior is relevant to the task.

These metrics are infra-owned. They are part of the standard observability picture, but they are not metrics the service code necessarily emits directly.

## Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Request distribution by status | `traefik_service_requests_total` | Request volume seen by Traefik, including HTTP status distribution |
| Request latency | `traefik_service_request_duration_seconds_bucket` | Response time distribution observed at Traefik |

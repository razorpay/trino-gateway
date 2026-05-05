# Baseline Metrics

This file defines the compulsory baseline metric families that every service should cover for the surfaces it owns.

The standard does not require every service to expose every protocol or workload. It does require every service to cover the baseline families that apply to its actual behavior.

## Universal Baseline

These baseline families should exist for every owned service surface, regardless of whether the service is HTTP, gRPC, worker-only, or mixed.

| Metric family | Required when | What it should measure | Notes |
|---|---|---|---|
| Traffic volume | The service accepts requests or processes jobs | Request rate, throughput, or processing rate | Protocol-specific details live in `http-metrics.md`, `grpc-metrics.md`, and `worker-metrics.md` |
| Error and outcome classification | The service returns responses or processing outcomes | Success, failure, and outcome breakdowns | Surface-specific outcome metrics are defined in the linked HTTP, gRPC, and worker references |
| Latency | The service handles synchronous or asynchronous work | Time taken to complete the owned work | Surface-specific latency metrics are defined in the linked HTTP, gRPC, and worker references |
| Health and availability | The service exposes health surfaces or is expected to be reachable | Uptime, health checks, service availability | `references/infra/uptime-metrics.md` contains the broader infra view |
| Runtime health | The service process is owned by the repo | Process and language runtime health | For Go services, read `go-runtime-metrics.md` |

## Surface-Conditional Baseline

If the service has any of the surfaces below, the linked metric families are mandatory for that surface.

| If you have this | Then make sure these metrics exist | Read |
|---|---|---|
| HTTP or REST ingress | HTTP request, status, latency, and error metric families | `http-metrics.md` |
| gRPC ingress | gRPC request, method, status, latency, and error metric families | `grpc-metrics.md` |
| External or downstream calls | Egress traffic, latency, and error metric families | `egress-metrics.md` |
| Background jobs or queue workers | Queue depth, processing rate, failed-job, and duration metric families | `worker-metrics.md` |
| Outbox flow | Outbox store, handler, pending-age, fetched-job, and total-time metric families | `outbox-metrics.md` |

## Routing Rule

Use this file as a routing index:

- every service should satisfy the universal baseline families above
- if the service has one of the listed surfaces, read the linked file for that surface
- if the service does not have that surface, skip that file

## Application Metrics

Application metrics are not prescribed by this file. The agent may define them based on business flows, domain states, or service-specific SLIs.

Application metrics are useful additions, but they do not replace the compulsory baseline families above.

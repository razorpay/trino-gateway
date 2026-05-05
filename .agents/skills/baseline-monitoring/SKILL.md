---
name: baseline-monitoring
description: Index of baseline metric families for Razorpay services. Use it to determine which standard metrics must exist for HTTP, gRPC, workers, egress, outbox, Go runtime, and relevant infra components.
---

# Baseline Observability

Defines the standard metric families agents must consider when working on service observability.

This skill covers metrics only. Logging and tracing standards are separate and should be loaded from their own skills or references when needed.

## Introduction

Observability in this skill is divided into three buckets:

- **Baseline metrics**: compulsory metrics every service surface must emit for the work it owns. These are the standard traffic, latency, error, health, and runtime signals that should exist whether the service is being created from scratch or improved incrementally.
- **Infra metrics**: metrics emitted by shared platform components such as Traefik, Kubernetes, Kafka, ALB, RDS, ElastiCache, SQS, and SNS. These are still part of the standard observability picture, but they should only be loaded when the service actually depends on or is fronted by those infra layers.
- **Application metrics**: service-specific business or domain metrics. The agent is free to define these as needed. They are valuable, but they do not replace the compulsory baseline metrics.

## Usage Rules

- Always read `references/baseline-metrics.md` first.
- Then load only the references relevant to the surfaces involved in the task.
- Use the surface-to-reference map below mechanically: if the service has that surface, the linked metric families must exist.
- If the task only touches gRPC, read `references/grpc-metrics.md` and skip `references/http-metrics.md`.
- If the task only touches service code, do not load `references/infra/` files unless the change explicitly involves those components.
- If the service owns workers, egress, or outbox flows, those metric families are part of the baseline for that workload and should be loaded.
- If the repo is a Go service, read `references/go-runtime-metrics.md`.
- Do not treat this skill as an implementation guide. It is an index of required or relevant metric families and their purposes.

## Surface To Metrics Map

Use this section as the default routing rule.

| If the service has this | Then make sure these metrics exist | Read |
|---|---|---|
| Any owned request or job-handling surface | Traffic, latency, outcome, health, and runtime baseline coverage | `references/baseline-metrics.md` |
| HTTP or REST ingress | HTTP request, status, latency, and error metric families | `references/http-metrics.md` |
| gRPC ingress | gRPC request, method, status, latency, and error metric families | `references/grpc-metrics.md` |
| Outbound or downstream calls | Egress traffic, latency, and error metric families | `references/egress-metrics.md` |
| Background workers or queue processors | Queue depth, processing rate, failure, and duration metric families | `references/worker-metrics.md` |
| Outbox flow | Outbox store, handler, backlog-age, and lifecycle-time metric families | `references/outbox-metrics.md` |
| Go runtime owned by the repo | Go process and runtime metric families | `references/go-runtime-metrics.md` |
| Traefik in front of the service | Traefik request-volume and latency metrics | `references/infra/traefik-metrics.md` |
| Edge or CDN layer relevant to the service | Edge latency, error, cache, geo, bandwidth, and certificate metrics | `references/infra/edge-metrics.md` |
| Uptime or health ownership is relevant | Uptime, health-check, and availability metrics | `references/infra/uptime-metrics.md` |
| Canary rollout analysis exists | Canary versus baseline comparison metrics | `references/infra/canary-metrics.md` |
| Kubernetes runtime is part of the service view | Pod, deployment, node, quota, and cluster-event metrics | `references/infra/kubernetes-metrics.md` |
| Kafka is part of the service topology | Kafka throughput, message-rate, and lag metrics | `references/infra/kafka-metrics.md` |
| ALB fronts the service | ALB request, latency, error, and healthy-host metrics | `references/infra/aws-alb-metrics.md` |
| Auto Scaling Groups are part of the service topology | ASG capacity, instance-health, and scaling activity metrics | `references/infra/aws-asg-metrics.md` |
| RDS is a dependency in the service path | RDS connection, capacity, latency, throughput, and replication metrics | `references/infra/aws-rds-metrics.md` |
| ElastiCache is a dependency in the service path | Cache usage, connection, CPU, eviction, and replication metrics | `references/infra/aws-elasticache-metrics.md` |
| SQS or SNS is part of the service flow | Queue backlog, queue-aging, publish, and delivery metrics | `references/infra/aws-sqs-sns-metrics.md` |

## Reference Index

### Core

- `references/baseline-metrics.md`: compulsory baseline metric families and how to choose the additional files that apply.
- `references/http-metrics.md`: HTTP and REST ingress metrics.
- `references/grpc-metrics.md`: gRPC ingress metrics.
- `references/egress-metrics.md`: outbound dependency and external API metrics.
- `references/worker-metrics.md`: async worker and queue-processing metrics.
- `references/outbox-metrics.md`: outbox lifecycle metrics.
- `references/go-runtime-metrics.md`: Go process and runtime health metrics.

### Infra

- `references/infra/traefik-metrics.md`: Traefik request and latency metrics.
- `references/infra/edge-metrics.md`: edge, CDN, and perimeter metrics.
- `references/infra/uptime-metrics.md`: uptime, SLA, and health-check metrics.
- `references/infra/canary-metrics.md`: canary-versus-baseline comparison metrics.
- `references/infra/kubernetes-metrics.md`: pod, deployment, node, quota, and cluster health metrics.
- `references/infra/kafka-metrics.md`: Kafka throughput and lag metrics.
- `references/infra/aws-alb-metrics.md`: ALB request, latency, error, and healthy-host metrics.
- `references/infra/aws-asg-metrics.md`: ASG capacity, health, and scaling metrics.
- `references/infra/aws-rds-metrics.md`: RDS capacity, performance, and error metrics.
- `references/infra/aws-elasticache-metrics.md`: ElastiCache usage and health metrics.
- `references/infra/aws-sqs-sns-metrics.md`: SNS publish and SQS queue backlog metrics.

## Examples

**New gRPC service:**

- Read `references/baseline-metrics.md`
- Read `references/grpc-metrics.md`
- Read `references/go-runtime-metrics.md`
- Read infra references only if the service setup includes those layers

**Existing HTTP API adding a new downstream call:**

- Read `references/baseline-metrics.md`
- Read `references/http-metrics.md`
- Read `references/egress-metrics.md`
- Read `references/go-runtime-metrics.md` for Go runtime coverage

**Worker-only service using SQS and outbox:**

- Read `references/baseline-metrics.md`
- Read `references/worker-metrics.md`
- Read `references/outbox-metrics.md`
- Read `references/infra/aws-sqs-sns-metrics.md`

## Decision Rule

When working on a repo, the agent should reason in this order:

1. What surfaces does this service have?
2. For each surface, which reference file says what metric families must exist?
3. Which compulsory baseline families are currently missing or incomplete?
4. Which application metrics should be added in addition to the baseline?

Use this skill as a checklist of required metric families.

## PR Requirement

At the end of the implementation, the agent must list every metric it added in the PR description.

For each metric, include:

- metric name
- short description of what it measures
- what are the normal and abnormal values for the metric, and how to interpret them
- potential alerting thresholds if applicable

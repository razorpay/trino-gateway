# Worker Metrics

Read this file when the service owns background workers, consumers, schedulers, or queue processors.

These metrics provide the standard visibility for asynchronous work.

## Metric Families

| Metric family | What it should measure | Canonical metric name | Key labels | Notes |
|---|---|---|---|---|
| Worker queue depth | Number of jobs waiting to be processed | Queue depth metrics | `kubernetes_namespace`, `queue` | Primary backlog signal |
| Job processing rate | Throughput of completed jobs | `job_processed_count` | `kubernetes_namespace`, `queue` | Standard volume signal; non-empty `error` label means failure |
| Failed job rate | Job failures over time | `job_processed_count` | `kubernetes_namespace`, `error`, `exported_job` | Filter `error!=""` for failures; `exported_job` for per-job-type breakdown |
| Processing time | Per-job duration | `job_processing_time_bucket` | `kubernetes_namespace`, `queue`, `le` | Histogram (unit: milliseconds); buckets are powers of 2 from 2 to 2097152 ms |
| Invalid message count | Messages that failed validation | `invalid_message_count` | `kubernetes_namespace`, `queue` | Tracks malformed or unprocessable messages |

## Infra Dependency

- if worker saturation or resource pressure needs to be monitored, load `references/infra/kubernetes-metrics.md` for CPU, memory, restart, and pod-health coverage

## Baseline Expectation

For worker-owned workloads, queue depth, processing rate, failed job rate, and processing time are baseline metrics.

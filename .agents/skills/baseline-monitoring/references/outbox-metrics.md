# Outbox Metrics

Read this file when the service uses an outbox pattern.

The standard treats outbox flows as a distinct workload with its own lifecycle metrics.

The metric names below are the canonical baseline names for outbox coverage.

## Store Operation Metrics

Store operations track the lifecycle of outbox job persistence. Each operation type emits both a counter and a latency histogram. Success vs error is distinguished by the `error` label: `error=~"0"` for success, `error=~"1"` for error.

| Metric family | Canonical metric name | Description |
|---|---|---|
| Create job rate | `outbox_store_create_job_invoked_total` | Number of outbox job creation attempts |
| Create job latency | `outbox_store_create_job_invoked_seconds_bucket` | Duration of job creation operations |
| Update job rate | `outbox_store_update_job_invoked_total` | Number of job update operations; has additional `job_status` label for breakdown by target status |
| Update job latency | `outbox_store_update_job_invoked_seconds_bucket` | Duration of job update operations |
| Delete job rate | `outbox_store_delete_job_invoked_total` | Number of job deletion operations |
| Delete job latency | `outbox_store_delete_job_invoked_seconds_bucket` | Duration of job deletion operations |
| Find pending jobs rate | `outbox_store_find_pending_jobs_invoked_total` | Number of pending-job lookup operations |
| Find pending jobs latency | `outbox_store_find_pending_jobs_invoked_seconds_bucket` | Duration of pending-job lookups |
| Find failed jobs rate | `outbox_store_find_failed_jobs_invoked_total` | Number of failed-job lookup operations |
| Find failed jobs latency | `outbox_store_find_failed_jobs_invoked_seconds_bucket` | Duration of failed-job lookups |

## Job Handler Metrics

Job handler metrics track the processing of outbox jobs after they are fetched. Success vs error is distinguished by the `error` label: `error=~"0"` for success, `error=~"1"` for error. Both metrics carry a `handler_name` label for per-handler breakdown.

| Metric family | Canonical metric name | Key labels | Description |
|---|---|---|---|
| Job handler processed count | `outbox_job_handler_processed_count` | `kubernetes_namespace`, `error`, `handler_name` | Number of jobs processed by the handler |
| Job handler processing time | `outbox_job_handler_processing_time_ms_bucket` | `kubernetes_namespace`, `error`, `handler_name`, `le` | Processing duration histogram (unit: milliseconds) |

## Job Lifecycle Metrics

| Metric family | Canonical metric name | Description |
|---|---|---|
| Oldest pending job age | `outbox_age_of_oldest_pending_job_seconds` | Age of the oldest unprocessed job; primary backlog aging signal |
| Fetched pending jobs | `outbox_store_fetched_pending_jobs_total_bucket` | Distribution of pending jobs fetched per cycle |
| Fetched failed jobs | `outbox_store_fetched_failed_jobs_total_bucket` | Distribution of failed jobs fetched per retry cycle |
| Job total processing time | `outbox_job_process_duration_seconds_bucket` | End-to-end job processing duration histogram; carries `job_name` label for per-job-type breakdown |

## Baseline Expectation

If the service owns an outbox flow, these metrics should be treated as part of the baseline for that workload.

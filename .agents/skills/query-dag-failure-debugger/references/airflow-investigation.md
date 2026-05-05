# Airflow-Orchestration and Upstream-Data Investigation

How to investigate Airflow-first failures before assuming the problem is Spark, Trino, or infrastructure.

---

## Investigation Goal

For Airflow-backed failures, the investigation chain should be:

`DAG file -> task state -> sensor/dependency/runtime config -> upstream run state -> final bucket`

Do not stop at `upstream_failed` or `sensor_timeout`. Those are often propagated states, not the root cause.

---

## Fast Mode vs Deep Mode

- **Fast mode**: Read the DAG, identify the blocking sensor/dependency/timeout setting, and run the single Airflow UI or CLI check that confirms whether this is orchestration or upstream-data.
- **Deep mode**: Trace dependency chains, inspect pool / concurrency pressure, compare run durations, and inspect retry/idempotency behavior when Fast mode is inconclusive.

Use Fast mode first. Switch to Deep mode only when the first confirming check does not settle the bucket.

---

## Step 1 — Read the DAG Definition First

Extract these fields before touching the Airflow UI:

- sensor type (`ExternalTaskSensor`, freshness validator, custom sensor)
- `execution_delta` / `execution_date_fn`
- `external_dag_id` / `external_task_id`
- `execution_timeout`
- `dagrun_timeout`
- `max_active_runs`
- pool name / concurrency settings
- retry count and retry delay

Questions to answer from the DAG file:
1. Is the task itself failing, or is it waiting on something else?
2. Is the sensor targeting the right execution date?
3. Is this DAG likely blocked by pool pressure or concurrency limits?
4. Could retry be failing because a prior partial run already changed state?

---

## Step 2 — Identify the Failure Shape

### A. `sensor_timeout`

Fast check:
- Airflow UI → upstream DAG → run for the exact execution date the sensor was targeting

Interpretation:
- upstream run missing or failed -> `UPSTREAM_DATA`
- upstream run succeeded -> `AIRFLOW_ORCHESTRATION` (sensor config, wrong date mapping, wrong task target)

### B. `upstream_failed`

Fast check:
- Airflow Graph view → trace upstream dependencies to the first task with a real exception or log

Interpretation:
- the `upstream_failed` task is almost never the real root cause
- classify from the first failed task, not the propagated state

### C. `execution_timeout`

Fast check:
- Compare the task's actual run duration with the `execution_timeout` set in code

Interpretation:
- timeout lower than normal runtime growth -> `AIRFLOW_ORCHESTRATION`
- timeout triggered because the task is blocked waiting for missing data -> may still be `UPSTREAM_DATA`

### D. `dagrun_timeout`

Fast check:
- Airflow UI → DAG run history → compare recent total DAG durations

Interpretation:
- growing DAG runtime or queue delays -> `AIRFLOW_ORCHESTRATION`
- one upstream task consistently slipping -> investigate that task as data/runtime owner

---

## Step 3 — Check Airflow Runtime Pressure

Use when the DAG is queued, delayed, or repeatedly timing out.

Check:
- pool usage for the pool assigned to the failing task
- `max_active_runs`
- whether the DAG is paused
- whether another run is still active and blocking the current one

Common signals:
- `max_active_runs=1` and prior run still active -> orchestration bottleneck
- sensor in `mode='poke'` holding slots for hours -> orchestration bottleneck
- pool saturation from unrelated DAGs -> orchestration bottleneck, not upstream-data

---

## Step 4 — Retry / Idempotency Checks

Use when the task fails only on retry or fails after partially succeeding.

Look for:
- resource already exists
- duplicate write / duplicate partition errors
- retry succeeding only after manual cleanup

Interpretation:
- if the fresh run passes but retry fails because state already exists, this is `AIRFLOW_ORCHESTRATION` via retry/idempotency bug, not infra

---

## Step 5 — DBT-in-Airflow Special Case

If the task runs `dbt run`, `dbt test`, or wraps DBT inside a BashOperator/PythonOperator:

- do not stop at the Airflow task failure line
- inspect `dbt.log` or DBT run output if available
- classify from the DBT model/test failure, not from the generic Airflow non-zero exit

Typical split:
- model/test logic error -> downstream SQL/data issue
- task timeout / orchestration around DBT execution -> `AIRFLOW_ORCHESTRATION`

---

## Common False Positives

- `upstream_failed` is a propagated state, not the root cause by itself
- `sensor_timeout` does not automatically mean upstream data is late; it may be targeting the wrong execution date
- multiple queued tasks can look like infrastructure issues when the real problem is pool starvation
- a task timing out after long runtime is not always a compute problem; check `execution_timeout` and `dagrun_timeout` first

---

## Recommended Investigation Order

1. Read the DAG file and extract sensor/dependency/runtime settings
2. Identify whether the task state is primary (`failed`) or propagated (`upstream_failed`, `skipped`)
3. Run the single fastest confirming check:
   - upstream run state for sensor timeout
   - first failed parent task for `upstream_failed`
   - timeout value vs actual runtime for timeout cases
4. Only then go deeper into pool pressure, concurrency, retry/idempotency, or DBT logs

This keeps Airflow investigation short and avoids blaming the wrong system first.

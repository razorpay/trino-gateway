# Triage Examples

Worked examples showing the full triage workflow from input to output. These include clean, ambiguous, and messy inputs.

> Keep examples compact. Use them as behavior references, not exhaustive runbooks.

---

## Example 1 — Spark OOM (SPARK_RESOURCE, High Confidence)

### Input

```
DAG: payments_settlements_spark
Task: run_spark_job
Error: "Container killed by YARN for exceeding memory limits. 12.5 GB of 12 GB physical memory used. Killing container. Dump of the process-tree follows..."
```

### Why it classifies this way
- exact YARN memory-kill phrase
- Spark/EMR system is clear
- no competing bucket signal

### First response shape
- Bucket: `SPARK_RESOURCE`
- Confidence: `High`
- Do this first: inspect DAG `spark_submit_params` and `spark.executor.memoryOverhead`

### Phase 2 extension
If investigation is requested and EMR `stderr.gz` confirms repeated YARN kills, update the evidence section with the exact log line plus the DAG memory settings.

---

## Example 2 — ExternalTaskSensor Timeout (Ambiguous)

### Input

```
DAG: cb_skipthreeds_derived_params
Task: wait_for_table_one
State: sensor_timeout
Error: "ExternalTaskSensor did not find a success state for dag_id=cb_skipthreeds_table_one, task_id=write_table, execution_date=2026-04-05T07:30:00"
```

### Why it is ambiguous
- sensor timeout matches both `AIRFLOW_ORCHESTRATION` and `UPSTREAM_DATA`
- one Airflow UI check resolves it

### First response shape
- Bucket: `AIRFLOW_ORCHESTRATION`
- Confidence: `Medium`
- Do this first: check whether `cb_skipthreeds_table_one` succeeded for that exact execution date

---

## Example 3 — Trino Table Not Found (Needs Decision Tree)

### Input

```
DAG: presto_payments_reconciliation
Error: io.trino.spi.connector.NotFoundException: Table 'hive.derived.payments_v3' does not exist
```

### Why it is not immediately `SQL_FAILURE`
- `TABLE_NOT_FOUND` can still be `PERMISSION_ACCESS` or `UPSTREAM_DATA`
- first confirming check is `SHOW TABLES`

### First response shape
- Bucket: `SQL_FAILURE` (pending table-existence check)
- Confidence: `Medium`
- Do this first: `SHOW TABLES IN hive.derived LIKE 'payments%'`

---

## Example 4 — Trino Access Denied

### Input

```
DAG: merchant_risk_dashboard
Task: build_hourly_metrics
Error: ACCESS_DENIED: Cannot access catalog hive
```

### Why it classifies this way
- exact Trino permission phrase
- catalog named explicitly
- no need to start with query-history analysis

### First response shape
- Bucket: `PERMISSION_ACCESS`
- Confidence: `High`
- Do this first: check the DAG `user` / connection argument

---

## Example 5 — Trino Internal Failure That Passes on Retry

### Input

```
DAG: daily_settlement_recon
Task: run_reconciliation_query
Error: db_error: trino: query failed (200 OK)
Retry: second attempt succeeded with no DAG changes
```

### Why it classifies this way
- retry success without code change is the strongest transient signal

### First response shape
- Bucket: `QUERY_ENGINE_TRANSIENT`
- Confidence: `High`
- Do this first: verify retry used the same SQL

---

## Example 6 — Messy Input With Too Little Signal

### Input

```
Slack paste:
- "daily dag failed again"
- "saw java stuff in logs"
- task state looked red in Airflow
```

### First response shape
- Bucket: `UNKNOWN`
- Confidence: `Low`
- Do this first: ask for the exact Airflow task log or the backend (Spark / Trino / Airflow-only)

Why: there is no exact error text, no confirmed system, and no direct signal that reaches any bucket confidently.

---

## Example 7 — Propagated `upstream_failed`

### Input

```
Task state: upstream_failed
No exception shown for this task
```

### First response shape
- Bucket: `UNKNOWN` until the first failed parent task is identified
- Confidence: `Low`
- Do this first: trace dependency chain to the first task with a real exception/log

Why: `upstream_failed` is a propagated state, not the root cause by itself.

---

## Example 8 — Livy Session Dead At Startup

### Input

```
Session 14 unexpectedly reached final status 'dead'
stderr shows unresolved dependency org.foo:bar
```

### First response shape
- Bucket: `SPARK_RUNTIME`
- Confidence: `High`
- Do this first: inspect dependency setup / `python_packages`, not executor memory

Why: session dies at startup and dependency resolution fails before tasks begin.

---

## Example 9 — Multi-DAG Same-Window Failure

### Input

```
Three unrelated DAGs failed in the same 20-minute window on shared EMR clusters
```

### First response shape
- Bucket: `INFRASTRUCTURE`
- Confidence: `Medium` before shared evidence, `High` after cluster/worker evidence
- Do this first: gather shared cluster termination / worker evidence before per-DAG debugging

Why: the infra pre-check fires before individual bucket classification.

# Compact Eval Cases

Regression-style cases for validating bucket choice, confidence, and first follow-up/action.

---

## Case 1 — Minimal Input

**Input**
```
DAG failed.
```

**Expected**
- Bucket: `UNKNOWN`
- Confidence: `Low`
- First follow-up: ask for the exact Airflow task log or the backend (Spark / Trino / Airflow-only)

---

## Case 2 — Spark OOM

**Input**
```
Container killed by YARN for exceeding memory limits.
```

**Expected**
- Bucket: `SPARK_RESOURCE`
- Confidence: `High`
- First follow-up: check DAG `spark_submit_params` or EMR `stderr.gz`

---

## Case 3 — Trino Table Exists But User Cannot See It

**Input**
```
io.trino.spi.connector.NotFoundException: Table 'hive.derived.payments_v3' does not exist
SHOW TABLES says payments_v3 exists
```

**Expected**
- Bucket: `PERMISSION_ACCESS`
- Confidence: `High`
- First follow-up: check DAG `user` and confirm catalog visibility under that user

---

## Case 4 — Sensor Timeout With Successful Upstream Run

**Input**
```
ExternalTaskSensor timed out waiting for dag_id=foo_upstream, task_id=write_table
Upstream DAG run for the same execution_date succeeded
```

**Expected**
- Bucket: `AIRFLOW_ORCHESTRATION`
- Confidence: `High`
- First follow-up: inspect `execution_delta` / `execution_date_fn` in DAG code

---

## Case 5 — Retry Success With No SQL Change

**Input**
```
db_error: trino: query failed (200 OK)
Second retry succeeded with same SQL
```

**Expected**
- Bucket: `QUERY_ENGINE_TRANSIENT`
- Confidence: `High`
- First follow-up: verify retry history and only then consider platform escalation if repeated

---

## Case 6 — Propagated `upstream_failed`

**Input**
```
Task state: upstream_failed
No exception shown for this task
```

**Expected**
- Bucket: `UNKNOWN` until the first failed parent task is identified
- Confidence: `Low`
- First follow-up: trace dependency chain to the first task with a real exception/log

---

## Case 7 — Livy Session Dead At Startup

**Input**
```
Session 14 unexpectedly reached final status 'dead'
stderr shows unresolved dependency org.foo:bar
```

**Expected**
- Bucket: `SPARK_RUNTIME`
- Confidence: `High`
- First follow-up: inspect `python_packages` / dependency setup, not executor memory

---

## Case 8 — Multi-DAG Same-Window Failure

**Input**
```
Three unrelated DAGs failed in the same 20-minute window on shared EMR clusters
```

**Expected**
- Bucket: `INFRASTRUCTURE`
- Confidence: `Medium` before shared evidence, `High` after cluster termination/worker evidence
- First follow-up: gather shared cluster/worker evidence before per-DAG debugging

# Failure Bucket Definitions

Detailed reference for the 9 failure buckets. Each entry includes signals, common root causes, ownership, false positives, and reclassification rules.

---

## Confidence and Evidence Rules

Use these rules before assigning confidence:

- **High confidence**: 2+ strong signals from one bucket, with at least one exact exception class or exact log phrase from the user's input
- **Medium confidence**: 1 strong signal or 2 indirect signals, but one confirming check is still required
- **Low confidence**: sparse input, conflicting signals, or multiple buckets remain plausible

Do not force a non-`UNKNOWN` bucket when confidence is Low and no remaining clarifying question can settle it.
Low-confidence outputs must always ask for a named artifact such as the Airflow task log, EMR `stderr.gz`, Trino `SHOW TABLES` result, or upstream DAG run state.

---

## Trino Error Analysis Guide

Before classifying a Trino failure, always cross-check three things — the error code alone is not enough:

| Check | What to look for |
|-------|-----------------|
| **Error code** | `SYNTAX_ERROR` / `TABLE_NOT_FOUND` / `ACCESS_DENIED` / `QUERY_EXPIRED` etc. |
| **Query structure** | Does the query reference the right catalog/schema? Are table names correct? Is there a JOIN with no filter? |
| **User context** | What `user` was passed to `execute_presto_query_cli`? Different users have different catalog visibility. Check the DAG source file. |
| **Retry history** | Did the same query succeed on a prior run? If yes, lean `QUERY_ENGINE_TRANSIENT`. If it's a new query or recent code change, lean `SQL_FAILURE` or `PERMISSION_ACCESS`. |

Common false positives to keep in mind before classifying:
- `TABLE_NOT_FOUND` can be `PERMISSION_ACCESS` or `UPSTREAM_DATA`, not just `SQL_FAILURE`
- `upstream_failed` is often propagated from an earlier task and is not the root cause by itself
- `ExecutorLostFailure` without OOM text can be transient shuffle/network instability, not necessarily memory pressure
- `QUERY_EXPIRED` that passes on retry is usually transient; one that fails every day the same way may be a deterministic timeout pattern
- multiple DAGs failing in the same window can still be pool pressure or shared upstream-data breakage, not always infra first

**Trino `TABLE_NOT_FOUND` decision tree**:
1. Does the table exist? Run `SHOW TABLES IN <schema> LIKE '<table>%'`
2. If table exists → user can't see it → `PERMISSION_ACCESS`
3. If table doesn't exist → is it created by an upstream DAG? → `UPSTREAM_DATA`
4. If table doesn't exist → hardcoded wrong name in query → `SQL_FAILURE`

**Trino `ACCESS_DENIED` sub-types**:
- Catalog-level access (`user X cannot access catalog Y`) → data team owns; check `user` param in DAG
- Row/column-level security violation → data team owns; check if the user has been granted the right role
- S3 read permission denied during Trino scan → platform/infra team owns; IAM issue on the Trino worker's instance profile

---

## 1. AIRFLOW_ORCHESTRATION

**Description**: DAG/task config, scheduling, pool, sensor, retry, or dependency issue.

**Key Signals**:
- `upstream_failed`, `skipped`, `sensor_timeout`
- `AirflowSkipException`
- `max_active_runs` limit hit
- pool exhaustion
- `execution_date` mismatch
- `ExternalTaskSensor` never satisfied
- `DagRunAlreadyExists`
- task `execution_timeout`
- `dagrun_timeout`

**Common Root Causes**:
- `max_active_runs=1` with a previous run still active
- sensor uses `mode='poke'` and holds worker slots for hours
- wrong `execution_delta` / `execution_date_fn`
- DAG paused in Airflow UI
- pool slots exhausted because many DAGs share the same pool
- timeout set too low for current runtime
- retry logic is not idempotent and fails on partial state
- DBT task failure hidden behind generic Airflow non-zero exit

**Typical Owner**: DAG author. If owner is unclear, start with the DAG file path and recent git history, then loop in the team owning the upstream dependency or shared pool.

**Common False Positives**:
- `upstream_failed` is usually propagated from an earlier task
- sensor timeout does not automatically mean missing upstream data
- multiple queued tasks can still be pool starvation or `max_active_runs`, not infra

**Stop / Reclassify When**:
- upstream run for the targeted execution date is missing or failed → `UPSTREAM_DATA`
- the actual first failed parent task shows Spark/Trino/runtime evidence → classify from that task instead
- the task never started because of pool or concurrency pressure → keep `AIRFLOW_ORCHESTRATION`

**Repo-Specific Patterns**:
- Datasync DAGs often fail due to hardcoded `execution_timeout` values as runtimes grow
- DBT-in-Airflow failures often need `dbt.log`; the Airflow task log alone is too generic
- `mode='poke'` sensors can exhaust worker slots and look like broader scheduler instability

See [airflow-investigation.md](airflow-investigation.md) for the stepwise investigation flow.

---

## 2. UPSTREAM_DATA

**Description**: Required input data missing, late, or corrupt.

**Key Signals**:
- freshness validator failure
- source table empty or partition missing
- `ExternalTaskSensor` timed out and upstream DAG did not run
- `hive.NoSuchObjectException`
- S3 path not found / `NoSuchKey`
- data quality threshold breach
- TiDB replication lag / empty table
- `FileNotFoundException` on expected input path

**Common Root Causes**:
- upstream pipeline delayed or failed
- schema change upstream dropped/renamed a partition column
- source system outage
- TiDB replication lag causing stale reads
- S3 lifecycle deleted a needed partition
- upstream DAG succeeded but wrote zero rows

**Typical Owner**: Upstream DAG owner. Escalate to source-system owner if the data originates outside this repo.

**Common False Positives**:
- `TABLE_NOT_FOUND` in Trino can still be a permission issue if the table exists
- S3 read failures may be permission problems, not missing data

**Stop / Reclassify When**:
- upstream DAG run exists and succeeded → reconsider `AIRFLOW_ORCHESTRATION`
- expected table exists but only some users cannot see it → move to `PERMISSION_ACCESS`
- query/object name is plainly wrong in DAG SQL → move to `SQL_FAILURE`

**Repo-Specific Patterns**:
- TiDB lag is a recurring cause of stale or empty reads
- upstream-written Hive/derived tables often fail consumers via missing partitions before full table absence is obvious

---

## 3. SPARK_RESOURCE

**Description**: EMR Spark job fails due to memory, shuffle, skew, executor, or cluster resource exhaustion.

**Key Signals**:
- `OutOfMemoryError`
- `Container killed by YARN for exceeding memory limits`
- `ExecutorLostFailure` with YARN/kill context
- `GC overhead limit exceeded`
- long-running stage with no progress
- data skew symptoms
- `BOOTSTRAP_FAILURE`
- EMR cluster terminated during/around step execution

**Common Root Causes**:
- `spark.executor.memory` too low for current data volume
- partition count too low causing oversized tasks
- instance type too small for job size
- data skew on join key
- `spark.executor.memoryOverhead` not set high enough
- unfiltered large-table join causing massive shuffle
- shared cluster config drift affecting multiple DAGs

**Typical Owner**: DAG author. Escalate to data-eng on-call for repeated failures or shared-cluster impact. If multiple unrelated DAGs share the same failure window, loop in platform/infra sooner.

**Common False Positives**:
- `ExecutorLostFailure` without explicit YARN/OOM text can be shuffle/network instability
- `TERMINATED` cluster status without step-level evidence may actually be `INFRASTRUCTURE`
- `BOOTSTRAP_FAILURE` may be dependency/bootstrap script breakage, not executor sizing

**Stop / Reclassify When**:
- cluster is `TERMINATED` with no step-level error and multiple DAGs are affected → `INFRASTRUCTURE`
- Livy/session startup dies before any tasks run → `SPARK_RUNTIME`
- `stderr.gz` lacks OOM/YARN signals and instead shows code/dependency exception → `SPARK_RUNTIME`

**Repo-Specific Patterns**:
- unfiltered joins against large tables like `payments` / `payments_analytics` can trigger huge shuffles and executor loss
- `spark.executor.memoryOverhead` is often the missing knob in PySpark-heavy jobs
- shared `datasink_common` style cluster configs can create repeated failures across DAGs

See [emr-log-access.md](emr-log-access.md) for the fast/deep investigation order.

---

## 4. SPARK_RUNTIME

**Description**: Spark application logic error, dependency failure, or bad input handling.

**Key Signals**:
- `NullPointerException`
- `AnalysisException`
- `ClassNotFoundException`
- `Task not serializable`
- `ClassCastException`
- assertion failure in job code
- `STEP_FAILURE` with non-OOM Java exception
- `ModuleNotFoundError` / `ImportError`
- `Session N unexpectedly reached final status 'dead'`

**Common Root Causes**:
- new/changed source schema not handled in code
- null values in columns assumed to be non-null
- wrong schema hardcoded in read path
- missing Python dependency in `python_packages`
- non-serializable object used in UDF / closure
- Livy session death caused by dependency resolution or classpath/proxy-user config

**Typical Owner**: DAG/script author.

**Common False Positives**:
- Livy session death is often mistaken for memory pressure when it actually fails before tasks start
- schema/data problems may be blamed on upstream-data too early when the code simply does not handle expected input safely

**Stop / Reclassify When**:
- exact YARN/OOM signals appear → `SPARK_RESOURCE`
- source data is clearly missing/late rather than malformed for code handling → `UPSTREAM_DATA`

**Repo-Specific Patterns**:
- dependency setup in Spark DAGs often fails via package/bootstrap mismatch before the job starts
- evolved schemas hitting hardcoded transforms are a common cause in stable DAGs after upstream changes

---

## 5. SQL_FAILURE

**Description**: Deterministic Trino/Presto query syntax, semantic, or data-type error.

**Key Signals**:
- `SYNTAX_ERROR`
- `TYPE_MISMATCH`
- `COLUMN_NOT_FOUND`
- `FUNCTION_NOT_FOUND`
- `INVALID_CAST_ARGUMENT`
- wrong catalog/schema reference
- ambiguous column name

**Common Root Causes**:
- table renamed or dropped in metastore
- wrong env/catalog/schema in SQL
- schema evolution broke a column reference
- query written for different engine syntax
- wrong `user` chosen in DAG makes a valid table appear missing

**Typical Owner**: DAG/query author.

**Common False Positives**:
- `TABLE_NOT_FOUND` may be `UPSTREAM_DATA` or `PERMISSION_ACCESS`
- repeated timeout-shaped failures can still be transient engine issues, not bad SQL

**Stop / Reclassify When**:
- same query passes unchanged on retry → `QUERY_ENGINE_TRANSIENT`
- table exists but DAG user cannot access it → `PERMISSION_ACCESS`
- object should have been created upstream but does not exist → `UPSTREAM_DATA`

**Repo-Specific Patterns**:
- stale `hive.derived` references and wrong environment prefixes are frequent causes
- service-user differences can make SQL look wrong when the real issue is visibility

---

## 6. QUERY_ENGINE_TRANSIENT

**Description**: Trino/Presto coordinator or worker instability — not a code/data bug.

**Key Signals**:
- `QUERY_EXPIRED`
- `NO_NODES_AVAILABLE`
- `GENERIC_INTERNAL_ERROR`
- `Connection reset by peer`
- `context deadline exceeded`
- `db_error: trino: query failed (200 OK)`
- query succeeds on retry with no code change

**Common Root Causes**:
- cluster under load during peak hours
- worker recycle / spot reclaim
- coordinator GC pause
- transient S3 throttling
- large result set / remote page issues
- external clients with shorter timeouts than Airflow retries
- Querybook app error rather than core Trino failure

**Typical Owner**: Platform/infra team if repeated across queries; DAG author adds retry if isolated.

**Common False Positives**:
- a failure that happens every day the same way is probably not transient
- Querybook/app-layer errors can look like core Trino issues

**Stop / Reclassify When**:
- same SQL fails consistently without any retry success → reconsider `SQL_FAILURE`, `UPSTREAM_DATA`, or `PERMISSION_ACCESS`
- exact permission/object-missing error appears underneath the transient wrapper → classify by that deeper signal

**Repo-Specific Patterns**:
- `db_error: trino: query failed (200 OK)` is usually transient internal execution failure
- n8n / reporting-client timeout errors often originate client-side, not in Trino itself

---

## 7. PERMISSION_ACCESS

**Description**: IAM, S3, network, credential, or Trino catalog permission failure.

**Sub-type A — Infrastructure permissions (IAM / S3 / network)**

**Key Signals**:
- `AccessDeniedException` on AWS/S3
- `403 Forbidden`
- `InvalidClientTokenId`
- connection refused to Livy/Hive
- missing IAM role / blocked network path

**Common Root Causes**:
- IAM role missing from EMR cluster config
- cross-account bucket policy not updated
- expired / rotated credentials in Airflow connection
- network ACL/security group drift

**Typical Owner**: Platform/infra team.

**Sub-type B — Trino catalog / data permissions**

**Key Signals**:
- `ACCESS_DENIED` from Trino
- `User X cannot access catalog Y`
- table exists but DAG user cannot see it
- query works under one user but fails under another

**Common Root Causes**:
- wrong DAG `user` argument
- missing grant on catalog/schema/table
- row/column security blocking access
- restrictive default permissions on newly created table

**Typical Owner**: Data team managing Trino roles/grants, unless the wrong `user` in DAG config is the cause.

**Common False Positives**:
- `TABLE_NOT_FOUND` may actually be permission visibility
- `NoSuchKey` on S3 may actually be missing upstream data rather than permission

**Stop / Reclassify When**:
- object/path truly does not exist → `UPSTREAM_DATA` or `SQL_FAILURE`
- wrong object name/schema is the real issue → `SQL_FAILURE`

---

## 8. INFRASTRUCTURE

**Description**: Platform-level environment failure outside the DAG's code or data.

**Key Signals**:
- EMR cluster `TERMINATED` / `TERMINATED_WITH_ERRORS` without a step-level application error
- Airflow worker killed without Python exception
- multiple unrelated DAGs failing in the same time window
- spot interruption / auto-termination evidence
- shared platform alert at same time as DAG failures

**Common Root Causes**:
- EMR auto-termination too aggressive
- spot reclaim without on-demand fallback
- Airflow worker recycle / OOM
- cross-cutting infra change (VPC, SG, IAM)
- shared EMR cluster terminated by automation or another team

**Typical Owner**: Platform/infra team. DAG author may need retry or cluster recreation logic afterward.

**Common False Positives**:
- multiple failures can still be shared pool pressure or shared upstream-data breakage
- one terminated cluster with a clear step-level OOM is usually `SPARK_RESOURCE`, not infra-first

**Stop / Reclassify When**:
- only one DAG is affected and step logs show a clear application error → reclassify to Spark/SQL/runtime bucket
- Airflow scheduling/pool pressure fully explains the delay/failure → `AIRFLOW_ORCHESTRATION`

---

## 9. UNKNOWN

**Description**: Insufficient context to classify with confidence.

**Use When**:
- no error text is provided
- log snippet is too short to identify the system/exception
- signals match 2+ buckets equally and disambiguation is exhausted
- failure happened in a system outside v1 scope
- error is generic (`Task failed`, `Non-zero exit code`) with no stack trace

**Action**: Always list 2–3 specific pieces of context that would resolve the ambiguity, for example:
- the full Airflow task log or first 30–50 relevant lines
- the backend used by the DAG (Spark / Trino / Airflow-only)
- whether the task started and then failed, or never scheduled
- EMR `stderr.gz`, Trino `SHOW TABLES`, or upstream DAG run state — whichever is the smallest decisive artifact

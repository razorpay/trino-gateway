# Trino and Presto Query Investigation

How to investigate a DAG failure that comes from a Trino or Presto query, with the same depth used for Spark/EMR DAG debugging.

---

## Investigation Goal

For Trino/Presto-backed DAGs, the investigation chain should be:

`DAG file -> exact SQL + execution user -> query history / task log -> metadata and permission checks -> failure bucket`

Do not classify from the error code alone when deeper evidence is available.

---

## Fast Mode vs Deep Mode

- **Fast mode**: read the DAG, extract SQL + `user`, inspect the task log, and run the single smallest metadata check that resolves the ambiguity.
- **Deep mode**: inspect query history, worker/coordinator failure details, repeated retry behavior, and cross-query platform instability.

Use Fast mode first. Switch to Deep mode only when the bucket remains ambiguous after the DAG read plus one confirming check.

Repo-specific patterns to keep in mind:
- `execute_presto_query_cli` and similar helpers are the first place to confirm the effective `user`
- `hive.derived` and environment-specific catalog/schema references are common sources of wrong-env / wrong-schema errors
- `TABLE_NOT_FOUND` is often a visibility or upstream-data problem before it is a pure SQL typo

---

## Step 1 — Gather Minimum Context

Collect the smallest set of identifiers that unlock a useful investigation:

- DAG name / ID
- task name
- execution date
- exact error text from the Airflow task log
- query ID, if present
- whether the task retried and whether retry succeeded
- the `user` or service account used to run the query

If query ID is missing, continue with DAG code and Airflow task-log evidence.

If the user already shared the exact error text or retry result, do not ask for it again.

---

## Step 2 — Read the DAG Definition First

Before reasoning about the error, inspect the DAG code and extract:

- the helper used to run the query, such as `execute_presto_query_cli`
- the exact SQL, or the function/template that renders it
- catalog and schema references
- table names and whether they are hardcoded or templated
- execution-time parameters such as `{{ ds }}`, partitions, date ranges, or environment-specific prefixes
- the `user`, connection, or service account passed to the query runner

Questions to answer from the DAG file:

1. Is this query static or generated dynamically?
2. Does it depend on upstream-created tables or partitions?
3. Could the query point to the wrong environment, catalog, or schema?
4. Is the query being run as the expected service user?
5. Is this likely one of the repo's common cases: wrong service user, wrong `hive.derived` object path, or stale environment prefix?

---

## Step 3 — Pull Evidence from the Airflow Task Log

From the task log, extract:

- exact error code and error message
- query ID if logged
- rendered SQL or table references if present
- whether the failure happened immediately, during planning, or after running for a while
- whether this was the first attempt or a retry

These timing signals matter:

- **Immediate failure** often means `SQL_FAILURE` or `PERMISSION_ACCESS`
- **Fails after long execution** may indicate `QUERY_ENGINE_TRANSIENT`, result-size issues, or backend instability
- **Passes on retry** strongly suggests `QUERY_ENGINE_TRANSIENT`

---

## Step 4 — Use Query History When Query ID Is Available

If you have a Trino query ID, inspect query history / UI and capture:

- final state
- failure reason and error code
- queued time
- execution time
- bytes scanned / rows scanned if available
- peak memory
- worker/coordinator error text

Use that data to separate:

- **Planning-time deterministic errors**: syntax, table missing, column missing, invalid cast
- **Execution-time deterministic errors**: division by zero, bad cast on data, semantic error that only appears after scan starts
- **Engine/transient failures**: worker crash, coordinator instability, query expired, remote page errors, internal server failures

If the same query later succeeds without SQL changes, classify as `QUERY_ENGINE_TRANSIENT`.

---

## Step 5 — Run Metadata Checks Before Classifying

These are the fastest high-signal checks for query failures.

### Check A: Does the table exist?

Use for `TABLE_NOT_FOUND` or any missing-object error.

```sql
SHOW TABLES IN <catalog>.<schema> LIKE '<table_prefix>%';
```

Interpretation:

- Table does not exist:
  - if an upstream DAG was supposed to create it -> `UPSTREAM_DATA`
  - if the query references the wrong name -> `SQL_FAILURE`
- Table exists:
  - if the DAG user still cannot access it -> `PERMISSION_ACCESS`

### Check B: Is the schema what the query expects?

Use for `COLUMN_NOT_FOUND`, `TYPE_MISMATCH`, `INVALID_CAST_ARGUMENT`.

```sql
DESCRIBE <catalog>.<schema>.<table>;
SHOW CREATE TABLE <catalog>.<schema>.<table>;
```

Interpretation:

- column absent or renamed -> `SQL_FAILURE` or `UPSTREAM_DATA` depending on whether the upstream schema changed unexpectedly
- type changed upstream and query was not updated -> usually `SQL_FAILURE`

### Check C: Does the query work under the same user?

Use for `ACCESS_DENIED`, ambiguous `TABLE_NOT_FOUND`, or cases where one person can run the query but the DAG cannot.

Interpretation:

- fails only for the DAG user -> `PERMISSION_ACCESS`
- fails for everyone with same SQL -> not a user-specific permission issue

### Check D: Did retry change the result?

Use for `QUERY_EXPIRED`, `GENERIC_INTERNAL_ERROR`, `NO_NODES_AVAILABLE`, `db_error: trino: query failed (200 OK)`.

Interpretation:

- succeeds on retry with no SQL change -> `QUERY_ENGINE_TRANSIENT`
- fails consistently -> `SQL_FAILURE`, `UPSTREAM_DATA`, or `PERMISSION_ACCESS`

---

## Decision Trees

## `TABLE_NOT_FOUND`

1. Confirm whether the table exists
2. If the table exists -> `PERMISSION_ACCESS`
3. If the table does not exist and an upstream DAG should have created it -> `UPSTREAM_DATA`
4. If the table does not exist and the query name/schema is wrong -> `SQL_FAILURE`

## `ACCESS_DENIED`

1. Does the error mention catalog/schema/table permissions? -> `PERMISSION_ACCESS`
2. Does the error point to S3/IAM or backend storage access? -> `PERMISSION_ACCESS` owned by platform/infra
3. Does the same query work under a different user? -> confirms Trino permission split

## `COLUMN_NOT_FOUND` / `TYPE_MISMATCH`

1. Compare query columns and casts with current schema
2. If query is stale or wrong -> `SQL_FAILURE`
3. If upstream schema changed unexpectedly and broke a stable DAG -> often `UPSTREAM_DATA`, but default to `SQL_FAILURE` unless upstream ownership is clear

## `QUERY_EXPIRED` / `NO_NODES_AVAILABLE` / `GENERIC_INTERNAL_ERROR`

1. Check retry history
2. Check query history for worker/coordinator instability
3. If retry succeeds without code change -> `QUERY_ENGINE_TRANSIENT`
4. If it never succeeds and there is a deterministic query error underneath -> classify by that deeper error instead

---

## Fast Query Checks

Use small, low-risk checks rather than rerunning the full expensive query immediately.

```sql
SHOW TABLES IN <catalog>.<schema> LIKE '<table_prefix>%';
DESCRIBE <catalog>.<schema>.<table>;
SHOW CREATE TABLE <catalog>.<schema>.<table>;
SELECT 1 FROM <catalog>.<schema>.<table> LIMIT 1;
```

These checks answer most of the first-order questions:

- object exists or not
- schema is what the DAG expects or not
- user can read the object or not

---

## Common Trino / Presto Failure Patterns

| Signal | Likely bucket | Fastest confirming check |
|---|---|---|
| `SYNTAX_ERROR` | `SQL_FAILURE` | Read exact SQL from DAG / rendered log |
| `TABLE_NOT_FOUND` | `SQL_FAILURE` / `UPSTREAM_DATA` / `PERMISSION_ACCESS` | `SHOW TABLES ...` + check DAG `user` |
| `COLUMN_NOT_FOUND` | `SQL_FAILURE` | `DESCRIBE table` |
| `TYPE_MISMATCH` / `INVALID_CAST_ARGUMENT` | `SQL_FAILURE` | Compare query casts to current schema |
| `ACCESS_DENIED` | `PERMISSION_ACCESS` | Retry under same user / inspect exact permission text |
| `QUERY_EXPIRED` | `QUERY_ENGINE_TRANSIENT` | Check retry history |
| `NO_NODES_AVAILABLE` | `QUERY_ENGINE_TRANSIENT` | Check cluster/query history |
| `db_error: trino: query failed (200 OK)` | `QUERY_ENGINE_TRANSIENT` | Query accepted but failed internally |

---

## Investigation Output Expectations

A good investigation answer should include:

- the exact DAG/task being investigated
- what SQL or object reference the DAG actually used
- what user/catalog/schema it ran under
- whether the table/object exists
- whether the issue is deterministic, permission-related, upstream-data-related, or transient
- the final failure bucket and confidence
- the next exact check or owner

---

## Ownership Guide

- `SQL_FAILURE` -> DAG/query author
- `UPSTREAM_DATA` -> upstream DAG owner
- `PERMISSION_ACCESS` caused by wrong DAG `user` / connection -> DAG author first
- `PERMISSION_ACCESS` for Trino grants -> data team managing Trino roles/grants
- `PERMISSION_ACCESS` for S3/IAM backend access -> platform/infra team
- `QUERY_ENGINE_TRANSIENT` -> platform team if repeated across queries; DAG author adds retry if isolated

If ownership is unclear after reading the DAG and running the smallest metadata check, default to the DAG author as first contact and include the exact SQL/user evidence in the escalation.

---

## Recommended Investigation Order

1. Read the DAG file and extract SQL + `user`
2. Pull exact error text and query ID from Airflow task log
3. Check retry history
4. Run the single smallest metadata check that resolves the ambiguity
5. If still unresolved and query ID exists, inspect query history
6. Only then assign the final bucket

This keeps the investigation fast and avoids classifying too early from ambiguous Trino errors.

- the `user`, connection, or service account passed to the query runner

Questions to answer from the DAG file:

1. Is this query static or generated dynamically?
2. Does it depend on upstream-created tables or partitions?
3. Could the query point to the wrong environment, catalog, or schema?
4. Is the query being run as the expected service user?

---

## Step 3 — Pull Evidence from the Airflow Task Log

From the task log, extract:

- exact error code and error message
- query ID if logged
- rendered SQL or table references if present
- whether the failure happened immediately, during planning, or after running for a while
- whether this was the first attempt or a retry

These timing signals matter:

- **Immediate failure** often means `SQL_FAILURE` or `PERMISSION_ACCESS`
- **Fails after long execution** may indicate `QUERY_ENGINE_TRANSIENT`, result-size issues, or backend instability
- **Passes on retry** strongly suggests `QUERY_ENGINE_TRANSIENT`

---

## Step 4 — Use Query History When Query ID Is Available

If you have a Trino query ID, inspect query history / UI and capture:

- final state
- failure reason and error code
- queued time
- execution time
- bytes scanned / rows scanned if available
- peak memory
- worker/coordinator error text

Use that data to separate:

- **Planning-time deterministic errors**: syntax, table missing, column missing, invalid cast
- **Execution-time deterministic errors**: division by zero, bad cast on data, semantic error that only appears after scan starts
- **Engine/transient failures**: worker crash, coordinator instability, query expired, remote page errors, internal server failures

If the same query later succeeds without SQL changes, classify as `QUERY_ENGINE_TRANSIENT`.

---

## Step 5 — Run Metadata Checks Before Classifying

These are the fastest high-signal checks for query failures.

### Check A: Does the table exist?

Use for `TABLE_NOT_FOUND` or any missing-object error.

```sql
SHOW TABLES IN <catalog>.<schema> LIKE '<table_prefix>%';
```

Interpretation:

- Table does not exist:
  - if an upstream DAG was supposed to create it -> `UPSTREAM_DATA`
  - if the query references the wrong name -> `SQL_FAILURE`
- Table exists:
  - if the DAG user still cannot access it -> `PERMISSION_ACCESS`

### Check B: Is the schema what the query expects?

Use for `COLUMN_NOT_FOUND`, `TYPE_MISMATCH`, `INVALID_CAST_ARGUMENT`.

```sql
DESCRIBE <catalog>.<schema>.<table>;
SHOW CREATE TABLE <catalog>.<schema>.<table>;
```

Interpretation:

- column absent or renamed -> `SQL_FAILURE` or `UPSTREAM_DATA` depending on whether the upstream schema changed unexpectedly
- type changed upstream and query was not updated -> usually `SQL_FAILURE`

### Check C: Does the query work under the same user?

Use for `ACCESS_DENIED`, ambiguous `TABLE_NOT_FOUND`, or cases where one person can run the query but the DAG cannot.

Interpretation:

- fails only for the DAG user -> `PERMISSION_ACCESS`
- fails for everyone with same SQL -> not a user-specific permission issue

### Check D: Did retry change the result?

Use for `QUERY_EXPIRED`, `GENERIC_INTERNAL_ERROR`, `NO_NODES_AVAILABLE`, `db_error: trino: query failed (200 OK)`.

Interpretation:

- succeeds on retry with no SQL change -> `QUERY_ENGINE_TRANSIENT`
- fails consistently -> `SQL_FAILURE`, `UPSTREAM_DATA`, or `PERMISSION_ACCESS`

---

## Decision Trees

## `TABLE_NOT_FOUND`

1. Confirm whether the table exists
2. If the table exists -> `PERMISSION_ACCESS`
3. If the table does not exist and an upstream DAG should have created it -> `UPSTREAM_DATA`
4. If the table does not exist and the query name/schema is wrong -> `SQL_FAILURE`

## `ACCESS_DENIED`

1. Does the error mention catalog/schema/table permissions? -> `PERMISSION_ACCESS`
2. Does the error point to S3/IAM or backend storage access? -> `PERMISSION_ACCESS` owned by platform/infra
3. Does the same query work under a different user? -> confirms Trino permission split

## `COLUMN_NOT_FOUND` / `TYPE_MISMATCH`

1. Compare query columns and casts with current schema
2. If query is stale or wrong -> `SQL_FAILURE`
3. If upstream schema changed unexpectedly and broke a stable DAG -> often `UPSTREAM_DATA`, but default to `SQL_FAILURE` unless upstream ownership is clear

## `QUERY_EXPIRED` / `NO_NODES_AVAILABLE` / `GENERIC_INTERNAL_ERROR`

1. Check retry history
2. Check query history for worker/coordinator instability
3. If retry succeeds without code change -> `QUERY_ENGINE_TRANSIENT`
4. If it never succeeds and there is a deterministic query error underneath -> classify by that deeper error instead

---

## Fast Query Checks

Use small, low-risk checks rather than rerunning the full expensive query immediately.

```sql
SHOW TABLES IN <catalog>.<schema> LIKE '<table_prefix>%';
DESCRIBE <catalog>.<schema>.<table>;
SHOW CREATE TABLE <catalog>.<schema>.<table>;
SELECT 1 FROM <catalog>.<schema>.<table> LIMIT 1;
```

These checks answer most of the first-order questions:

- object exists or not
- schema is what the DAG expects or not
- user can read the object or not

---

## Common Trino / Presto Failure Patterns

| Signal | Likely bucket | Fastest confirming check |
|---|---|---|
| `SYNTAX_ERROR` | `SQL_FAILURE` | Read exact SQL from DAG / rendered log |
| `TABLE_NOT_FOUND` | `SQL_FAILURE` / `UPSTREAM_DATA` / `PERMISSION_ACCESS` | `SHOW TABLES ...` + check DAG `user` |
| `COLUMN_NOT_FOUND` | `SQL_FAILURE` | `DESCRIBE table` |
| `TYPE_MISMATCH` / `INVALID_CAST_ARGUMENT` | `SQL_FAILURE` | Compare query casts to current schema |
| `ACCESS_DENIED` | `PERMISSION_ACCESS` | Retry under same user / inspect exact permission text |
| `QUERY_EXPIRED` | `QUERY_ENGINE_TRANSIENT` | Check retry history |
| `NO_NODES_AVAILABLE` | `QUERY_ENGINE_TRANSIENT` | Check cluster/query history |
| `db_error: trino: query failed (200 OK)` | `QUERY_ENGINE_TRANSIENT` | Query accepted but failed internally |

---

## Investigation Output Expectations

A good investigation answer should include:

- the exact DAG/task being investigated
- what SQL or object reference the DAG actually used
- what user/catalog/schema it ran under
- whether the table/object exists
- whether the issue is deterministic, permission-related, upstream-data-related, or transient
- the final failure bucket and confidence
- the next exact check or owner

---

## Ownership Guide

- `SQL_FAILURE` -> DAG/query author
- `UPSTREAM_DATA` -> upstream DAG owner
- `PERMISSION_ACCESS` for Trino grants -> data team managing Trino roles/grants
- `PERMISSION_ACCESS` for S3/IAM backend access -> platform/infra team
- `QUERY_ENGINE_TRANSIENT` -> platform team if repeated across queries; DAG author adds retry if isolated

---

## Recommended Investigation Order

1. Read the DAG file and extract SQL + `user`
2. Pull exact error text and query ID from Airflow task log
3. Check retry history
4. If query ID exists, inspect query history
5. Run the smallest metadata check that resolves the ambiguity
6. Only then assign the final bucket

This keeps the investigation fast and avoids classifying too early from ambiguous Trino errors.

---

## Step 1 — Gather Minimum Context

Collect the smallest set of identifiers that unlock a useful investigation:

- DAG name / ID
- task name
- execution date
- exact error text from the Airflow task log
- query ID, if present
- whether the task retried and whether retry succeeded
- the `user` or service account used to run the query

If query ID is missing, continue with DAG code and Airflow task-log evidence.

---

## Step 2 — Read the DAG Definition First

Before reasoning about the error, inspect the DAG code and extract:

- the helper used to run the query, such as `execute_presto_query_cli`
- the exact SQL, or the function/template that renders it
- catalog and schema references
- table names and whether they are hardcoded or templated
- execution-time parameters such as `{{ ds }}`, partitions, date ranges, or environment-specific prefixes
- the `user`, connection, or service account passed to the query runner

Questions to answer from the DAG file:

1. Is this query static or generated dynamically?
2. Does it depend on upstream-created tables or partitions?
3. Could the query point to the wrong environment, catalog, or schema?
4. Is the query being run as the expected service user?

---

## Step 3 — Pull Evidence from the Airflow Task Log

From the task log, extract:

- exact error code and error message
- query ID if logged
- rendered SQL or table references if present
- whether the failure happened immediately, during planning, or after running for a while
- whether this was the first attempt or a retry

These timing signals matter:

- **Immediate failure** often means `SQL_FAILURE` or `PERMISSION_ACCESS`
- **Fails after long execution** may indicate `QUERY_ENGINE_TRANSIENT`, result-size issues, or backend instability
- **Passes on retry** strongly suggests `QUERY_ENGINE_TRANSIENT`

---

## Step 4 — Use Query History When Query ID Is Available

If you have a Trino query ID, inspect query history / UI and capture:

- final state
- failure reason and error code
- queued time
- execution time
- bytes scanned / rows scanned if available
- peak memory
- worker/coordinator error text

Use that data to separate:

- **Planning-time deterministic errors**: syntax, table missing, column missing, invalid cast
- **Execution-time deterministic errors**: division by zero, bad cast on data, semantic error that only appears after scan starts
- **Engine/transient failures**: worker crash, coordinator instability, query expired, remote page errors, internal server failures

If the same query later succeeds without SQL changes, classify as `QUERY_ENGINE_TRANSIENT`.

---

## Step 5 — Run Metadata Checks Before Classifying

These are the fastest high-signal checks for query failures.

### Check A: Does the table exist?

Use for `TABLE_NOT_FOUND` or any missing-object error.

```sql
SHOW TABLES IN <catalog>.<schema> LIKE '<table_prefix>%';
```

Interpretation:

- Table does not exist:
  - if an upstream DAG was supposed to create it -> `UPSTREAM_DATA`
  - if the query references the wrong name -> `SQL_FAILURE`
- Table exists:
  - if the DAG user still cannot access it -> `PERMISSION_ACCESS`

### Check B: Is the schema what the query expects?

Use for `COLUMN_NOT_FOUND`, `TYPE_MISMATCH`, `INVALID_CAST_ARGUMENT`.

```sql
DESCRIBE <catalog>.<schema>.<table>;
SHOW CREATE TABLE <catalog>.<schema>.<table>;
```

Interpretation:

- column absent or renamed -> `SQL_FAILURE` or `UPSTREAM_DATA` depending on whether the upstream schema changed unexpectedly
- type changed upstream and query was not updated -> usually `SQL_FAILURE`

### Check C: Does the query work under the same user?

Use for `ACCESS_DENIED`, ambiguous `TABLE_NOT_FOUND`, or cases where one person can run the query but the DAG cannot.

Interpretation:

- fails only for the DAG user -> `PERMISSION_ACCESS`
- fails for everyone with same SQL -> not a user-specific permission issue

### Check D: Did retry change the result?

Use for `QUERY_EXPIRED`, `GENERIC_INTERNAL_ERROR`, `NO_NODES_AVAILABLE`, `db_error: trino: query failed (200 OK)`.

Interpretation:

- succeeds on retry with no SQL change -> `QUERY_ENGINE_TRANSIENT`
- fails consistently -> `SQL_FAILURE`, `UPSTREAM_DATA`, or `PERMISSION_ACCESS`

---

## Decision Trees

## `TABLE_NOT_FOUND`

1. Confirm whether the table exists
2. If the table exists -> `PERMISSION_ACCESS`
3. If the table does not exist and an upstream DAG should have created it -> `UPSTREAM_DATA`
4. If the table does not exist and the query name/schema is wrong -> `SQL_FAILURE`

## `ACCESS_DENIED`

1. Does the error mention catalog/schema/table permissions? -> `PERMISSION_ACCESS`
2. Does the error point to S3/IAM or backend storage access? -> `PERMISSION_ACCESS` owned by platform/infra
3. Does the same query work under a different user? -> confirms Trino permission split

## `COLUMN_NOT_FOUND` / `TYPE_MISMATCH`

1. Compare query columns and casts with current schema
2. If query is stale or wrong -> `SQL_FAILURE`
3. If upstream schema changed unexpectedly and broke a stable DAG -> often `UPSTREAM_DATA`, but default to `SQL_FAILURE` unless upstream ownership is clear

## `QUERY_EXPIRED` / `NO_NODES_AVAILABLE` / `GENERIC_INTERNAL_ERROR`

1. Check retry history
2. Check query history for worker/coordinator instability
3. If retry succeeds without code change -> `QUERY_ENGINE_TRANSIENT`
4. If it never succeeds and there is a deterministic query error underneath -> classify by that deeper error instead

---

## Fast Query Checks

Use small, low-risk checks rather than rerunning the full expensive query immediately.

```sql
SHOW TABLES IN <catalog>.<schema> LIKE '<table_prefix>%';
DESCRIBE <catalog>.<schema>.<table>;
SHOW CREATE TABLE <catalog>.<schema>.<table>;
SELECT 1 FROM <catalog>.<schema>.<table> LIMIT 1;
```

These checks answer most of the first-order questions:

- object exists or not
- schema is what the DAG expects or not
- user can read the object or not

---

## Common Trino / Presto Failure Patterns

| Signal | Likely bucket | Fastest confirming check |
|---|---|---|
| `SYNTAX_ERROR` | `SQL_FAILURE` | Read exact SQL from DAG / rendered log |
| `TABLE_NOT_FOUND` | `SQL_FAILURE` / `UPSTREAM_DATA` / `PERMISSION_ACCESS` | `SHOW TABLES ...` + check DAG `user` |
| `COLUMN_NOT_FOUND` | `SQL_FAILURE` | `DESCRIBE table` |
| `TYPE_MISMATCH` / `INVALID_CAST_ARGUMENT` | `SQL_FAILURE` | Compare query casts to current schema |
| `ACCESS_DENIED` | `PERMISSION_ACCESS` | Retry under same user / inspect exact permission text |
| `QUERY_EXPIRED` | `QUERY_ENGINE_TRANSIENT` | Check retry history |
| `NO_NODES_AVAILABLE` | `QUERY_ENGINE_TRANSIENT` | Check cluster/query history |
| `db_error: trino: query failed (200 OK)` | `QUERY_ENGINE_TRANSIENT` | Query accepted but failed internally |

---

## Investigation Output Expectations

A good investigation answer should include:

- the exact DAG/task being investigated
- what SQL or object reference the DAG actually used
- what user/catalog/schema it ran under
- whether the table/object exists
- whether the issue is deterministic, permission-related, upstream-data-related, or transient
- the final failure bucket and confidence
- the next exact check or owner

---

## Ownership Guide

- `SQL_FAILURE` -> DAG/query author
- `UPSTREAM_DATA` -> upstream DAG owner
- `PERMISSION_ACCESS` for Trino grants -> data team managing Trino roles/grants
- `PERMISSION_ACCESS` for S3/IAM backend access -> platform/infra team
- `QUERY_ENGINE_TRANSIENT` -> platform team if repeated across queries; DAG author adds retry if isolated

---

## Recommended Investigation Order

1. Read the DAG file and extract SQL + `user`
2. Pull exact error text and query ID from Airflow task log
3. Check retry history
4. If query ID exists, inspect query history
5. Run the smallest metadata check that resolves the ambiguity
6. Only then assign the final bucket

This keeps the investigation fast and avoids classifying too early from ambiguous Trino errors.

# Debugging Steps — Detailed Procedures

## STEP 2: Check the Alerted Service's Own Metrics

### Datasink Lag Alert
```promql
# Current lag value
max(datum_recon_lag{source_name="<source_name>"})
# Trend over last 1h
max_over_time(datum_recon_lag{source_name="<source_name>"}[1h])
# Records behind — Kafka consumer lag for the datasink consumer group
sum(kafka_consumergroup_lag{consumergroup=~"<consumer_group>.*"})
# Processing rate (records/sec) — used to compute ETA
rate(datasink_total_records{consumer=~"<consumer_group>.*"}[5m])
```
**ETA calculation**: `ETA_minutes = kafka_consumer_lag / (processing_rate_records_per_sec * 60)`

### Maxwell Replication Lag Alert
```promql
# Replication lag
maxwell_metrics_replication_lag{k8s_pod=~"<pod_pattern>.*"}
# Failed messages
sum(rate(maxwell_metrics_messages_failed{k8s_pod=~"<pod_pattern>.*"}[5m]))
# Succeeded message rate
rate(maxwell_metrics_messages_succeeded{k8s_pod=~"<pod_pattern>.*"}[5m])
# Container restarts
increase(kube_pod_container_status_restarts_total{pod=~"<pod_pattern>.*"}[30m])
```

### Harvester Ingestion Lag Alert
```promql
# Spark job lag (ms since last record)
(time() * 1000) - max(harvester_v2_latest_database_record_time{index_name="<index_name>"})
# Batch processing time
max(harvester_v2_batch_processing_time{index_name="<index_name>"})
# Records behind — Kafka consumer lag for harvester consumer group
sum(kafka_consumergroup_lag{consumergroup=~"startree-harvester-v2_<index_pattern>.*"})
# Processing rate (records/sec)
rate(harvester_v2_batch_processing_time{index_name="<index_name>"}[5m])
```
**ETA calculation**: `ETA_minutes = kafka_consumer_lag / (processing_rate_records_per_sec * 60)`

### Kafka-Connect Alert
```promql
# Connector lag (ms behind source)
max(debezium_metrics_millisecondsbehindsource) by (name)
# Time since last event
debezium_metrics_millisecondssincelastevent{context="streaming"}
# Task errors
kafka_connect_task_error_total_record_errors
kafka_connect_task_error_total_record_failures
# Heap memory usage
sum(jvm_memory_bytes_used{area="heap", k8s_pod=~"kafka-connect-.*"}) by (k8s_pod) / sum(jvm_memory_bytes_max{area="heap", k8s_pod=~"kafka-connect-.*"}) by (k8s_pod)
```

### Staging-Table-Pipeline Lag Alert
```promql
# Per-table commit time lag (ms since last processed record)
(time() * 1000) - stc_v2_db_tbl_max_commit_time{workspace="<workspace>", database="<database>", table="<table>"}
# Batch processing time
stc_v2_batch_processing_time_seconds{bu="<bu>"}
# Batch duration (ms)
stc_v2_duration_ms{bu="<bu>"}
# Row throughput per batch
stc_v2_num_input_rows{bu="<bu>"}
# Error vs processed records
stc_v2_error_record_count{bu="<bu>"}
stc_v2_process_record_count{bu="<bu>"}
# Records behind — Kafka consumer lag for the staging pipeline consumer group
sum(kafka_consumergroup_lag{consumergroup=~"entity-operator-<bu>.*"})
```
**ETA calculation**: Derive processing rate as `stc_v2_num_input_rows / (stc_v2_duration_ms / 1000)` rows/sec. Then `ETA_minutes = kafka_consumer_lag / (rows_per_sec * 60)`.

If `stc_v2_error_record_count` is growing → check DLQ topic `staging_cdc_{bu}_error_topic` for error patterns.

### Replication-Table-Pipeline Lag Alert
```promql
# PRIMARY LAG METRIC — ms since last committed timestamp per table
rtp_v2_max_ts_ms{database="<database>", table="<table>", bu="<bu>"}
# Records per batch (liveness + throughput)
rtp_v2_total_records{database="<database>", table="<table>"}
# Batch processing duration
rtp_v2_batch_duration{database="<database>", table="<table>"}
# Overall batch duration by BU
rtp_v2_duration_ms{bu="<bu>"}
# Query delay — time spread in batch (high = catching up after a pause)
rtp_v2_query_delay{database="<database>", table="<table>"}
# Sidelined table counter (table removed from processing)
stp_v2_counter{database="<database>", table="<table>"}
```
**ETA calculation**: Use `rtp_v2_max_ts_ms` lag value (ms) divided by `rtp_v2_total_records / rtp_v2_batch_duration` (records per ms) to estimate records behind, then `ETA_minutes = records_behind / (records_per_sec * 60)`.

If `stp_v2_counter` is non-zero → table has been sidelined. **Immediately run the Trino sidelined-table investigation below.**
If `rtp_v2_query_delay` is very high → the batch is reading a wide time range, likely catching up after a job restart or pause.

### Sidelined Table Investigation (Trino MCP)

**Run this whenever `stp_v2_counter > 0` for any table.**

First probe Trino MCP availability by attempting the query. If the tool is not present or returns an error, note "Trino MCP unavailable — sidelined reason unknown, check `cdc_replication_metadata_v2.rzp_{bu}_bu` manually" in the RCA and proceed without this step.

**Trino query** (replace `{bu}` with the BU from the alert, e.g. `evolvehq`, `rx`, `payment`):
```sql
SELECT db_name, table_name, job_name, meta_data, is_active, status, bu_name, job_priority, sidelined_at
FROM cdc_replication_metadata_v2.rzp_{bu}_bu
WHERE status = 'SIDELINED'
  AND meta_data != '{}'
```

The `meta_data` column is a JSON object. Extract:
- `sideline_reason` — short error type (e.g. `StreamingQueryException`, `SchemaEvolutionException`)
- `exception_class` — full Java class name of the exception
- `sidelined_at` — ISO timestamp when the table was sidelined
- `stack_trace` — first line of the stack trace (the most informative part)

**Diagnosis by sideline_reason:**
- `StreamingQueryException` — Spark Streaming query crashed on this table. Usually a schema mismatch, OOM on a specific partition, or a corrupt record. **Fix: identify the bad record or schema drift, then reactivate via entity-operator config.**
- `SchemaEvolutionException` — Source table schema changed (new column, type change) in a way the pipeline couldn't auto-handle. **Fix: update the schema config for this table in entity-operator, then reactivate.**
- Any other exception — treat as an application-level bug. Include the `exception_class` and first stack frame in the RCA.

**Always include in the RCA paragraph:**
- The table name and BU
- The `sideline_reason` and `sidelined_at` timestamp
- How long the table has been sidelined (`now - sidelined_at`)
- Whether `is_active = true` (table should be running but isn't) or `is_active = false` (intentionally disabled)
- The recommended fix action

## STEP 3: Check Upstream Dependencies

### If Datasink is lagging → Check Maxwell / Kafka-Connect upstream

Use the **Topic-to-CDC-Producer Mapping** in `topic-mappings.md` to identify which Maxwell or Kafka-Connect produces the topic this Datasink job consumes.

```promql
# Check Maxwell for the source database
maxwell_metrics_replication_lag{k8s_pod=~"datahub-<db_name>-maxwell.*"}
# Check data-streams transformation lag (if topic is a transformed topic)
kafka_consumergroup_lag{consumergroup=~"payment_card_transformation.*"}
```

### If Harvester is lagging → Check Datasink TiDB lag (lookup dependency)

```promql
max(datum_recon_lag{source_name=~"tidb_.*"}) by (source_name)
max(datum_recon_lag{source_name="tidb_admin_api_heartbeat"})
max(datum_recon_lag{source_name="tidb_de_reporting_api_payments"})
```

### If Datasink is lagging → Check TiDB health (write target)

```promql
histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket{cluster="eks-prod-white-infra"}[5m])) by (le, sql_type))
sum(increase(tikv_engine_write_stall_reason{}[5m])) by (type, cluster)
sum(tikv_engine_pending_compaction_bytes{cluster="eks-ops-common-infra"}) by (cf)
100 - (100 * (sum(pd_scheduler_store_status{type="store_available"}) / sum(pd_scheduler_store_status{type="store_capacity"})))
kube_statefulset_status_replicas_ready{statefulset=~"tidb-.*"}
```

### If Staging-Table-Pipeline is lagging → Check upstream (Maxwell or TiCDC depending on BU)

**FIRST: Identify the BU.** The upstream differs fundamentally:
- **shared BU** → upstream is **TiCDC** (NOT Maxwell). See "Shared BU TiCDC upstream check" below.
- **all other BUs** → upstream is **Maxwell / Kafka-Connect** (standard CDC path).

**For non-shared BUs:**
```promql
maxwell_metrics_replication_lag{k8s_pod=~"datahub-<db_name>-maxwell.*"}
max(debezium_metrics_millisecondsbehindsource) by (name)
kafka_consumergroup_lag{consumergroup=~"<staging_consumer_group>.*"}
```

If Maxwell/Kafka-Connect is lagging → staging pipeline is starved of input. Root cause is upstream CDC, not the staging job.

**For shared BU** — check TiCDC upstream:
```promql
kafka_server_brokertopicmetrics_messagesin_total{topic="ticdc-rzp-payment-api"}
kafka_consumergroup_lag{consumergroup=~"entity-operator-shared.*"}
datasink_total_records{consumer=~"datasink_mysql_consumer.*"}
```

If `messagesin_total` for `ticdc-rzp-payment-api` is near zero → TiCDC is not producing. Full cascade: Datasink → TiDB (rzp-payment-api) → TiCDC → shared BU staging.

### If Replication-Table-Pipeline is lagging → Check Staging-Table-Pipeline upstream

```promql
stc_v2_duration_ms{bu="<bu>"}
stc_v2_db_tbl_max_commit_time{workspace="<workspace>", database="<database>", table="<table>"}
```

If staging is stale → replication has no new data. Trace further upstream to Maxwell/Kafka-Connect.
If staging is fresh but replication is lagging → replication pipeline itself has issues (slow MERGE, schema drift, sidelined table).

### Full Datalake Upstream Cascade (run in parallel where possible)

1. Check replication metrics (`rtp_v2_max_ts_ms`) → if lagging, go to step 2
2. Check staging metrics (`stc_v2_db_tbl_max_commit_time`) → if lagging, go to step 3
3. Check Maxwell/Kafka-Connect → if lagging, root cause is CDC producer
4. Check source DB write rate → if zero, source is silent (not a DP issue — escalate to owning team)

### If Maxwell is lagging → Check Maxwell resource usage

```promql
sum(maxwell_metrics_jvm_memory_usage_heap_used{k8s_pod=~"<pod_pattern>.*"}) / sum(maxwell_metrics_jvm_memory_usage_total_max{k8s_pod=~"<pod_pattern>.*"})
sum(container_memory_usage_bytes{namespace=~"maxwell.*", pod=~"<pod_pattern>.*"}) by (pod)
sum(rate(container_cpu_usage_seconds_total{namespace=~"maxwell.*", pod=~"<pod_pattern>.*"}[5m])) by (pod)
kafka_producer_producer_metrics_record_error_total{k8s_pod=~"<pod_pattern>.*"}
kafka_producer_producer_metrics_buffer_exhausted_rate{k8s_pod=~"<pod_pattern>.*"}
kafka_producer_producer_metrics_request_latency_avg{k8s_pod=~"<pod_pattern>.*"}
```

# Alert Thresholds Reference

All alert rules with PromQL expressions and thresholds for the CDC pipeline services.

## Datasink Lag Alerts

Source: `de_cdc_pipeline_rules.yml`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| Admin Shared API lag | `max(datum_recon_lag{source_name="tidb_admin_api_heartbeat"})` | > 300s | 5m | critical |
| DE-Reporting Shared API lag | `max(datum_recon_lag{source_name="tidb_de_reporting_api_payments"})` | > 300s | 5m | critical |
| Prod-White Shared API lag | `max(datum_recon_lag{source_name="tidb_prod_white_api_payments"})` | > 600s | 5m | critical |
| Admin Decomp lag | `max(datum_recon_lag{source_name="tidb_admin_api_customers"})` | > 600s | 5m | critical |
| DE-Reporting Decomp lag | `max(datum_recon_lag{source_name="tidb_de_reporting_api_customers"})` | > 600s | 5m | critical |
| Admin Test-Databases lag | `max(datum_recon_lag{source_name="tidb_admin_api_test_heartbeat"})` | > 600s | 5m | warning |
| Admin PaymentsInfra Live lag | `max(datum_recon_lag{source_name="tidb_admin_ledger_live_journal"})` | > 300s | 5m | critical |
| DE-Reporting PaymentsInfra Live lag | `max(datum_recon_lag{source_name="tidb_de_reporting_ledger_live_journal"})` | > 300s | 5m | critical |
| Admin PG Router lag | `max(datum_recon_lag{source_name="tidb_admin_pg_router_orders"})` | > 300s | 5m | critical |
| Admin Payments UPS lag | `max(datum_recon_lag{source_name="tidb_admin_upi_payments"})` | > 600s | 5m | critical |
| DE-Reporting Payments UPS lag | `max(datum_recon_lag{source_name="tidb_de_reporting_upi_payments"})` | > 600s | 5m | critical |
| Recon lag metrics unavailable | `absent(datum_recon_lag) == 1` | N/A | 15m | warning |

## Maxwell Alerts

Source: `de_cdc_pipeline_rules.yml`, `data_replication_rules.yaml`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| Replication lag | `maxwell_metrics_replication_lag` | > 300s | varies | critical |
| High Container Restarts P1 | `increase(kube_pod_container_status_restarts_total{pod=~".*maxwell.*"}[30m])` | > 5 | 5m | warning |
| High Container Restarts P0 | `increase(kube_pod_container_status_restarts_total{pod=~".*maxwell.*"}[30m])` | > 10 | 5m | critical |
| Message Push Failed | `sum(rate(maxwell_metrics_messages_failed{...}[15m])*90)` | > 2 | 2m | critical |
| Non-Functional (recon) | `maxwell_metrics_custom_transaction_commit_time - datum_database_freshness_metric` | > 3600s (1h) | varies | critical |

## Harvester Ingestion Alerts

Source: `de_harvester_v2_pinot_rules.yaml`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| Payments Index Lag | `max_over_time(datum_recon_lag{source_name="pinot_payments"}[5m])` | > 300s | 2m | critical |
| SR View Lag | `max_over_time(datum_recon_lag{source_name="pinot_sr_view"}[5m])` | > 300s | 2m | critical |
| Thirdeye SR Lag | `max_over_time(datum_recon_lag{source_name="pinot_thirdeye_sr"}[5m])` | > 300s | 2m | critical |
| Low Volume Index Lag | `max(datum_recon_lag{source_name=~"pinot_refunds\|pinot_balances\|pinot_settlements_fo\|pinot_contacts"})` | > 450s | 20m | warning |
| High Volume Jobs Lag | `avg_over_time(datum_recon_lag{source_name=~"pinot_payouts\|pinot_orders\|pinot_magic_orders"}[5m])` | > 600s | 2m | warning |
| transactions_razorpay_x Lag | `max(datum_recon_lag{source_name="pinot_transactions_razorpay_x"})` | > 14400s (4h) | 20m | critical |
| checkout_order_analytics Lag | `((time() * 1000) - max(harvester_v2_latest_database_record_time{index_name="checkout_order_analytics"}))` | > 300000ms | 5m | critical |
| Kafka Consumer Processing Lag | `sum by (consumergroup) (rate(kafka_consumergroup_current_offset{consumergroup=~"startree-harvester-v2_..."}[5m]))` | > 3000 msgs/sec | 10m | critical |

## Harvester Web Alerts

Source: `de_harvester_web_rules.yaml`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| 5xx Error Rate | `http_responses_total{code=~"5.."}` / `http_requests_total` | > 2% | 5m | critical |
| Response Time P99 | `histogram_quantile(0.99, http_durations_ms_histogram_bucket{code="200", server="PqlService"})` | > 70000ms | 2m | critical |
| Response Time Warning | Same as above | > 30000ms | 2m | warning |
| Pinot Query Errors | `sum(rate(harvester_v2_querysink_failures{query_sink="pinot", err_code=~"5(.*)"}[5m]) * 60)` | > 10/min | 3m | critical |
| Pinot Query Exec Time | `histogram_quantile(0.99, harvester_v2_querysink_execution_time_bucket{query_sink="pinot"})` | > 25000ms | 2m | warning |
| Replicas Down | `count(go_goroutines{k8s_pod=~"harvester-v2-web.*"})` | < 1 | 30s | critical |
| Metrics Unavailable | `absent(go_goroutines{k8s_pod=~"harvester-v2-web.*"}) == 1` | N/A | 2m | critical |
| Container Restarts | `increase(kube_pod_container_status_restarts_total{pod=~".*harvester-v2-web.*"}[1h])` | > 5 | 1m | warning |
| Available Replicas | `kube_deployment_status_replicas_available{deployment="harvester-v2-web"}` | < 1 | 30s | critical |
| Traefik 5xx | `sum(increase(traefik_router_requests_total{service=~".*harvester-v2-web.*", code=~"[5].."}[1m]))` | > 10 | 5m | critical |

## Kafka-Connect Alerts

Source: `de_cdc_pipeline_rules.yml`, `prod_ops_kafka_alerts.yaml`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| Workers Available P0 | `kube_deployment_status_replicas_available{deployment="kafka-connect-cluster-connect"}` | < 2 | immediate | critical |
| Workers Available P1 | `kube_deployment_status_replicas_available{deployment="kafka-connect-cluster-connect"}` | < 3 | immediate | warning |
| Replicas Metrics Unavailable | `absent(kube_deployment_status_replicas_available{deployment="kafka-connect-cluster-connect"}) == 1` | N/A | immediate | critical |
| Heap Memory P1 | `jvm_memory_bytes_used{area="heap"} / jvm_memory_bytes_max{area="heap"}` | > 80% | immediate | warning |
| Heap Memory P0 | Same | > 90% | immediate | critical |
| Connector Failed | `kafka_connect_connector_status{status="failed"} == 1` | == 1 | immediate | critical |
| Ops-Kafka High Disk P0 | `kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes * 100` | > 90% | 30s | critical |
| Ops-Kafka High Disk P1 | Same | > 80% | 10m | warning |
| Ops-Kafka Broker Count P0 | `kube_statefulset_status_replicas_ready{statefulset="kafka-kafka"}` | < 4 | 10s | critical |
| Ops-Kafka Broker Count P1 | Same | < 5 | 10m | warning |

## Entity-Operator Staging-Table-Pipeline Alerts

Source: entity-operator alerting rules

Alert pattern: `[DE][Entity][Staging] {BU} BU lag alert`

Fields in alert: `alertname`, `alertgroup`, `bu`, `jobtype=staging`, `severity`, `slack_channel`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| Staging Per-Table Lag P1 | `(time()*1000) - stc_v2_db_tbl_max_commit_time{workspace="<workspace>"}` | > 1h (3600s) | 5m | warning |
| Staging Per-Table Lag P0 | `(time()*1000) - stc_v2_db_tbl_max_commit_time{workspace="<workspace>"}` | > 3h (10800s) | 5m | critical |
| Staging Batch Stall P1 | `absent(stc_v2_duration_ms{bu="<bu>"})` | absent > 15m | 15m | warning |
| Staging Batch Stall P0 | `absent(stc_v2_duration_ms{bu="<bu>"})` | absent > 30m | 30m | critical |
| Staging Error Records | `stc_v2_error_record_count{bu="<bu>"}` | > 0 sustained | 5m | warning |

Key metrics for debugging:
- `stc_v2_duration_ms` — batch duration (liveness indicator)
- `stc_v2_num_input_rows` — rows per batch (throughput)
- `stc_v2_db_tbl_max_commit_time{workspace, database, table}` — per-table freshness
- `stc_v2_process_record_count` / `stc_v2_error_record_count` — success vs error ratio
- `stc_v2_batch_processing_time_seconds` — processing time distribution

## Entity-Operator Replication-Table-Pipeline Alerts

Source: entity-operator alerting rules

Alert pattern: `[DE][Entity][Replication] {BU} BU lag alert`

Fields in alert: `alertname`, `alertgroup`, `bu`, `jobtype=replication`, `severity`, `slack_channel`, `database`, `table`

| Alert | Metric | Threshold | Duration | Severity |
|---|---|---|---|---|
| Replication Lag P1 | `(time()*1000) - rtp_v2_max_ts_ms{database="<db>", table="<tbl>", bu="<bu>"}` | > 1h (3600000ms) | 5m | warning |
| Replication Lag P0 | `(time()*1000) - rtp_v2_max_ts_ms{database="<db>", table="<tbl>", bu="<bu>"}` | > 2h (7200000ms) | 5m | critical |
| Replication Batch Stall P1 | `absent(rtp_v2_duration_ms{bu="<bu>"})` | absent > 15m | 15m | warning |
| Replication Batch Stall P0 | `absent(rtp_v2_duration_ms{bu="<bu>"})` | absent > 30m | 30m | critical |
| Table Sidelined | `stp_v2_counter{database="<db>", table="<tbl>"}` | > 0 | immediate | warning |

Key metrics for debugging:
- `rtp_v2_max_ts_ms{database, table, bu}` — PRIMARY lag metric (current_ts - max committed ts)
- `rtp_v2_total_records{database, table}` — records per batch (liveness)
- `rtp_v2_batch_duration{database, table}` — batch processing time
- `rtp_v2_duration_ms{bu}` — overall batch duration
- `rtp_v2_query_delay{database, table}` — time spread in batch (high = catching up)
- `stp_v2_counter{database, table}` — sidelined table counter (table removed from processing)

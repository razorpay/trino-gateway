# CDC Pipeline Topic Mappings

Complete mapping of CDC producers to Kafka topics to consumers, for tracing alerts back to root cause.

## Data-Streams: Kafka-to-Kafka Transformations

These intermediate topics are produced by data-streams (Kafka Streams apps) and consumed by Harvester. If a Harvester index consumes a transformed topic, a failure in data-streams can cause Harvester lag even if the original CDC producer is healthy.

| Transformed Topic (Output) | Source Topic (Input) | Stream App | Source DB |
|---|---|---|---|
| `internal_db_payments_card_transformed_api` | `cdc_postgres_rzp_payment_pgpayments_card_live` | PaymentsCardStreamApp | PostgreSQL (Kafka-Connect) |
| `internal_api_payments_nbplus_live_transformed` | `cdc_rzp_payment_payments_nbplus_live` | PaymentsNbplusStreamApp | MySQL (Maxwell: payments-nbplus) |
| `internal_api_payments_upi_live_transformed` | `cdc_rzp_payment_payments_upi_live` | PaymentsUpiStreamApp | MySQL (Maxwell: payments-upi) |
| `internal_api_payments_emandate_live_transformed` | `cdc_rzp_payment_prod_emandate_live` | PaymentsEmandateStreamApp | MySQL (Maxwell: prod-emandate-live) |
| `internal_api_payments_card_present_live_transformed` | `cdc_rzp_payment_prod_payments_card_present` | PaymentsCardPresentStreamApp | MySQL (Maxwell: payments-card-present) |
| `internal_api_optimizer_core_live_transformed` | `cdc_postgres_rzp_payment_optimizer_core_live` | PaymentsOptimizerStreamApp | PostgreSQL (Kafka-Connect) |
| `internal_api_pg_router_orders_v1` | `cdc_rzp_payment_prod_pg_router` | PGRouterTopologyBuilder | MySQL (Maxwell: pg-router) |

## Maxwell Deployments to Kafka Topics

Most Maxwell deployments use default topic routing: `cdc_%{workspace_tag}_%{database}`.

| Maxwell Deployment | Source Database | Kafka Topic Pattern | Kafka Cluster |
|---|---|---|---|
| maxwell-api | api | `cdc_rzp_payment_api` | WHITE_KAFKA |
| maxwell-payments-upi | payments_upi_live | `cdc_rzp_payment_payments_upi_live` | WHITE_KAFKA |
| maxwell-payments-mandate | payments_mandate_live | `cdc_rzp_payment_payments_mandate_live` | WHITE_KAFKA |
| maxwell-scrooge | scrooge | `cdc_rzp_payment_scrooge-live` | WHITE_KAFKA |
| maxwell-pg-router | pg_router | `cdc_rzp_payment_prod_pg_router` | WHITE_KAFKA |
| maxwell-nbplus | payment_nbplus | `cdc_rzp_payment_payments_nbplus_live` | WHITE_KAFKA |
| maxwell-prod-emandate-live | prod_emandate_live | `cdc_rzp_payment_prod_emandate_live` | WHITE_KAFKA |
| maxwell-payments-card-present | payments_card_present | `cdc_rzp_payment_prod_payments_card_present` | WHITE_KAFKA |
| maxwell-capital | capital | default routing | WHITE_KAFKA |
| maxwell-fts | fts | default routing | WHITE_KAFKA |
| maxwell-subscriptions | subscriptions | default routing | WHITE_KAFKA |
| maxwell-prod-account-service | prod_account_service | default routing | WHITE_KAFKA |
| maxwell-prod-payouts | prod_payouts | default routing | WHITE_KAFKA |
| maxwell-disputes | prod-disputes | default routing | WHITE_KAFKA |
| maxwell-api-templatised | api_templatised | `templatised_cdc_%{workspace_tag}_%{database}` | WHITE_KAFKA |
| maxwell-templatised-reporting-api | templatised_reporting_api | `templatised_realtime_reporting_warehouse_%{database}` | WHITE_KAFKA |
| maxwell-reporting-api | reporting_api | default routing | OPS_KAFKA |
| maxwell-unfiltered-api | unfiltered_api | default routing | WHITE_KAFKA |
| maxwell-tokens-tidb | tokens_tidb | default routing | WHITE_KAFKA |

## Kafka-Connect Connectors to Kafka Topics

All connectors use PostgreSQL Debezium with `pgoutput` plugin.

| Connector | Source Database | Output Topic |
|---|---|---|
| payments_card_db_connector | pgpayments_card_live | `internal_db_payments_card` |
| payments_card_db_connector_rds | pgpayments_card_live | `internal_db_payments_card_rds` |
| settlements_live_db_connector | SETTLEMENTS_LIVE | `internal_db_settlements_live` |
| settlements_live_db_connector_ops | SETTLEMENTS_LIVE | `realtime_reporting_settlements_live` |
| rx_ledger_live_db_connector | PROD_LEDGER_LIVE | `internal_db_prod_rx_ledger_live` |
| ledger_live_payments_db_connector | PROD_PAYMENTS_LEDGER_LIVE | `internal_db_prod_payments_ledger_live` |
| optimizer_core_live_db_connector | OPTIMIZER_CORE_LIVE | `cdc_postgres_rzp_payment_optimizer_core_live` |
| offers_db_connector | PROD_OFFERS_ENGINE | `cdc_postgres_rzp_evolvehq_prod_offers_engine` |
| terminals_db_connector | TERMINALSLIVE | `internal_db_terminals_live` |
| terminals_live_db_connector_tidb | TERMINALSLIVE | `realtime_reporting_terminals_live` |
| batch_db_connector_v01 | batch | `internal_db_batch` |
| recon_db_connector | RECON | `internal_db_recon` |
| pg_router_db_connector | PROD_PG_ROUTER | `internal_db_pg_router` |

## Datasink Job Types to Topics to TiDB Clusters

| Job Type | Kafka Topics (abbreviated) | Consumer Group | Target TiDB |
|---|---|---|---|
| `shared_jobtype.admin_api` | `realtime_reporting_warehouse_api` | `datasink_mysql_consumer_admin_api_v1` | ops-common-tidb |
| `shared_jobtype.de_reporting_api` | `realtime_reporting_warehouse_api` | `datasink_mysql_consumer_de_reporting` | prod-de-white-tidb |
| `shared_jobtype.prod-white-mum_api` | `realtime_reporting_warehouse_api` | `datasink_mysql_consumer_prod-white-mum-5` | prod-white-mum-tidb |
| `paymentsinfra_jobtype.admin_live_databases` | `cdc_postgres_rzp_platform_prod_pg_ledger_live,...` | `datasink_mysql_consumer_admin_live_v01` | ops-common-tidb |
| `paymentsinfra_jobtype.de_reporting_live_databases` | `cdc_postgres_rzp_platform_prod_pg_ledger_live,...` | `datasink_mysql_consumer_de_reporting_live_v01` | prod-de-white-tidb |
| `payments_jobtype.admin_pg_router` | `cdc_rzp_payment_prod_pg_router,bcpdr_hyd_...` | `datasink_mysql_consumer_admin_pg_router_v0` | ops-common-tidb |
| `payments_jobtype.de_reporting_pg_router` | `cdc_rzp_payment_prod_pg_router,bcpdr_hyd_...` | `datasink_mysql_consumer_de_reportong_pg_router_v0` | prod-de-white-tidb |
| `payments_jobtype.admin_payments_ups` | `cdc_rzp_payment_payments_upi_live,bcpdr_hyd_...` | `datasink_mysql_consumer_admin_payments_ups_v0` | ops-common-tidb |
| `payments_jobtype.de_reporting_payments_ups` | `cdc_rzp_payment_payments_upi_live,bcpdr_hyd_...` | `datasink_mysql_consumer_de_reporting_payments_ups_v0` | prod-de-white-tidb |
| `payments_jobtype.admin_payments_mandates_live` | `cdc_rzp_payment_payments_mandate_live` | `datasink_mysql_consumer_admin_payments_mandates_live_v0` | ops-common-tidb |
| `payments_jobtype.de_reporting_payments_mandates_live` | `cdc_rzp_payment_payments_mandate_live` | `datasink_mysql_consumer_de_reporting_payments_mandates_live_v0` | prod-de-white-tidb |
| `payments_jobtype.admin_payments_card_rds` | `cdc_postgres_rzp_payment_pgpayments_card_live,...` | `datasink_mysql_consumer_admin_payments_card_rds_v0` | ops-common-tidb |
| `payments_jobtype.de_reporting_payments_card_rds` | `cdc_postgres_rzp_payment_pgpayments_card_live,...` | `datasink_mysql_consumer_de_reporting_payments_card_rds_v0` | prod-de-white-tidb |
| `rx_jobtype.admin_rx_payouts` | `cdc_rzp_rx_prod_payouts` | `datasink_mysql_consumer_admin_rx_payouts_v0` | ops-common-tidb |
| `rx_jobtype.de_reporting_rx_payouts` | `cdc_rzp_rx_prod_payouts` | `datasink_mysql_consumer_de_reporting_rx_payouts_v0` | prod-de-white-tidb |
| `settlements_jobtype.settlements_warm_db` | `prod_internal_db_settlements_live_warm` | `datasink_postgres_consumer_settlements_warm_db_v1` | PostgreSQL (warm) |
| `shared_jobtype.admin_decomp_databases` | `cdc_postgres_rzp_evolvehq_prod_offers_engine,...` | `datasink_mysql_consumer_admin_decomp_v01` | ops-common-tidb |
| `shared_jobtype.de_reporting_decomp_databases` | `cdc_postgres_rzp_evolvehq_prod_offers_engine,...` | `datasink_mysql_consumer_de_reporting_decomp_v01` | prod-de-white-tidb |
| `payments_jobtype.admin_p2_live_databases` | `cdc_rzp_payment_scrooge-live,...` | `datasink_mysql_consumer_admin_p2_live_databases_v0` | ops-common-tidb |
| `payments_jobtype.admin_dispute_service` | `cdc_rzp_payment_prod_dispute_service` | `datasink_mysql_consumer_admin_dispute_service_v0` | ops-common-tidb |
| `payments_jobtype.admin_prod_payments_bank_transfer_live` | `cdc_rzp_payment_prod_payments_bank_transfer_live` | `datasink_mysql_consumer_admin_prod_payments_bank_transfer_live_v1` | ops-common-tidb |

## Datasink source_name to Job Type Mapping

For tracing alerts: the `datum_recon_lag` metric's `source_name` label maps to TiDB tables written by specific Datasink jobs.

| source_name | TiDB Cluster | Datasink Job Type |
|---|---|---|
| `tidb_admin_api_heartbeat` | ops-common | `shared_jobtype.admin_api` |
| `tidb_admin_api_test_heartbeat` | ops-common | `shared_jobtype.admin_test_databases` |
| `tidb_de_reporting_api_payments` | de-reporting | `shared_jobtype.de_reporting_api` |
| `tidb_prod_white_api_payments` | prod-white-mum | `shared_jobtype.prod-white-mum_api` |
| `tidb_admin_ledger_live_journal` | ops-common | `paymentsinfra_jobtype.admin_live_databases` |
| `tidb_de_reporting_ledger_live_journal` | de-reporting | `paymentsinfra_jobtype.de_reporting_live_databases` |
| `tidb_admin_pg_router_orders` | ops-common | `payments_jobtype.admin_pg_router` |
| `tidb_de_reporting_pg_router_orders` | de-reporting | `payments_jobtype.de_reporting_pg_router` |
| `tidb_admin_upi_payments` | ops-common | `payments_jobtype.admin_payments_ups` |
| `tidb_de_reporting_upi_payments` | de-reporting | `payments_jobtype.de_reporting_payments_ups` |
| `tidb_admin_api_customers` | ops-common | `shared_jobtype.admin_decomp_databases` |
| `tidb_de_reporting_api_customers` | de-reporting | `shared_jobtype.de_reporting_decomp_databases` |

## Harvester Ingestion Indexes to Topics to Lookup Dependencies

| Index | Primary Table | Key Kafka Topics | TiDB Lookup Tables | Lag Check |
|---|---|---|---|---|
| `payments` | api.payments | `cdc_rzp_payment_api`, `internal_db_payments_card_transformed_api`, `internal_api_payments_nbplus_live_transformed`, `internal_api_payments_upi_live_transformed`, `internal_api_optimizer_core_live_transformed`, `internal_api_payments_emandate_live_transformed`, `internal_api_payments_card_present_live_transformed` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `refunds` | scrooge-live.refunds | `cdc_rzp_payment_scrooge-live`, `bcpdr_hyd_cdc_rzp_payment_scrooge-live` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `settlements` | api.settlements | `cdc_rzp_payment_api` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `settlements_fo` | settlements_live.settlements | `cdc_postgres_rzp_payment_settlements_live` | api-reporting-tidb, settlements-live-aurora, fts-aurora | Multiple |
| `balances` | api.balance | `cdc_rzp_payment_api` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `contacts` | api.contacts | `cdc_rzp_payment_api` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `batches` | api.batches | `cdc_rzp_payment_api` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `commissions` | api.commissions | `cdc_rzp_payment_api` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `payouts` | prod_payouts.payouts_temp | `cdc_rzp_rx_prod_payouts` | api-reporting-tidb, fts-aurora | `api.heartbeat`, FTS |
| `orders` | api.orders | `cdc_rzp_payment_api`, `internal_api_pg_router_orders_v1` | None | N/A |
| `magic_orders` | api.orders | `cdc_rzp_payment_api`, `internal_api_pg_router_orders_v1` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `sr_view` | api.payments | Same as payments + card topics | api-prodwhite-tidb, api-admin-tidb, pc-prodwhite-tidb, pc-admin-tidb | Multiple |
| `thirdeye_sr` | api.payments | Same as payments + card topics | api-reporting-tidb, api-admin-tidb, pc-reporting-tidb, pc-admin-tidb | Multiple |
| `transactions_razorpay_x` | api.transactions | `cdc_rzp_payment_api` | api-reporting-tidb, api-admin-tidb | `api.heartbeat` |
| `checkout_order_analytics` | api.orders | `internal_api_pg_router_orders_v1` | api-reporting-tidb, api-admin-tidb | `api.order_meta` |

## Harvester Lookup Table to TiDB Host Mapping

| Lookup Config Name | TiDB Host | Database | Lag Check Query |
|---|---|---|---|
| api-reporting-tidb | ops-common-tidb.razorpay.com:4000 | api | `SELECT FLOOR(UNIX_TIMESTAMP(max(ts))) as max_time FROM api.heartbeat` |
| api-admin-tidb | ops-common-tidb.razorpay.com:4000 | api | Same |
| api-prodwhite-tidb | prod-white-mum-tidb.razorpay.com:4000 | api | Same |
| api-test-reporting-tidb | ops-common-tidb.razorpay.com:4000 | api-test | `SELECT FLOOR(UNIX_TIMESTAMP(max(ts))) as max_time FROM api-test.heartbeat` |
| pc-reporting-tidb | ops-common-tidb.razorpay.com:4000 | pgpayments_card_live | `SELECT created_at as max_time FROM pgpayments_card_live.payments ORDER BY id DESC LIMIT 1` |
| pc-prodwhite-tidb | prod-white-mum-tidb.razorpay.com:4000 | pgpayments_card_live | Same |
| fts-aurora | prod-aurora-mysql-fts-razorpayx (RDS) | fts-live | `SELECT UNIX_TIMESTAMP() as max_time` |
| settlements-live-aurora | PostgreSQL RDS | settlements_live | `SELECT cast(EXTRACT(EPOCH FROM NOW()) as int) as max_time` |

## Consumer Group Patterns

**Harvester (Pinot):** `startree-harvester-v2_<index_name>_<mode>_v<version>`
- `startree-harvester-v2_settlements_live_v.*`
- `startree-harvester-v2_batches_live_v.*`
- `startree-harvester-v2_refunds_live_v.*`
- `startree-harvester-v2_commissions_live_v.*`
- `startree-harvester-v2_balances_live_v.*`
- `startree-harvester-v2_contacts_live_v.*`

**Datasink:** `datasink_<db_type>_consumer_<target_name>_v<version>`
- See Datasink Job Types table above for exact consumer group per job type.

## EMR Clusters: AWS Account & Naming Patterns

> **All EMR clusters (entity-operator, datasink, harvester) run in the `qubole` AWS account.**
> Always specify `account_alias: qubole` when using Friday MCP for any EMR lookup.

### Entity-Operator Staging Cluster Names

**Pattern**: `entity_operator_staging_{size_tier}_graviton[_ticdc]_entityoperator_{workspace}_staging`

| BU | Full Cluster Name |
|---|---|
| payment | `entity_operator_staging_payment_graviton_entityoperator_payment_staging` |
| rx | `entity_operator_staging_common_graviton_entityoperator_rx_staging` |
| platform | `entity_operator_staging_common_graviton_entityoperator_platform_staging` |
| capital | `entity_operator_staging_common_graviton_entityoperator_capital_staging` |
| evolvehq | `entity_operator_staging_common_graviton_entityoperator_evolvehq_staging` |
| data | `entity_operator_staging_common_graviton_entityoperator_data_staging` |
| shared | `entity_operator_staging_common_graviton_ticdc_entityoperator_shared_staging` |

**Tip**: The alert `app_name` label (e.g. `entityoperator_rx_staging`) is the suffix of the full cluster name. Use `contains(Name, \`<app_name>\`)` in Friday MCP queries to resolve the exact cluster without knowing the cluster ID.

### Entity-Operator Replication Cluster Names

**Pattern**: `entity_operator_replication_{size_tier}_graviton[_ticdc]_entityoperator_{workspace}_replication_job_{N}`

Where `{size_tier}` maps by BU: `mid_large_plus` (payment) → `mid_large` (platform) → `medium` (rx) → `small_plus` (evolvehq) → `small` (capital, data). TiCDC clusters (`_ticdc` suffix) are used only for the `shared` BU.

| BU | Full Cluster Name |
|---|---|
| payment | `entity_operator_replication_mid_large_plus_graviton_entityoperator_payment_replication_job_{N}` |
| platform | `entity_operator_replication_mid_large_graviton_entityoperator_platform_replication_job_{N}` |
| rx | `entity_operator_replication_medium_graviton_entityoperator_rx_replication_job_{N}` |
| evolvehq | `entity_operator_replication_small_plus_graviton_entityoperator_evolvehq_replication_job_{N}` |
| capital | `entity_operator_replication_small_graviton_entityoperator_capital_replication_job_{N}` |
| data | `entity_operator_replication_small_graviton_entityoperator_data_replication_job_{N}` |
| shared (job 1) | `entity_operator_replication_small_graviton_ticdc_entityoperator_shared_replication_job_1` |
| shared (job 2) | `entity_operator_replication_medium_graviton_ticdc_entityoperator_shared_replication_job_2` |
| shared (job 3+) | `entity_operator_replication_large_graviton_ticdc_entityoperator_shared_replication_job_3` |

### Datasink Cluster Names

**Pattern**: `datasink_{cluster_tier}_{jobtype_prefix}_jobtype.{env}_{job_name}_{consumer_type}`

The EMR cluster is determined by the job's `cluster_tier` prefix (not the full job type):

| Job Type Pattern | EMR Cluster Name Prefix | Example Full Cluster Name |
|---|---|---|
| `shared_jobtype.admin_api`, `de_reporting_api`, `merchant_api` | `datasink_api` | `datasink_api_datasink_shared_jobtype.admin_api_mysql-consumer` |
| `shared_jobtype.prod-white-mum_api`, `prod-white-mum_combined` thirdeye | `datasink_prod_white_medium` | `datasink_prod_white_medium_datasink_shared_jobtype.prod-white-mum_api_mysql-consumer` |
| `shared_jobtype.prod-white-mum_combined`, `prod-white-mum_thirdeye` | `datasink_prod_white_combined` | `datasink_prod_white_combined_datasink_shared_jobtype.prod-white-mum_combined_mysql-consumer` |
| `payments_jobtype.*_card_rds`, `shared_jobtype.*_card_rds` | `datasink_payments_card` | `datasink_payments_card_datasink_payments_jobtype.admin_payments_card_rds_mysql-consumer` |
| `payments_jobtype.*_ups`, `shared_jobtype.*_tidb_backfill` | `datasink_payments_upi` | `datasink_payments_upi_datasink_shared_jobtype.prod-white-mum_payments_ups_mysql-consumer` |
| `rx_jobtype.*` | `datasink_rx` | `datasink_common_datasink_rx_jobtype.admin_rx_payouts_mysql-consumer` |
| `settlements_jobtype.settlements_warm_db` | `datasink_settlements_warm_db` | `datasink_settlements_warm_db_datasink_settlements_jobtype.settlements_warm_db_postgres-consumer` |
| All others (default) | `datasink_common` | `datasink_common_datasink_payments_jobtype.admin_pg_router_mysql-consumer` |

### Harvester Cluster Names

**Pattern**: `harvester_{volume_tier}_job_harvester_v2_{domain}_stream_{job_name}`

| Volume Tier | EMR Cluster Name Prefix | Example Full Cluster Name |
|---|---|---|
| `low_volume` | `harvester_low_volume_job` | `harvester_low_volume_job_harvester_v2_payments_stream_orders_v2` |
| `high_volume` | `harvester_high_volume_job` | `harvester_high_volume_job_harvester_v2_payments_stream_payments_v4` |
| batch/S3 | `harvester_batch_job` | `harvester_batch_job_harvester_v2_settlements_stream_settlements_sla_v2` |

Multiple clusters share the same prefix (one cluster per index group). Query all matching clusters, then identify the alerting node by IP.

## Datasink EMR Cluster Names

Source: `datahub/datasink/dockerconf/entrypoint.sh`

| Job Type | EMR Cluster Name |
|---|---|
| Default (all others) | `datasink_common` |
| `shared_jobtype.{merchant_api,de_reporting_api,admin_api}` | `datasink_api` |
| `shared_jobtype.prod-white-mum_api` | `datasink_prod_white_medium` |
| `shared_jobtype.prod-white-mum_combined`, `prod-white-mum_thirdeye` | `datasink_prod_white_combined` |
| `payments_jobtype.*_card_rds`, `shared_jobtype.*_card_rds` | `datasink_payments_card` |
| `payments_jobtype.*_ups`, `shared_jobtype.*_tidb_backfill` | `datasink_payments_upi` |
| `settlements_jobtype.settlements_warm_db` | `datasink_settlements_warm_db` |
| `payments_jobtype.p5_harvester_replica_aurora_v2` | `datasink_harvester_rearch` |

## Harvester EMR Cluster Names

Source: `shakshuka/harvesterPipeline/dockerconf/entrypoint.sh`

| Alert `application_name` | Condition | EMR Cluster Name Prefix |
|---|---|---|
| `harvester-low-volume` | Default | `harvester_low_volume_job` |
| `harvester-high-volume` / `live_low_vol_indx` | `APP_NAME=live_low_vol_indx` | `harvester_high_volume_job` |
| (batch/S3 jobs) | `SINK_IDENTIFIER=s3` | `harvester_batch_job` |


## Entity-Operator: EMR Cluster Name Mapping

### Staging-Table-Pipeline Clusters

| BU | EMR Cluster Name | App Name |
|---|---|---|
| payment | `entity_operator_staging_payment_graviton` | `entityoperator_payment_staging` |
| shared | `entity_operator_staging_common_graviton_ticdc` | `entityoperator_shared_staging` |
| rx | `entity_operator_staging_common_graviton` | `entityoperator_rx_staging` |
| platform | `entity_operator_staging_common_graviton` | `entityoperator_platform_staging` |
| capital | `entity_operator_staging_common_graviton` | `entityoperator_capital_staging` |
| evolvehq | `entity_operator_staging_common_graviton` | `entityoperator_evolvehq_staging` |
| data | `entity_operator_staging_common_graviton` | `entityoperator_data_staging` |

### Replication-Table-Pipeline Clusters

| BU | EMR Cluster Name | App Name |
|---|---|---|
| payment | `entity_operator_replication_mid_large_plus_graviton` | `entityoperator_payment_replication_job_{N}` |
| platform | `entity_operator_replication_mid_large_graviton` | `entityoperator_platform_replication_job_{N}` |
| rx | `entity_operator_replication_medium_graviton` | `entityoperator_rx_replication_job_{N}` |
| evolvehq | `entity_operator_replication_small_plus_graviton` | `entityoperator_evolvehq_replication_job_{N}` |
| capital | `entity_operator_replication_small_graviton` | `entityoperator_capital_replication_job_{N}` |
| data | `entity_operator_replication_small_graviton` | `entityoperator_data_replication_job_{N}` |
| shared (job 1) | `entity_operator_replication_small_graviton_ticdc` | `entityoperator_shared_replication_job_1` |
| shared (job 2) | `entity_operator_replication_medium_graviton_ticdc` | `entityoperator_shared_replication_job_2` |
| shared (job 3+) | `entity_operator_replication_large_graviton_ticdc` | `entityoperator_shared_replication_job_3` |

## Entity-Operator: DLQ (Error) Topics

| Pipeline | BU | DLQ Topic |
|---|---|---|
| Staging | payment | `staging_cdc_payment_error_topic` |
| Staging | platform | `staging_cdc_platform_error_topic` |
| Staging | rx | `staging_cdc_rx_error_topic` |
| Staging | capital | `staging_cdc_capital_error_topic` |
| Staging | evolvehq | `staging_cdc_evolvehq_error_topic` |
| Staging | shared | `staging_cdc_shared_error_topic` |
| Staging | data | `staging_cdc_data_error_topic` |
| Replication | payment | `replication_pipeline_payment_error_topic` |
| Replication | platform | `replication_pipeline_platform_error_topic` |
| Replication | rx | `replication_pipeline_rx_error_topic` |
| Replication | capital | `replication_pipeline_capital_error_topic` |
| Replication | evolvehq | `replication_pipeline_evolvehq_error_topic` |
| Replication | shared | `replication_pipeline_shared_error_topic` |

## Entity-Operator: Upstream Dependency Mapping

The staging-table-pipeline consumes from the **same Kafka CDC topics** as Datasink. Use the Maxwell/Kafka-Connect mapping tables above to identify which CDC producer feeds a given database.

```
Staging upstream (non-shared BUs):  Maxwell (MySQL) / Kafka-Connect (PostgreSQL) → Kafka Topics → staging-table-pipeline
Staging upstream (shared BU):       TiCDC (see below) → ticdc-rzp-payment-api → staging-table-pipeline (ticdc cluster)
Replication upstream:               staging-table-pipeline → Delta Staging Tables (S3) → replication-table-pipeline
```

When debugging replication lag, trace backwards: replication → staging → CDC producer → source DB.

## TiCDC Pipeline (Shared BU Only)

**What it is**: Datasink writes API/token data to TiDB cluster `rzp-payment-api`. TiCDC (TiDB's native CDC mechanism) captures those changes and produces to a dedicated Kafka topic. The shared BU entity-operator staging and replication pipelines consume exclusively from this topic.

**This path is ONLY for the `shared` BU.** All other BUs use Maxwell/Kafka-Connect as their staging upstream.

| Component | Value |
|---|---|
| TiDB Source Cluster | `rzp-payment-api` |
| CDC Mechanism | TiCDC (native TiDB CDC) |
| Kafka Topic | `ticdc-rzp-payment-api` |
| Staging Cluster | `entity_operator_staging_common_graviton_ticdc` |
| Replication Cluster (job 1) | `entity_operator_replication_small_graviton_ticdc` |
| Replication Cluster (job 2) | `entity_operator_replication_medium_graviton_ticdc` |
| Replication Cluster (job 3+) | `entity_operator_replication_large_graviton_ticdc` |
| Staging Error Topic | `staging_cdc_shared_error_topic` |
| Replication Error Topic | `replication_pipeline_shared_error_topic` |

**Full cascade for shared BU lag:**
```
Datasink job (api/token data) → TiDB rzp-payment-api
                                       │
                                   TiCDC (binlog CDC)
                                       │
                               Kafka: ticdc-rzp-payment-api
                                       │
              entity_operator_staging_common_graviton_ticdc
                              (shared BU staging)
                                       │
              entity_operator_replication_*_graviton_ticdc
                            (shared BU replication)
                                       │
                                Trino / Presto
```

**Diagnostic metrics for TiCDC path:**
- `kafka_server_brokertopicmetrics_messagesin_total{topic="ticdc-rzp-payment-api"}` — production rate from TiCDC; if near zero, TiCDC has stalled
- `kafka_consumergroup_lag{consumergroup=~"entity-operator-shared.*"}` — consumer lag for shared BU staging
- `stc_v2_duration_ms{bu="shared"}` — staging batch liveness
- If TiCDC not producing → check TiDB `rzp-payment-api` health + Datasink job writing to it

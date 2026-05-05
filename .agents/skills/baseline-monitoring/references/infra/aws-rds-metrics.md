# AWS RDS Metrics

Read this file when the service depends on RDS and database health is part of the standard observability picture.

## Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Database connections | `aws_rds_database_connections` | Number of active database connections |
| CPU utilization | `aws_rds_cpu_utilization` | RDS CPU usage |
| Freeable memory | `aws_rds_freeable_memory` | Memory available to the instance |
| Free storage space | `aws_rds_free_storage_space` | Remaining storage capacity |
| Read IOPS | `aws_rds_read_iops` | Read operation rate |
| Write IOPS | `aws_rds_write_iops` | Write operation rate |
| Read latency | `aws_rds_read_latency` | Read response time |
| Write latency | `aws_rds_write_latency` | Write response time |
| Network receive throughput | `aws_rds_network_receive_throughput` | Inbound network throughput |
| Network transmit throughput | `aws_rds_network_transmit_throughput` | Outbound network throughput |
| Disk queue depth | `aws_rds_disk_queue_depth` | Disk queue backlog |
| Burst balance | `aws_rds_burst_balance` | Burst credit or burst-performance balance |
| Replica lag | `aws_rds_replica_lag` | Replication delay |
| Failed SQL Server Agent jobs | `aws_rds_failed_sql_server_agent_jobs` | Failed SQL Server Agent jobs for SQL Server deployments only |
| Database load | `aws_rds_database_load` | Overall load on the database |
| Login failures | `aws_rds_login_failures` | Authentication failure visibility |
| Queries | `aws_rds_queries` | Query volume |
| Read throughput | `aws_rds_read_throughput` | Read data rate |
| Write throughput | `aws_rds_write_throughput` | Write data rate |
| Transaction logs generation | `aws_rds_transaction_logs_generation` | Transaction log generation rate |

## Engine Note

- some RDS metrics are engine-specific; `aws_rds_failed_sql_server_agent_jobs` applies only to SQL Server-backed deployments and should be ignored for MySQL, Aurora MySQL, PostgreSQL, or other engines

## Baseline Expectation

When RDS is part of the service path, connection health, compute and storage capacity, latency, throughput, and replication state should be visible in the standard infra view.

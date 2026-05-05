# AWS ElastiCache Metrics

Read this file when the service depends on ElastiCache or Redis-like cache infrastructure and those signals are part of the standard coverage.

## Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Cache hit rate | `aws_elasticache_cache_hits` | Cache hit performance |
| Memory usage | `aws_elasticache_bytes_used_for_cache` | Cache memory consumption |
| Network I/O | `aws_elasticache_network_bytes_in` | Cache network traffic |
| Connections | `aws_elasticache_curr_connections` | Active cache connections |
| Evictions | `aws_elasticache_evictions` | Evictions due to memory pressure |
| CPU usage | `aws_elasticache_cpu_utilization` | CPU utilization |
| Swap usage | `aws_elasticache_swap_usage` | Memory swap usage |
| Replication lag | `aws_elasticache_replication_lag` | Replication delay |
| Engine CPU utilization | `aws_elasticache_engine_cpu_utilization` | Engine-specific CPU usage |

## Baseline Expectation

When ElastiCache is part of the service path, cache efficiency, memory pressure, connection health, compute usage, and replication state should be visible in the standard infra view.

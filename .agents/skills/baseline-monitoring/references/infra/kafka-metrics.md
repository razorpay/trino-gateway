# Kafka Metrics

Read this file when the service owns Kafka producers, consumers, or topics whose health must be covered.

## Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Byte throughput (in) | `kafka_server_brokertopicmetrics_bytesin_total` | Traffic volume into Kafka topics |
| Byte throughput (out) | `kafka_server_brokertopicmetrics_bytesout_total` | Traffic volume out of Kafka topics |
| Message rate | `kafka_server_brokertopicmetrics_messagesin_total` | Incoming message rate |
| Log size | `kafka_log_log_size` | Actual disk space consumed by the topic log, in bytes |
| Consumer lag | `kafka_consumergroup_lag` | Lag by consumer group and topic |
| Partition offset lag | `kafka_cluster_partition_laststableoffsetlag` | Per-partition stable offset lag; tracks processing backlog at partition granularity |

# Business-Critical Metrics Examples

This reference provides examples of metrics that should exist for business-critical flows, with **real production examples from Razorpay services**.

## Real Metrics from Terminals Service

### HTTP Metrics (terminals namespace)
- `terminals_http_requests_total{code, route, method, mode, env}` - Total HTTP requests
- `terminals_http_durations_ms_histogram_bucket{route, method, le}` - Request latency histogram

### Business Flow Metrics
- `terminals_no_terminals_found{route, mode, env}` - Terminal routing failures (NTF events)
- `terminals_merchant_instrument_request_created{mode, env}` - MIR creation tracking
- `terminals_terminal_gateway_validation_events{status, operation, gateway}` - Gateway validation results
- `terminals_hitachi_index_used{index_key}` - Hitachi index usage (resource exhaustion tracking)

### Cache Metrics
- `terminals_cache_hit_and_miss{event, mode, env}` - Cache hit/miss tracking (event="cache_hit" or "cache_miss")

### Worker/Kafka Metrics
- `terminals_kafka_consumer_message_consumption_total{topic}` - Messages consumed
- `terminals_kafka_consumer_message_processed_failure_total{topic}` - Processing failures
- `terminals_kafka_consumer_message_processing_duration{topic}` - Processing latency

### Worker-Specific Metrics
- `terminals_webscraper_latency_sec_avg{mode, env}` - Webscraper average latency
- `terminals_webscraper_latency_sec_max{mode, env}` - Webscraper max latency
- `terminals_webscraper_throughput{mode, env}` - Webscraper throughput
- `terminals_webscraper_failure_count{mode, env}` - Webscraper failures

## HTTP Request Metrics

### Essential
- `service_http_requests_total{code, route, method, mode}` - Total request count
- `service_http_durations_ms_histogram_bucket{route, method, le}` - Request latency distribution

**Business need:** Track API availability, performance SLAs, identify failing endpoints

## Error Metrics

### Application Errors
- `service_rzp_errors{error, mode, env}` - Application error counts by error type
- `service_error_response_metric{internalErrorCode, mode, env}` - Internal error code tracking

**Business need:** Identify error patterns, track error budgets, debug production issues

## Business Flow Metrics

### Payment Processing
- `service_payment_created{status, method}` - Payment creation tracking
- `service_payment_status_change{from_status, to_status}` - Payment state transitions
- `service_no_terminals_found{route, paymentMethod}` - Terminal routing failures (NTF)

**Business need:** Monitor payment success rates, identify routing issues, track conversion funnels

### Order Processing
- `service_order_created{status}` - Order creation tracking
- `service_order_paid{method}` - Successful order completions

**Business need:** Track order funnel, identify drop-off points, measure business KPIs

## External Service Metrics

**⚠️ DO NOT ADD - External services handle their own alerts**

External service calls (API, Mozart, Reminder, etc.) should be monitored by those services, not by the calling service. Do not add metrics or alerts for:
- Downstream API latency
- External service timeouts
- Third-party integration failures

**Reason:** Alert ownership should follow service ownership. Each service is responsible for monitoring its own performance.

**Exception:** Only track client-side errors that indicate issues with YOUR service's integration code (e.g., malformed requests, missing auth tokens).

## Resource Metrics

### Database
- `service_db_query_duration{operation, table}` - Database query performance
- `service_db_connection_pool{state}` - Connection pool utilization

**Business need:** Identify slow queries, prevent connection exhaustion, capacity planning

### Cache
- `service_cache_hit_and_miss{event, route}` - Cache hit/miss ratio
- `service_cache_latency{operation}` - Cache operation latency

**Business need:** Optimize cache effectiveness, identify cache performance issues

## Worker/Kafka Metrics

### Consumer Health
- `service_kafka_consumer_message_consumption_total{topic}` - Messages consumed
- `service_kafka_consumer_message_processed_failure_total{topic}` - Processing failures
- `service_kafka_consumer_message_processing_duration{topic}` - Processing latency

**Business need:** Ensure async processing health, identify consumer lag, track throughput

## Panic/Crash Metrics

- `service_panic{path, mode}` - Application panic count

**Business need:** Detect application crashes, identify unstable code paths

## Common Metric Naming Patterns

### Counters (always increasing)
- `<service>_<entity>_<action>_total` (e.g., `router_requests_total`)
- `<service>_<entity>_count` (e.g., `terminals_terminal_created_count`)

### Histograms (for latency/duration)
- `<service>_<operation>_duration_ms_histogram_bucket`
- `<service>_<operation>_latency_bucket`

### Gauges (current value)
- `<service>_<resource>_<metric>` (e.g., `service_db_connections_active`)

## ⚠️ CRITICAL: High Cardinality Label Rules

### NEVER Use These as Labels (Memory Exhaustion Risk)

**Unbounded identifiers - FORBIDDEN:**
- ❌ `merchant_id` - Millions of merchants
- ❌ `terminal_id` - Millions of terminals
- ❌ `user_id` - Millions of users
- ❌ `payment_id` - Millions of payments
- ❌ `order_id` - Millions of orders
- ❌ `transaction_id` - Unbounded
- ❌ `request_id` - Unbounded
- ❌ `session_id` - Unbounded
- ❌ `device_id` - Unbounded
- ❌ `ip_address` - Very high cardinality
- ❌ Any unique identifier with >100 possible values

**Why forbidden:** Each unique label value creates a new time series in Prometheus. With millions of IDs, this causes:
- **Memory exhaustion** - OOM kills Prometheus
- **Query timeout** - Aggregations become impossibly slow
- **Storage explosion** - Disk space consumed rapidly
- **Alert failures** - Prometheus cannot evaluate rules

### ALWAYS Use These (Low Cardinality - Safe)

**Bounded categorical values - SAFE:**
- ✅ `status` - Limited values (success, failure, pending, timeout)
- ✅ `method` - HTTP methods (GET, POST, PUT, DELETE, PATCH)
- ✅ `route` - API endpoints (~10-50 routes per service)
- ✅ `code` - HTTP status codes (200, 400, 404, 500, etc.)
- ✅ `gateway` - Payment gateways (~5-20 gateways)
- ✅ `event_type` - Event categories (~10-30 types)
- ✅ `operation` - CRUD operations (create, read, update, delete)
- ✅ `cache_type` - Cache backends (redis, memory, memcached)
- ✅ `env` - Environments (prod, stage, dev)
- ✅ `mode` - Operating modes (live, test)
- ✅ `table` - Database tables (~10-50 tables)
- ✅ `topic` - Kafka topics (~10-50 topics)

**Rule of thumb:** Label cardinality should be **< 100 unique values**

### Examples

**❌ BAD - High Cardinality:**
```go
// WRONG - merchant_id is unbounded
paymentCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{Name: "payments_total"},
    []string{"merchant_id", "status"},  // ❌ Millions of merchants!
)
```

**✅ GOOD - Low Cardinality:**
```go
// CORRECT - Only status is used
paymentCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{Name: "payments_total"},
    []string{"status", "gateway"},  // ✅ Few values each
)
```

**❌ BAD - Terminal-specific tracking:**
```go
// WRONG - terminal_id is unbounded
terminalLatency := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{Name: "terminal_latency_ms"},
    []string{"terminal_id", "operation"},  // ❌ Millions of terminals!
)
```

**✅ GOOD - Aggregate tracking:**
```go
// CORRECT - No terminal_id, use operation + status
terminalLatency := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{Name: "terminal_latency_ms"},
    []string{"operation", "status", "gateway"},  // ✅ Low cardinality
)
```

### What If I Need Per-Entity Metrics?

**Use logs, not metrics:**
- For debugging specific merchant issues → Use application logs with merchant_id
- For tracking specific payment failures → Use structured logs with payment_id
- For audit trails → Use database records or event streams

**Metrics are for aggregates, logs are for details.**

# Common Alert Patterns for Razorpay Services

This reference documents common alert patterns observed in Razorpay's alert-rules repository, with **real examples from production services**.

## Table of Contents
1. [HTTP Error Alerts](#http-error-alerts)
2. [Latency Alerts](#latency-alerts)
3. [NTF Alerts](#ntf-alerts-no-terminal-found)
4. [Resource Alerts](#resource-alerts)
5. [Kafka/Worker Alerts](#kafkaworker-alerts)
6. [Database Alerts](#database-alerts)
7. [Business-Specific Alerts](#business-specific-alerts)
8. [Time-Based Conditional Alerts](#time-based-conditional-alerts)

## HTTP Error Alerts

### 5xx Errors - Route Specific

**Real example from terminals service:**
```yaml
- alert: "Terminals Service - 5xx (rpm) - Critical Fetch Terminal routes"
  expr: >
    sum by (route, method) (
      increase(terminals_http_requests_total{
        code=~"5..*",
        env="prod",
        mode="live",
        route=~"/v1/merchants/terminals|/v2/terminals/credentials/:id|/v1/terminals/:id",
        k8s_pod!~".*dark.*"
      }[5m])
    ) > 0
  for: 2m
  labels:
    severity: critical
    bu: Payments
    pod: payments-onboarding
    service: terminals
    live: true
    slack_channel: "#p0_alert_rules_terminals_service"
  annotations:
    identifier: terminals_5xx_critical_routes
    description: onlinepayments_onboarding Terminals 5xx on critical routes {{ $value }}
    Runbook: https://docs.google.com/document/d/...
    vajra_link: https://vajra.razorpay.com/d/...
```

**Key patterns:**
- **Route-specific monitoring:** Alert on critical routes only (customer-facing APIs)
- **Exclude dark traffic:** `k8s_pod!~".*dark.*"` prevents canary/test alerts
- **Threshold = 0:** Any 5xx on critical routes fires immediately
- **Common thresholds:** 0-50 errors per 5 minutes depending on criticality

### 4xx Errors

**Real example from terminals service:**
```yaml
- alert: "Terminals Service - 4xx (rpm)"
  expr: >
    sum by (route, method, code) (
      increase(terminals_http_requests_total{
        code=~"4..*",
        env="prod",
        mode="live",
        route!~"/status"
      }[5m])
    ) > 100
  for: 2m
```

**Common thresholds:** 20-100 errors per 5 minutes (higher tolerance than 5xx)

## Latency Alerts

### Multi-Percentile Latency Ladder

**Real example from terminals service - Multiple percentiles for same endpoint:**

```yaml
# p99 alert
- alert: "Response Time For Fetch Terminal by ID [p99]"
  expr: >
    (
      avg(
        histogram_quantile(0.99,
          sum(rate(terminals_http_durations_ms_histogram_bucket{
            route="/v1/terminals/:id",
            method="GET"
          }[1m])) by (le)
        )
      ) > 40
    )
  for: 2m
  labels:
    severity: critical
    bu: Payments
    pod: payments-onboarding
    service: terminals
    live: true
    slack_channel: "#p0_alert_rules_terminals_service"

# p95 alert
- alert: "Response Time For Fetch Terminal by ID [p95]"
  expr: >
    histogram_quantile(0.95,
      sum(rate(terminals_http_durations_ms_histogram_bucket{
        route="/v1/terminals/:id",
        method="GET"
      }[1m])) by (le)
    ) > 20
  for: 2m

# p90 alert
- alert: "Response Time For Fetch Terminal by ID [p90]"
  expr: >
    histogram_quantile(0.90, ...) > 20
  for: 2m

# p50 alert
- alert: "Response Time For Fetch Terminal by ID [p50]"
  expr: >
    histogram_quantile(0.50, ...) > 15
  for: 2m
```

**Pattern: Progressive SLA ladder**
- **p99 > 40ms** - Catches worst-case tail latency
- **p95 > 20ms** - Median degradation
- **p90 > 20ms** - Broad performance issue
- **p50 > 15ms** - Baseline performance

**Real thresholds from terminals:**
- `/v1/terminals/:id` GET: p99=40ms, p95=20ms, p90=20ms, p50=15ms
- `/v1/merchants/terminals` POST: p99=70ms, p95=40ms, p90=30ms, p50=25ms
- `/v2/terminals/credentials/:id` GET: p99=40ms, p95=20ms

## NTF Alerts (No Terminal Found)

**Real example from terminals service:**

```yaml
- alert: "Terminals Service NTF count greater than threshold"
  expr: >
    sum(rate(terminals_no_terminals_found{
      route="/v1/internal/pg-router/terminals",
      mode="live"
    }[1m])) > 8500
  for: 5m
  labels:
    severity: critical
    bu: Payments
    pod: payments-onboarding
    service: terminals
    live: true
    slack_channel: "#p0_alert_rules_terminals_service"
  annotations:
    identifier: terminal_ntf_count_greater_than_8500
    description: onlinepayments_onboarding NTF count greater than threshold {{ $value }}
    Runbook: https://docs.google.com/document/d/...
    vajra_link: https://vajra.razorpay.com/d/...
```

**Real thresholds from production:**
- **Terminals service:** > 8500 NTF/min (high traffic service)
- **Router service:** > 20 NTF/2min for general, payment-method specific thresholds:
  - Cards: > 20 NTF/2min
  - Emandate: > 10 NTF/2min
  - Nach: > 10 NTF/2min
  - Wallet: > 5 NTF/2min
  - EMI: > 10 NTF/2min

**Business impact:** NTF = payment cannot be routed = direct revenue loss

## Resource Alerts

### Pod Count
```yaml
- alert: "Low Pod Count"
  expr: sum(kube_deployment_status_replicas_available{deployment="service", namespace="service"}) < <min_pods>
  for: 1m
```

### CPU Usage
```yaml
- alert: "High CPU Usage"
  expr: avg(sum by (pod)(rate(container_cpu_usage_seconds_total{namespace="service", pod=~"service.*"}[2m])))*100 > <threshold>
  for: 5m
```

### Memory Usage
```yaml
- alert: "High Memory Usage"
  expr: avg(sum by(pod)(container_memory_usage_bytes{namespace="service", pod=~"service.*"})) > <threshold>
  for: 5m
```

## Kafka/Worker Alerts

### Consumer Lag
```yaml
- alert: "Kafka Consumer Lag High"
  expr: sum(kafka_consumer_lag{topic="<topic>"}) by (consumer_group) > <threshold>
  for: 5m
```

### Processing Failures
```yaml
- alert: "Worker Processing Failures"
  expr: sum(increase(service_kafka_consumer_message_processed_failure_total{topic="<topic>"}[5m])) by (topic) > <threshold>
  for: 1m
```

## Database Alerts

**Real examples from terminals service:**

### High Connections
```yaml
- alert: "Terminals DB high connections - prod-aurora-postgres-terminals-1a"
  expr: >
    aws_rds_database_connections_maximum{
      dbinstance_identifier="prod-aurora-postgres-terminals-1a"
    } > 600
  for: 1m
  labels:
    severity: critical
    bu: Payments
    pod: payments-onboarding
    service: terminals
    live: true
    slack_channel: "#p0_alert_rules_terminals_service"
```

### High CPU
```yaml
- alert: "Terminals DB high CPU - prod-aurora-postgres-terminals-1a"
  expr: >
    aws_rds_cpuutilization_maximum{
      dbinstance_identifier="prod-aurora-postgres-terminals-1a"
    } > 50
  for: 30s
```

**Real thresholds:**
- **Connections:** > 600 (terminals), monitor both 1a and 1b instances
- **CPU:** > 50% for 30s

## Business-Specific Alerts

These alerts track business logic and operational metrics unique to each service.

### Resource Exhaustion

**Real example from terminals - Hitachi Index Exhaustion:**
```yaml
- alert: "Hitachi Index Usage 45R"
  expr: >
    max(terminals_hitachi_index_used{
      index_key="hitachi_gateway_terminal_creation_index_45r"
    }) by (index_key) > 99000
  for: 2m
  labels:
    severity: critical
    bu: Payments
    pod: payments-onboarding
    service: terminals
    live: true
    slack_channel: "#p0_alert_rules_terminals_service"
  annotations:
    identifier: terminal_hitachi_index_usage_45r_greater_than_99000
    description: Hitachi Index Usage exceeds 99000 (limit 100000)
```

**Business impact:** Index exhaustion prevents new terminal creation = merchant onboarding blocked

### Cache Performance

**Real example from terminals - Cache Miss Rate:**
```yaml
- alert: "Cache Miss Rate greater than 8%"
  expr: >
    (100 *
      sum(increase(terminals_cache_hit_and_miss{mode="live", env="prod", event="cache_miss"}[1m]))
      /
      sum(increase(terminals_cache_hit_and_miss{mode="live", env="prod"}[1m]))
    ) > 8
  for: 1m
```

**Business impact:** High cache miss = increased DB load + slower API response

### Business Flow Metrics

**Real example from terminals - MIR Creation Rate:**
```yaml
- alert: "MIR Per Day Creation Rate less than 100"
  expr: >
    sum(increase(terminals_merchant_instrument_request_created{mode="live", env="prod"}[24h])) < 100
  for: 30m
```

**Business impact:** Low MIR creation rate = potential onboarding pipeline issue

### Worker Performance

**Real example from terminals - Webscraper Latency:**
```yaml
- alert: "Webscraper Latency greater than 60 seconds (Avg)"
  expr: >
    avg(terminals_webscraper_latency_sec_avg{mode="live", env="prod"}) > 60
  for: 5m

- alert: "Max Webscraper PDF Generation time greater than 15 mins"
  expr: >
    max(terminals_webscraper_latency_sec_max{mode="live", env="prod"}) > 900
  for: 5m
  labels:
    live: false  # Non-critical, won't trigger PagerDuty
```

## Time-Based Conditional Alerts

Alerts with different thresholds for day vs night or business hours.

### Day vs Night Pod Count

**Real example from terminals:**

```yaml
# Daytime alert - Need more pods
- alert: "Terminals Live Available Pods - Daytime"
  expr: >
    (
      (sum(kube_deployment_status_replicas_available{
        job=~"eks-cde-green-kube-state-metrics|eks-cde-blue-kube-state-metrics",
        namespace=~".*terminals.*",
        deployment=~"terminals-live.*",
        deployment!~"terminals-live-baseline.*|terminals-live-canary.*"
      }) < 12)
      and
      (hour()*60 + minute() >= 210 and hour()*60 + minute() <= 930)
    )
  for: 1m

# Nighttime alert - Fewer pods needed
- alert: "Terminals Live Available Pods - Night time"
  expr: >
    (
      (sum(kube_deployment_status_replicas_available{...}) < 6)
      and
      (hour()*60 + minute() <= 210 or hour()*60 + minute() >= 930)
    )
  for: 1m
```

**Pattern:** `hour()*60 + minute()` converts time to minutes since midnight
- **210 minutes** = 3:30 AM
- **930 minutes** = 3:30 PM (15:30)

### Business Hours Traffic

**Real example from terminals:**

```yaml
- alert: "Terminals Fetch by ID Request Less Than 500"
  expr: >
    (
      (avg(sum(rate(terminals_http_requests_total{
        route="/v1/terminals/:id",
        method="GET"
      }[2m]))) < 500)
      and
      (hour()*60 + minute() >= 210 and hour()*60 + minute() <= 930)
    )
  for: 2m
```

**Pattern:** Only alert during business hours (3:30 AM - 3:30 PM IST) when traffic should be high

## Label Structure

All alerts should include:
```yaml
labels:
  severity: critical
  bu: <business-unit>
  pod: <pod-name>
  service: <service-name>
  live: true
  slack_channel: "<channel>"
```

## Annotation Structure

All alerts should include:
```yaml
annotations:
  identifier: <unique_identifier>
  description: <description with {{$value}}>
  Runbook: <google-doc-url>
  vajra_link: <vajra-dashboard-url>
```

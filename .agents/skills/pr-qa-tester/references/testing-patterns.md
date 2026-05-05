# Testing Patterns Reference

## API Testing

### REST API Validation

```bash
# Basic API call from within the cluster
kubectl exec -n <namespace> <pod> -- curl -s http://<service>:<port>/endpoint

# With headers and body
kubectl exec -n <namespace> <pod> -- curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Merchant-Id: <merchant_id>" \
  -d '{"key": "value"}' \
  http://<service>:<port>/endpoint

# Via ingress (external)
curl -s https://<service>-<devstack_label>.dev.razorpay.in/endpoint
```

### gRPC / Twirp Validation

```bash
# Twirp endpoint (JSON over HTTP)
kubectl exec -n <namespace> <pod> -- curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"field": "value"}' \
  http://<service>:<port>/twirp/<package>.<Service>/<Method>
```

## State Verification

### Database Queries

```bash
# MySQL
kubectl exec -n <namespace> <mysql-pod> -- mysql -u root -p<password> \
  -e "SELECT * FROM <table> WHERE id='<id>' LIMIT 5;" <database>

# PostgreSQL
kubectl exec -n <namespace> <pg-pod> -- psql -U <user> -d <database> \
  -c "SELECT * FROM <table> WHERE id='<id>' LIMIT 5;"
```

### Redis / Cache

```bash
kubectl exec -n <namespace> <redis-pod> -- redis-cli GET <key>
kubectl exec -n <namespace> <redis-pod> -- redis-cli HGETALL <key>
```

### SQS Queue Inspection

```bash
# Check queue depth
kubectl exec -n <namespace> <pod> -- aws sqs get-queue-attributes \
  --queue-url <queue-url> --attribute-names ApproximateNumberOfMessages
```

## Log Analysis

### Structured Log Search

```bash
# Get recent logs
kubectl logs -n <namespace> -l name=<service>-<label> --tail=100

# Search for specific patterns
kubectl logs -n <namespace> -l name=<service>-<label> --tail=500 | grep -i "error\|exception\|panic"

# Follow logs during test execution
kubectl logs -n <namespace> -l name=<service>-<label> -f
```

### Coralogix (Production-grade Logs)

Use the `rzp-discover:coralogix` agent for searching structured logs when kubectl logs are insufficient.

## Service-Type Patterns

### Java / Spring Boot

- Health check: `GET /actuator/health`
- Metrics: `GET /actuator/metrics`
- Config verification: Check `application.properties` or env vars via `kubectl exec -- env | grep <KEY>`
- Spring Batch jobs: Trigger via API, verify `BATCH_JOB_EXECUTION` table

### Go Services

- Health check: `GET /health` or `GET /healthz`
- Debug: `GET /debug/pprof/` (if enabled)
- Config: Environment variables via `kubectl exec -- env`

### PHP / API Service

- Health check: `GET /health`
- Config: Check `config/*.php` or `env` command
- Route testing: Use the internal route names

## Experiment / Feature Flag Validation

### Splitz Experiments

```bash
# Check experiment status
# Use rzp-discover Splitz MCP or API directly
curl -s https://splitz-<label>.dev.razorpay.in/v1/experiment/<experiment_id>/evaluate \
  -H "Content-Type: application/json" \
  -d '{"entity_id": "<merchant_id>"}'
```

### Test with Experiment Enabled/Disabled

1. Enable experiment for specific merchant via Splitz API
2. Run test flow → verify bypass/new behavior
3. Disable experiment for same merchant
4. Run test flow → verify default behavior
5. Compare results

## Integration Testing Patterns

### Service-to-Service Communication

1. Identify the call chain from PR analysis (e.g., batch → splitz → scrooge)
2. Verify each service is deployed and healthy
3. Trigger the entry point
4. Verify downstream effects via logs + state

### Idempotency Testing

1. Execute the same request twice with identical parameters
2. Verify only one state change occurred
3. Verify second response indicates idempotent handling

### Error Handling

1. Send malformed requests → verify 4xx responses
2. Simulate downstream failures (if possible) → verify graceful degradation
3. Check error logs for proper error classification

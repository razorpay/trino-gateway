# Phase 6: Test Execution

Execute the approved test plan and collect results.

## Execution Protocol

For each test case:

1. **Log the test**: Record test name, category, and timestamp
2. **Execute**: Run the API call / trigger the flow
3. **Capture response**: Save HTTP status, response body, headers
4. **Verify**: Check against expected outcome
5. **Verify state**: Run state verification checks (DB queries, logs)
6. **Record result**: Pass/fail with details

## Execution Methods

### Direct API Calls

```bash
# From outside cluster (via ingress)
curl -s -w "\nHTTP_STATUS:%{http_code}" \
  -X POST https://<service>-<label>.dev.razorpay.in/endpoint \
  -H "Content-Type: application/json" \
  -d '<payload>'

# From inside cluster (service-to-service)
kubectl exec -n <namespace> <pod> -- curl -s \
  http://<target-service>:<port>/endpoint \
  -H "Content-Type: application/json" \
  -d '<payload>'
```

### Batch Job Triggers

```bash
# For Spring Batch services (like batch service)
curl -s -X POST https://<service>-<label>.dev.razorpay.in/batch/trigger \
  -H "Content-Type: application/json" \
  -d '{"batch_type": "<type>", "file_id": "<id>"}'
```

### Queue Message Publishing

```bash
kubectl exec -n <namespace> <pod> -- aws sqs send-message \
  --queue-url <url> \
  --message-body '<json>'
```

## Log Monitoring During Tests

Stream logs while executing tests to catch real-time errors:

```bash
# In a separate terminal / background
kubectl logs -n <namespace> -l name=<service>-<label> -f --tail=0 &
LOG_PID=$!

# Run test...
# After test, stop log streaming
kill $LOG_PID
```

Or use `kubectl logs --since=1m` after test execution to capture relevant logs.

## Result Collection

Track results in a structured format for the report generator:

```json
{
  "test_cases": [
    {
      "name": "Happy path refund bypass with experiment enabled",
      "type": "happy_path",
      "status": "pass",
      "details": "Refund created via Scrooge bypass, HTTP 200"
    },
    {
      "name": "Refund without experiment falls back to default",
      "type": "integration",
      "status": "pass",
      "details": "Default refund.json config used, processed via API proxy"
    }
  ],
  "failures": [],
  "fixes_applied": [],
  "entity_changes": [
    "Refund rfnd_xxx created for payment pay_xxx"
  ]
}
```

## Failure Handling

When a test fails:

1. **Capture the failure**: HTTP status, response body, error message
2. **Check logs**: `kubectl logs` for the relevant service
3. **Check events**: `kubectl get events -n <namespace> --sort-by='.lastTimestamp'`
4. **Classify**: Is this a test issue, deployment issue, or actual bug?
5. **Record**: Add to failures list with classification
6. **Decide**: Continue testing or enter debug loop (Phase 7)

### Continue vs Debug Decision

- **Continue**: If failure is isolated and other tests are independent
- **Debug**: If failure indicates a systemic issue (service down, config error, wrong image)

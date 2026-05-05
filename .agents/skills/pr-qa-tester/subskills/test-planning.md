# Phase 5: Test Plan Generation

Generate a structured test plan based on the inferred flows from Phase 1.

## Test Categories

### 1. Happy Path Validation

Test the primary intended flow end-to-end.

For each inferred flow:
- Construct a valid request with realistic test data
- Execute through the entry point
- Verify the expected response
- Verify downstream state changes (DB records, queue messages, logs)

### 2. Negative Cases

Test error handling and boundary conditions:

- **Invalid input**: Missing required fields, malformed data, wrong types
- **Unauthorized access**: Missing/invalid auth headers
- **Not found**: Non-existent entity IDs
- **Duplicate requests**: Idempotency validation

### 3. Integration Validation

Test service-to-service interactions:

- Verify downstream service receives the expected request
- Verify response propagation back to the caller
- For async flows (queues): verify message published and consumed

### 4. Entity/State Verification

Verify data consistency after the flow:

- Database records created/updated correctly
- Cache entries set/invalidated
- Experiment evaluations recorded
- Audit logs generated

## Test Plan Template

Present the plan to the user for approval:

```
## Test Plan for <PR title>

### Prerequisites
- [ ] Service <X> deployed with PR commit
- [ ] Service <Y> available (base pod)
- [ ] Test data prepared (merchant ID, payment ID, etc.)

### Test Cases

#### Happy Path
| # | Test Case | Input | Expected Output | Verify |
|---|-----------|-------|-----------------|--------|
| 1 | <name> | <input summary> | <expected response> | <what to check> |

#### Negative Cases
| # | Test Case | Input | Expected Output |
|---|-----------|-------|-----------------|
| 1 | <name> | <invalid input> | <error response> |

#### Integration
| # | Test Case | Flow | Verify |
|---|-----------|------|--------|
| 1 | <name> | A → B → C | <downstream effect> |

#### State Verification
| # | Check | Command | Expected |
|---|-------|---------|----------|
| 1 | <what> | <kubectl/curl command> | <expected state> |
```

## Test Data Strategies

### Use Existing Test Entities

- Check if the service has test/sandbox merchants pre-configured
- Look for seed data in the repo (fixtures, migrations, test configs)
- Use devstack-specific test data documented in repo READMEs

### Create Test Entities

If no test data exists:
- Create via API (admin endpoints, internal routes)
- Insert directly via kubectl exec + database client
- Document created entities for cleanup

## Approval Gate

Present the test plan and wait for explicit user approval before executing. This prevents:
- Unintended state mutations
- Testing against wrong endpoints
- Missing important test cases the user knows about

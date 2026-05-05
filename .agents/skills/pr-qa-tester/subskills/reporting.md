# Phase 8: Reporting

Generate a comprehensive test report after all test execution and debugging is complete.

## Report Generation

### Collect Results

Ensure all test results are captured in a JSON file:

```json
{
  "test_cases": [
    {"name": "...", "type": "happy_path|negative|integration|state_verification", "status": "pass|fail|skip", "details": "...", "fix_applied": "..."}
  ],
  "failures": ["failure description 1"],
  "fixes_applied": ["fix description 1"],
  "entity_changes": ["entity change description 1"]
}
```

### Generate Report

Use the bundled script:

```bash
~/.agents/skills/e2e-orchestrator/scripts/generate-report.sh \
  --pr-url "<PR_URL>" \
  --services "service1,service2" \
  --flows "flow1|flow2" \
  --results /tmp/e2e-results.json \
  --output /tmp/e2e-report.md
```

Or generate inline if the script is not available — use the same markdown structure.

## Report Structure

The report includes:

1. **Summary table**: PR URL, timestamp, verdict (PASS/FAIL), test counts
2. **Services deployed**: List of services and their deployment method
3. **Flows tested**: Each E2E flow that was validated
4. **Test cases table**: All test cases with name, type, status, details
5. **Failures detected**: List of failures with context
6. **Fixes applied**: Changes made during the debug loop
7. **Entity/state changes**: Mutations observed during testing

## Proof-Based Completion (MANDATORY)

Every claim in the report MUST have observable evidence. Never assume fixes worked.

| Claim | Required Proof |
|-------|---------------|
| "Bypass flow works" | Log line showing config loaded + correct URL used |
| "Auth works" | Actual HTTP response (not 401) from target service |
| "Deployment healthy" | `kubectl get pods` showing Running status |
| "CI passed" | GitHub Actions link with build status |
| "Contract verified" | Actual response body from target service endpoint |

If proof cannot be obtained (e.g., service down), clearly state "BLOCKED" with reason instead of assuming success.

## Post-Report Actions

After presenting the report:

1. **If all tests pass**: Suggest the PR is ready for merge (from an E2E perspective)
2. **If failures remain**: Summarize blockers and suggest next steps
3. **Ask user**: Whether to post the report as a PR comment

### Post Report as PR Comment

```bash
gh pr comment <PR_URL> --body-file /tmp/e2e-report.md
```

## Cleanup

After testing is complete, ask the user about cleanup:

- **Keep pods**: For further manual testing
- **Delete pods**: To free devstack resources
  ```bash
  # Delegate to /devstack or run directly
  kubectl delete deployment -n <namespace> -l devstack_label=<label>
  ```
- **Keep testing branch**: If fixes were pushed
- **Delete testing branch**: If fixes were cherry-picked into the PR

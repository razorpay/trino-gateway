# Phase 7: Debug Loop

When test failures indicate code or configuration issues, iterate to fix them.

## Triage

Classify the failure:

| Category | Action |
|----------|--------|
| **Deployment issue** (pod crash, config error) | Delegate to `/devstack` for debugging |
| **Code bug** (wrong behavior, unexpected response) | Fix code → rebuild → redeploy |
| **Test issue** (wrong test data, incorrect assertion) | Fix test and re-run |
| **Missing dependency** (service not deployed) | Deploy the missing service |
| **Experiment/config issue** (feature flag state) | Update Splitz/config and retry |

## Fix → Rebuild → Redeploy Cycle

### Option A: Devspace Hot-Reload (Preferred)

If devspace is set up for the service:

1. Fix the code locally
2. Devspace auto-syncs to the pod
3. Service restarts automatically (depending on language/framework)
4. Re-run the failing test

Best for: Go, Node.js services with file-watch restart. May not work for Java services that need recompilation.

### Option B: CI/CD Rebuild (Testing Branch — MANDATORY)

If devspace is not available or the fix requires a full build:

**RULE: Never test changes directly on main PR branches. Always use a testing branch.**

1. Create a testing branch from the PR branch:
   ```bash
   git checkout -b <pr-branch>_testing
   # Apply fix using backend-engineer skills
   git add <files>
   git commit -m "fix: <description>"
   git push origin <pr-branch>_testing
   ```
2. Create a testing PR (testing branch → PR branch):
   ```bash
   gh pr create --base <pr-branch> --head <pr-branch>_testing --title "fix: <description> - E2E testing"
   ```
3. If CI not triggered → check merge conflicts FIRST, then workflow triggers
4. Wait for CI to build the new image
5. Get the new commit SHA and update helmfile
6. Redeploy via `helmfile sync`
7. Wait for pods to become ready
8. Re-run the failing test

### Anti Hit-and-Trial Rule

If similar fixes repeat without new diagnostic signal:
1. STOP iterative guessing
2. Add logging/instrumentation to the code to get better signal
3. Refine hypothesis based on new evidence
4. Explain root cause reasoning to user before next attempt
Iteration without new signal is forbidden.

### Option C: Configuration Fix

For configuration issues (helm values, env vars, secrets):

1. Identify the misconfiguration from logs/events
2. Fix in kube-manifests helmfile values
3. Redeploy via helmfile sync (delegate to `/devstack`)
4. Re-run the failing test

## Iteration Limits

- **Max deployment retries**: 2 (after 2 failed deployments, escalate to user)
- **Max test retries**: 1 per test case (re-run once after fix)
- **Max debug cycles**: No hard limit, but ask user after 3 cycles if they want to continue

## Escalation

When the debug loop cannot resolve the issue:

1. Summarize what was tried
2. Share relevant logs and error messages
3. Suggest next steps (e.g., "This may require changes in the downstream service")
4. Ask user how to proceed:
   - Continue debugging
   - Skip this test case and move on
   - Abort testing

## Tracking Fixes

Record every fix applied for the final report:

```json
{
  "fixes_applied": [
    "Fixed null entityId handling in ScroogeRefundApiProcessor line 45",
    "Updated helm values to include SCROOGE_INTERNAL_URL env var"
  ]
}
```

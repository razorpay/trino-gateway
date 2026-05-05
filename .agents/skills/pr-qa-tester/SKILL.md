---
name: pr-qa-tester
description: Autonomous end-to-end testing orchestrator for validating Pull Requests on devstack. Given a PR URL, analyzes the diff, infers impacted services and flows, deploys to devstack, generates test plans, executes tests, debugs failures, and generates reports. Use when (1) testing a PR end-to-end on devstack, (2) validating a code change against downstream services, (3) user says "test this PR", "validate this PR", "e2e test", "run E2E", or provides a PR URL for testing, (4) deploying and testing a feature branch, (5) generating an E2E test report for a PR.
---

# E2E Testing Orchestrator

Orchestrate the full PR validation lifecycle: analyze → deploy → test → debug → report.

## Invocation

```
/pr-qa-tester

Test this PR: https://github.com/razorpay/<repo>/pull/<number>
```

## Input Requirements

- **PR URL** (required): GitHub PR URL
- **PR context** (optional): Additional context from the user about what to test

## Execution Order (MANDATORY)

```
Phase 0: Pre-Execution Context    → Load repo skill, run structured PR review
Phase 1: Context Discovery        → Analyze PR, infer flows, check infra deps
Phase 2: Flow Confidence Check    → Present findings, get approval
Phase 3: Service Enumeration      → Determine what to deploy
Phase 4: Deployment               → Deploy via helmfile sync
Phase 5: Test Plan Generation     → Generate test cases
Phase 6: Test Execution           → Run tests, collect results
Phase 7: Debug Loop               → Fix failures via testing branch, iterate
Phase 8: Reporting                → Generate final report with proof
```

---

## Phase 0: Pre-Execution Context (REQUIRED)

Before ANY deployment, debugging, or coding task, you MUST:

### 0a. Load Repository Skill

Check if a repo skill exists for the PR's repository. Repo skills are located inside the locally cloned repo directory:
```bash
# e.g., for scrooge: ls /Users/<user>/scrooge/.agents/skills/ 2>/dev/null
ls <local_repo_clone_path>/.agents/skills/ 2>/dev/null
```
If missing, ask user: "Repository skill not found. Continue without repo context or install skill?"
Do NOT silently continue.

### 0b. Structured PR Review

Use `code-review` agent to analyze the PR. The review MUST identify:
- Logical correctness
- Infra dependencies (new services, new configs, new experiments)
- CI/CD risks (workflow triggers, merge conflicts, image builds)
- Helmfile expectations (does the service have a helm chart in kube-manifests?)
- Config drift risks (devstack vs prod property differences)
- Contract compatibility (request/response shape between services)

You are optimizing for **failure prediction, not reaction**.

### 0c. Environment Drift Check

Compare devstack vs production configs for the impacted service:
- Base paths, hostnames, ports
- Feature flags (`splitz.mock`, env-specific overrides)
- Auth credentials (env var references vs defaults)
- DNS resolution (short names vs FQDN for cross-namespace calls)

See [references/infra-deployment.md](references/infra-deployment.md) for persistent platform knowledge.

---

## Phase 1: Context Discovery

Analyze the PR to understand changes and infer testable flows. See [subskills/context-discovery.md](subskills/context-discovery.md) for detailed steps.

1. Extract PR metadata: `gh pr view <URL> --json title,body,files,commits`
2. Read the diff: `gh pr diff <URL>`
3. **Check merge status**: `gh pr view <URL> --json mergeable,mergeStateStatus` — merge conflicts block CI
4. Identify impacted services from code patterns (HTTP clients, gRPC stubs, queue publishers, experiment checks)
5. Use `rzp-discover` subagents to understand service interactions:
   - Route to the correct subgroup agent based on the repo
   - Ask about the specific flow being modified
6. If a repo skill exists (`.agents/skills/` in the repo), read it for architecture context
7. Infer testable flows with entry points, call chains, and expected outcomes
8. **Cross-repository reasoning**: Check if changes require updates in other repos (kube-manifests, config repos, experiment setup)
9. Compute confidence score per [references/confidence-model.md](references/confidence-model.md)

## Phase 2: Flow Confidence Check

Present findings to the user and wait for approval before proceeding:

```
## Context Discovery Results

**PR**: <url>
**Primary service**: <name>
**Changes**: <one-line summary>
**Merge Status**: <clean / conflicts>

### Inferred Flows
1. <flow with entry point and call chain>

### Impacted Services
- <service> (deploy PR commit)
- <service> (base pod / latest master)

### Cross-Repo Dependencies
- <kube-manifests change needed? Y/N>
- <experiment setup needed? Y/N>
- <config update needed? Y/N>

### Predicted Risks
- <risk from PR review>

### Assumptions
- <assumption>

### Confidence: <score>/100
```

**Proceed only after user approves.** If confidence < 50, ask for guidance.

## Phase 3-4: Service Enumeration & Deployment

Determine deployment strategy. See [subskills/deployment.md](subskills/deployment.md) and [references/infra-deployment.md](references/infra-deployment.md) for platform knowledge.

**Pre-deployment checklist:**
1. Verify CI image is built: `gh pr checks <URL>` — look for build jobs
2. If CI not triggered → check merge conflicts FIRST (see CI Not Triggered Rule below)
3. Resolve conflicts if needed → push → wait for CI
4. Locate helmfile chart in `kube-manifests/helmfile/charts/<service>/`
5. Edit `helmfile/helmfile.yaml` with correct commit SHA
6. Run `helmfile sync` with custom devstack label

**NEVER assume helmfiles are in the service repo. They are centrally defined in kube-manifests.**

Verify all pods are healthy after deployment. If unhealthy, check logs before retrying.

## Phase 5: Test Plan Generation

Generate test cases and present for user approval. See [subskills/test-planning.md](subskills/test-planning.md).

Categories:
- **Happy path**: Primary flow end-to-end
- **Negative cases**: Invalid input, auth failures, not-found
- **Integration**: Service-to-service communication
- **State verification**: DB records, cache, queues
- **Contract verification**: Request/response shapes between services

Present the test plan as a table and wait for user approval before executing.

## Phase 6: Test Execution

Execute approved tests and collect results. See [subskills/test-execution.md](subskills/test-execution.md).

Methods:
- API calls via `curl` (through ingress or kubectl exec)
- Log monitoring via `kubectl logs`
- State checks via database queries / cache lookups

Collect results as structured JSON for report generation.

## Phase 7: Debug Loop

When tests fail, iterate to fix. See [subskills/debug-loop.md](subskills/debug-loop.md).

**Code fix workflow (MANDATORY):**
```
code change → create testing branch → raise testing PR → run CI → validate
```
Use `backend-engineer` skills for development work. Never push fixes directly to the PR's main branch.

**Anti Hit-and-Trial Rule**: If similar fixes repeat without new signal, STOP. Improve logging, add debugging instrumentation, refine hypothesis. Iteration without new evidence is forbidden.

Limits: 2 deployment retries, 1 test retry per case. Escalate to user after 3 debug cycles.

## Phase 8: Reporting

Generate the final test report. See [subskills/reporting.md](subskills/reporting.md).

**Proof-Based Completion**: Every claim in the report MUST have observable evidence:
- CI success screenshots/links
- Deployment verification (pod status)
- Endpoint/contract validation (actual responses)
- Log evidence for flow activation

Use the bundled script:
```bash
~/.agents/skills/pr-qa-tester/scripts/generate-report.sh \
  --pr-url "<URL>" --services "svc1,svc2" --flows "flow1|flow2" \
  --results /tmp/e2e-results.json --output /tmp/e2e-report.md
```

Optionally post as a PR comment: `gh pr comment <URL> --body-file /tmp/e2e-report.md`

---

## Behavioral Rules

### CI Not Triggered Rule

If CI pipeline does not start, check in this order:
1. **Merge conflicts** — `gh pr view --json mergeable` (most common cause)
2. **Workflow triggers** — check `.github/workflows/` for `on:` conditions
3. **Branch filters** — CI may only trigger for PRs against `master`
4. **Concurrency groups** — previous run might be blocking
Do NOT assume runtime failure. Do NOT try empty commits or close/reopen before checking conflicts.

### Scaling Safety Constraint (STRICT)

**NEVER scale down base pods.** Base pods (`-base` suffix) are system-critical shared infrastructure. Scaling them breaks other developers' workflows. Only modify your own custom-labeled resources. Only change base resources if explicitly instructed by user.

### Anti Hit-and-Trial Policy

If similar fixes repeat without producing new diagnostic signal:
1. STOP iterative guessing
2. Improve logging / add debugging instrumentation
3. Refine hypothesis with new evidence
4. Explain root cause reasoning to user
Iteration without new signal is forbidden.

### Proof-Based Completion

Never assume fixes worked. Completion requires ALL of:
- CI success evidence (build passed link)
- Deployment verification (pods Running, no CrashLoopBackOff)
- Endpoint/contract validation (actual curl response)
- Observable proof (log lines confirming flow activation)
No assumption-based success claims.

### Memory Optimization Rule

Avoid repeating identical analysis. If repetition detected, inform user and switch debugging strategy.

### Cross-Repository Awareness

When deployment or testing fails, ALWAYS evaluate:
- Missing helmfile entry in kube-manifests
- Service registration absence (e.g., splitz experiment not created)
- CI pipeline expectations mismatch
- Config drift between environments
Assume multi-repo dependency by default.

### Plugin Availability Handling

If a required plugin/skill is missing, prompt user clearly instead of degrading silently.

---

## Decision Rules

**Ask user when:**
- Confidence < 70%
- Destructive actions required (deleting pods, dropping data)
- Deployment strategy is ambiguous
- Test plan needs approval (always)
- Required plugin/skill is missing

**Act autonomously when:**
- Researching flows via discover plugins
- Analyzing logs and events
- Running read-only diagnostic commands
- Computing confidence scores
- Creating testing branches for fixes

## Testing Patterns

See [references/testing-patterns.md](references/testing-patterns.md) for:
- API testing patterns (REST, gRPC/Twirp)
- State verification (MySQL, PostgreSQL, Redis, SQS)
- Log analysis patterns
- Service-type patterns (Java/Spring, Go, PHP)
- Experiment/feature flag validation (Splitz)

## Infrastructure Knowledge

See [references/infra-deployment.md](references/infra-deployment.md) for persistent platform context:
- Helmfile architecture and kube-manifests structure
- CI/CD workflow behavior and triggers
- Devstack deployment patterns
- Cross-namespace service discovery
- Environment drift detection
- Experiment testing strategies (Splitz override, mock, Redis cache)

## State Tracking

Maintain a running state indicator throughout the workflow:

```
STATE: [Pre-Execution | Discovery | Deployment | Testing | Debugging | Reporting]
```

Provide concise progress updates at each phase transition.

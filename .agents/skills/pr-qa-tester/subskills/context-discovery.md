# Phase 1: Context Discovery

Analyze the PR to understand what changed, what services are impacted, and what flows need testing.

## Step 1: Extract PR Metadata

```bash
gh pr view <PR_URL> --json title,body,files,additions,deletions,baseRefName,headRefName,commits
gh pr diff <PR_URL>
```

Capture:
- **Files changed**: Map to service modules (e.g., `src/main/java/...` → Java service)
- **Additions vs deletions**: Gauge change scope
- **PR description**: Extract intent, JIRA IDs, testing notes
- **Commits**: Understand incremental changes

## Step 2: Identify Impacted Services

From the PR diff, determine:

1. **Primary service**: The repo the PR belongs to
2. **Downstream services**: Services called by the changed code (look for HTTP clients, gRPC stubs, queue publishers)
3. **Upstream dependencies**: Services that call into the changed code
4. **Infrastructure dependencies**: Databases, caches, queues, experiments (Splitz)

### Inference Patterns

| Code Pattern | Indicates |
|-------------|-----------|
| `http.Get/Post`, `curl`, REST client | Downstream service call |
| `twirp`, `grpc.Dial` | gRPC/Twirp dependency |
| `sqs.SendMessage`, queue publisher | Async downstream |
| `splitzClient.evaluate` | Splitz experiment dependency |
| `redis.Get/Set` | Cache dependency |
| `db.Query`, `repository.find` | Database dependency |
| New JSON config files | New batch type / flow configuration |

## Step 3: Use Discover Plugins

Use `rzp-discover` subagents to deepen flow understanding:

1. Identify which subgroup owns the service (use `rzp-discover:brainstorm` if unsure)
2. Launch the relevant subgroup agent (e.g., `rzp-discover:payments-processing-platform` for batch service)
3. Ask about the specific flow: "How does [changed component] interact with [downstream service]?"

If a **repo skill** exists for the service (check `<local_repo_clone_path>/.agents/skills/`), read it for architecture context. Repo skills are located inside the locally cloned repo directory (e.g., `/Users/<user>/scrooge/.agents/skills/`).

## Step 4: Infer Testable Flows

From the analysis, produce a list of testable flows:

```
Flow: [Name]
Entry point: [API endpoint / batch trigger / queue message]
Steps: [service A] → [service B] → [service C]
Expected outcome: [state change / response / side effect]
Dependencies: [services that must be deployed]
```

## Step 4b: Cross-Repository Dependency Analysis

Reason across repositories — do NOT assume local completeness:

1. **Helmfile check**: Does `kube-manifests/helmfile/charts/<service>/` exist? If not, deployment will fail.
2. **Config drift**: Compare `application-devstack.properties` vs `application-prod.properties` for base paths, ports, DNS
3. **Experiment setup**: If code checks Splitz experiments, verify the experiment exists or plan override strategy
4. **Credential alignment**: Verify auth credentials match between calling and receiving services (e.g., batch's `RECON_REFUND_SECRET` must match scrooge's `BATCH_AUTH_PASSWORD`)
5. **Cross-namespace DNS**: Services in different K8s namespaces need FQDN, not short names

## Step 4c: Environment Drift Detection

Compare devstack vs production for EACH config property used by the changed code:

| Check | Devstack | Production | Risk |
|-------|----------|------------|------|
| Base paths | Short hostname? Missing `/v1/`? | Full URL with path | URL routing failure |
| Ports | Direct app port vs nginx? | Through load balancer | Connection refused |
| Auth | Mock/test credentials? | Real credentials | Auth mismatch |
| Feature flags | `splitz.mock=true`? | Actual Splitz service | Different behavior |

## Step 5: Compute Confidence Score

Use the scoring rubric in [references/confidence-model.md](../references/confidence-model.md).

Present findings to user:

```
## Context Discovery Results

**PR**: <url>
**Primary service**: <name>
**Changes**: <summary>

### Inferred Flows
1. <flow description>
2. <flow description>

### Impacted Services
- <service> (primary, deploy PR commit)
- <service> (downstream, use base pod or deploy latest master)

### Assumptions
- <assumption 1>
- <assumption 2>

### Confidence: <score>/100
- Flow Understanding: <score>/100
- Service Mapping: <score>/100
- Deployment Clarity: <score>/100
```

**Proceed only after user approves the inferred flows.**

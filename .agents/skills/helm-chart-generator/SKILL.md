---
name: helm-chart-generator
description: >-
  Generate Helm charts and Kubernetes manifests for Razorpay services, then open PRs on
  razorpay/kube-manifests via MCP tools (no local git commands). Supports onboarding new
  services, adding components (workers, cronjobs, HPA, PDB, KEDA, ConfigMap) to existing
  services, onboarding UPI gateways, and previewing chart files. All chart generation and
  PR creation happens server-side through MCP tool calls. Use this skill whenever the user
  asks to onboard a service to Kubernetes, scaffold Helm charts, add workers/cronjobs/
  HPA/PDB/KEDA to an existing service, onboard a UPI gateway, or create kube-manifests PRs.
  Triggers on: '/helm-chart-generator', 'generate helm chart', 'create helm chart',
  'onboard service to k8s', 'add worker to service', 'add cronjob', 'deploy my service',
  'I need k8s manifests', 'kube-manifests PR', 'onboard UPI gateway', 'helm chart',
  'IngressRoute', 'deploy a new microservice'.
user-invocable: true
argument-hint: "<describe what to generate — e.g. 'onboard payment-svc as web service in prod'>"
metadata:
  version: "1.0.0"
  category: "Infrastructure"
  mcp_servers:
    - helm-chart-generator
---

# Helm Chart Generator

Generate production-ready Helm charts for Razorpay services and open PRs on `razorpay/kube-manifests`. Provide a natural-language description of what you need — the skill routes to the right MCP tool automatically.

## MCP-Only PR Creation (Important)

**All chart generation and PR creation MUST happen through MCP tool calls.** The MCP server handles git operations (branch creation, file commits, PR opening) server-side via the GitHub API. You must NEVER:
- Clone or checkout `razorpay/kube-manifests` locally
- Run local `git` commands (`git add`, `git commit`, `git push`, `gh pr create`) to create kube-manifests PRs
- Generate Helm chart YAML files on the local filesystem and then push them manually

**Always prefer all-in-one MCP tools** (`onboard_service`, `add_component`, `create_helm_chart_pr`, `onboard_gateway`) that generate charts AND create the PR in a single call. Do NOT split this into `generate_helm_chart_files` followed by `create_pr_from_local_files` — use the combined tool instead.

## MCP Prerequisite

The `helm-chart-generator` MCP server must be configured in Claude Code settings:

```json
{
  "mcpServers": {
    "helm-chart-generator": {
      "type": "sse",
      "url": "https://helm-generator-mcp.razorpay.com/mcp"
    }
  }
}
```

The server is also reachable cluster-internally at `helm-generator-mcp.jarvis.svc.cluster.local:8000`.

## Inputs

The user provides a free-form prompt. Extract these from the prompt (ask only for missing **required** fields):

| Field | Required | Description |
|-------|----------|-------------|
| Service name | Yes | Name of the service (e.g. `offer-payout`) |
| Intent | Yes | What to do — onboard new service, add component, onboard gateway, or preview |
| Image | Yes (for new) | Container image URL (e.g. `c.rzp.io/razorpay/my-svc:v1.0`) |
| Environment | Depends | Target env: `stage`, `prod`, `ops`, `ai`, `dev-serve`, etc. |
| Service type | For onboard | `web`, `worker`, `cronjob`, or `mixed` |
| Namespace | For onboard | Kubernetes namespace |

All other parameters (ports, replicas, ingress hosts, HPA, secrets, ConfigMap, service account) are optional with sensible defaults.

## Procedure

### Step 1 — Classify the Request

Parse the user's prompt and determine which MCP tool to call. **Prefer all-in-one tools (top 4 rows) that generate charts AND create the PR in a single MCP call:**

| Priority | User wants... | MCP tool | Key signals |
|----------|---------------|----------|-------------|
| **Default** | New service, first time in K8s | `onboard_service` | "onboard", "new service", "first time", "deploy new" |
| **Default** | Add worker/cronjob/HPA/PDB/KEDA/ConfigMap to existing service | `add_component` | "add worker", "add cronjob", "add HPA", "existing service" |
| **Default** | Single-environment chart + PR | `create_helm_chart_pr` | Mentions one specific env, simple deployment |
| **Default** | UPI gateway integration | `onboard_gateway` | "UPI gateway", "gateway onboarding" |
| Preview | Preview files only, no PR | `generate_helm_chart_files` | "preview", "show me first", "don't create PR yet" |
| Fallback | User explicitly provides raw YAML content and wants a PR from it | `create_pr_from_local_files` | User pastes file contents, "create PR from these files" |

**Routing rules:**
- Always route to a **Default** tool when the user wants chart generation + PR. These tools handle everything server-side in one call.
- Only use `generate_helm_chart_files` when the user explicitly asks to preview without creating a PR.
- Only use `create_pr_from_local_files` when the user has already-written YAML content they want to submit as-is. Do NOT use it as a two-step workaround (generate files first, then create PR) — use the all-in-one tool instead.

If unclear, ask: "Are you onboarding a brand new service or adding to an existing one?"

### Step 2 — Gather Missing Parameters

Only ask for parameters that are **required** and not provided. Do NOT ask about optional params — use defaults.

**For `onboard_service`** (required: service_name, service_type, namespace, image):
- If service_type is `web` or `mixed` and no ingress_hosts provided, auto-derive them
- If service_type is `worker` or `mixed`, ask for worker configs (name, queue_type, queue_name)

**For `add_component`** (required: service_name, component_type):
- For `worker`: ask for name, queue_type, queue_name
- For `cronjob`: ask for name, schedule
- For `hpa`/`pdb`/`keda`: ask for target_deployment if not obvious

**For `create_helm_chart_pr`** (required: app_name, environment, image):
- All other params have defaults

**For `onboard_gateway`** (required: gateway_name, image):
- All other params have defaults

### Step 3 — Call the MCP Tool

Build the tool call with extracted + default parameters.

**Critical**: These parameters must be **JSON strings**, not native objects:
- `ingress_hosts` — e.g. `'{"stage":"my-svc.stage.razorpay.in","prod":"my-svc.razorpay.in"}'`
- `workers` — e.g. `'[{"name":"bg-proc","queue_type":"sqs","queue_name":"my-queue"}]'`
- `cronjobs` — e.g. `'[{"name":"nightly","schedule":"0 2 * * *"}]'`
- `env_vars` — e.g. `'{"LOG_LEVEL":"info"}'`
- `configmap_data` — e.g. `'{"KEY":"value"}'`
- `worker` (in add_component) — e.g. `'{"name":"proc","queue_type":"sqs","queue_name":"q"}'`
- `cronjob` (in add_component) — e.g. `'{"name":"job","schedule":"0 * * * *"}'`

### Step 4 — Return Results

- If a PR was created: show the **PR URL** prominently
- If previewing files: show the generated file contents in a code block
- Suggest `/helm-monitor <PR URL>` if the user needs the S3 chart URL

## Service Types for onboard_service

| Type | What gets generated |
|------|-------------------|
| `web` | Deployment + Service + IngressRoute + HPA (optional) |
| `worker` | Workers only (requires `workers` list) |
| `cronjob` | CronJobs only (requires `cronjobs` list) |
| `mixed` | Deployment + Service + IngressRoute + Workers + CronJobs |

## Auto-Derived Defaults

If not explicitly provided:
- **Ingress host**: `stage` -> `{app}.stage.razorpay.in`, `prod` -> `{app}.razorpay.in`
- **Node selector**: `ai` -> `node.kubernetes.io/worker-generic`, others -> `node-role.kubernetes.io/worker-generic`
- **Middleware namespace**: `stage`/`dev-serve` -> `traefik-v2-flux`, others -> `traefik-v2`
- **Port**: 8080
- **Replicas**: stage=1, prod=2
- **Resources**: requests 100m CPU / 256Mi memory, limits 512Mi memory

## Environment Resolution

Valid environments: `stage`, `prod`, `ai`, `dev-serve`, `ops`, `perf`, `perf1`, `perf2`, `bvt`, `cde`, and many more.
Auto-aliases: `ops-common` -> `ops`, `prod-green` -> `prod`, `stage-white` -> `stage`, `dev-serve-eks` -> `dev-serve`.

## Worker & CronJob Config Schemas

**Worker** (minimum):
```json
{"name": "my-worker", "queue_type": "sqs", "queue_name": "my-queue"}
```
Optional: `stage_replicas`, `prod_replicas`, `enable_keda`, `keda_trigger_type`, `keda_metric_query`, `keda_threshold`.

**CronJob** (minimum):
```json
{"name": "nightly-cleanup", "schedule": "0 2 * * *", "command": ["/bin/cleanup"]}
```

## Error Handling

- If the MCP tool returns an error, show the error message and suggest corrections
- If `onboard_service` rejects `environment` param, use `create_helm_chart_pr` for single-env charts
- If a required MCP server is not connected, instruct user to configure it in settings

## Common Mistakes

- **Using local git commands instead of MCP tools** — never `git clone kube-manifests`, `git push`, or `gh pr create` locally. All PR creation goes through the MCP server.
- **Splitting generation + PR into two calls** — do NOT call `generate_helm_chart_files` then `create_pr_from_local_files`. Use `onboard_service`, `add_component`, `create_helm_chart_pr`, or `onboard_gateway` instead — they do both in one call.
- Passing Python dicts instead of JSON strings for `ingress_hosts`, `workers`, `cronjobs`, `env_vars`, `worker`, `cronjob`
- Using `onboard_service` when the service already exists (use `add_component` instead)
- Forgetting `queue_type` and `queue_name` for worker configs
- Not calling `monitor_helm_s3_push` after PR creation when user needs the S3 URL

## Examples

**Onboard a new web service:**
> /helm-chart-generator onboard payment-reconciler as a web service in reconciler namespace, image c.rzp.io/razorpay/payment-reconciler:v2.1.0, stage and prod hosts, HPA min 2 max 8, secret payment-reconciler-secrets

**Add a worker to existing service:**
> /helm-chart-generator add SQS worker invoice-processor to billing-engine, queue billing-invoices-q, 1 stage replica, 4 prod replicas

**Single-env chart:**
> /helm-chart-generator create chart for offer-payout in ops env, image razorpay/offer-payout:latest, with configmap and service account

**Preview only:**
> /helm-chart-generator preview cronjob nightly-report-gen, runs at 2am daily, image c.rzp.io/razorpay/report-gen:latest, prod env, don't create PR

## Full Parameter Reference

For complete parameter tables for all MCP tools, consult `references/tool-parameters.md`.

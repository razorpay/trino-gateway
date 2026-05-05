---
name: devstack
description: Deploy and debug helmfile-based services with automated configuration validation, intelligent troubleshooting, and autonomous error recovery. Use when (1) Setting up devstack for new developers, (2) Onboarding new applications to devstack, (3) Creating helm charts and ephemeral resources (DB, cache, SQS, SNS), (4) Deploying helmfile services with devstack labels, (5) Debugging pod failures and crashes, (6) Validating helmfile/helm chart configurations, (7) Troubleshooting ImagePullBackOff, CrashLoopBackOff, OOMKilled errors, (8) Auto-fixing resource limits, probes, and configuration issues, (9) Post-deployment health monitoring and recovery, (10) Setting up devspace for live code sync and debugging.
version: "1.0.6"
category: "Infrastructure"
author: "razorpay"
---

# Devstack Deployment & Debug Assistant

## Prerequisites

- `kube-manifests` repository cloned locally (auto-cloned if not found — see [Path Detection](references/path-detection.md))
- `kubectl` configured with devstack cluster access
- `helmfile` installed
- Helmfile directory configured — see [Configuration](references/configuration.md)

💡 **For automatic PR creation** install GitHub MCP server or `gh` CLI — see [Configuration](references/configuration.md).

## ⚠️ MANDATORY: Kubernetes Tool Priority

**For ALL Kubernetes operations (get pods, describe, logs, events, exec, etc.), use tools in this order:**

1. **Friday Kubernetes MCP** (`kubectl_execute` tool) — use this first if available. It runs kubectl inside the cluster with proper context and avoids local credential issues.
2. **Local `kubectl`** — fallback if Friday MCP is not available or returns an error.

**How to use**:
- **Prefer Friday MCP**: attempt every kubectl operation via `kubectl_execute` first.
- **Fall back to local `kubectl --context dev-serve`** if Friday MCP is unavailable (tool not found) OR if a specific operation fails or returns an error. Do not stop — retry the same operation with local kubectl immediately.
- **Per-operation fallback**: fallback applies to each individual operation, not the whole session. If Friday MCP works for `get pods` but fails for `exec`, use local kubectl only for that exec and continue using Friday MCP for the rest.

## ⚠️ MANDATORY: Context Verification

**Before ANY kubectl, helmfile, or helm operation**, verify the `dev-serve` context exists:

```bash
kubectl config get-contexts dev-serve
```

**If found**: use `--context dev-serve` for all `kubectl` commands and `--kube-context dev-serve` for all `helmfile` and `helm` commands throughout this skill.

**If NOT found — STOP and ask the user**:

```
⚠️ The `dev-serve` kubectl context was not found on your machine.

Do you have the devstack context configured under a different name?
Run: kubectl config get-contexts
and share the context name you use for devstack.

If you don't have a devstack context configured at all, you'll need to run the onboarding flow:
  /devstack  →  "Setup devstack on my machine for the first time"
```

**If the user provides an alternative context name**: do NOT use it automatically. First show them:

```
⚠️ Before proceeding, please confirm this context points to your devstack cluster and NOT a staging or production cluster.

Run: kubectl config get-contexts <their-context-name>

Check the cluster/server URL — confirm it points to the devstack cluster and not a production or staging cluster.
```

Wait for explicit user confirmation that it is safe before proceeding. If they confirm, proceed using their context name.

**If the user confirms they have no devstack context**: direct them to run the onboarding flow and stop.

> Do NOT auto-run the onboarding flow. Wait for the user's response before proceeding.

## ⚠️ MANDATORY: Helmfile Version Detection

**Before any helmfile operation**, detect the installed version:

```bash
helmfile --version
```

**If 0.x** (e.g. `0.171.0`): proceed normally. Use `-f helmfile.yaml` for all helmfile commands.

**If 1.x**: two issues must be handled before proceeding:

### Issue 1 — Go template rendering (`.gotmpl` extension required)

Helmfile 1.x does not render Go templates in `.yaml` files. The current `helmfile.yaml` uses Go templates extensively (`{{ .Values.devstack_label }}` etc.), so it must be referenced with a `.gotmpl` extension.

Create a symlink (one-time, local):
```bash
cd <kube-manifests-repo>/helmfile
ln -sf helmfile.yaml helmfile.yaml.gotmpl
```

**CRITICAL**: Never stage or commit `helmfile.yaml.gotmpl`. If the user attempts to do so, stop them immediately.

Then use `-f helmfile.yaml.gotmpl` in place of `-f helmfile.yaml` for all helmfile commands throughout this session.

### Issue 2 — Multi-document YAML (`---` separator)

Helmfile 1.x dropped support for multi-document YAML. The current `helmfile.yaml` uses a `---` separator between the `environments` and `releases` sections. If helmfile 1.x errors on this, inform the user:

```
⚠️ helmfile 1.x does not support the multi-document YAML format (--- separator) used in helmfile.yaml.

To work around this, merge the two sections into a single YAML document:
1. Open helmfile/helmfile.yaml.gotmpl (the symlink you just created)
2. Remove the `---` line between the environments and releases sections
3. Save as helmfile-merged.yaml.gotmpl in the same directory
4. Use -f helmfile-merged.yaml.gotmpl for all helmfile commands this session
```

> Do NOT commit `helmfile-merged.yaml.gotmpl`. It is a local workaround only.

## ⚠️ CRITICAL: Deployment Method

**ALWAYS use helmfile as the default deployment method.** werf is a tool used internally by helmfile — it is not an alternative to it. The only alternative deployment method is v2 (a separate system), and you should only use it if the user **explicitly** says "use v2". Never infer the deployment method from context — if unclear, use helmfile.

## ⚠️ CRITICAL: Helm Chart Location

**ALWAYS create/update helm charts in `helmfile/charts/<service-name>/` within the kube-manifests repository ONLY.**

## When to Use

- Set up devstack for a new developer
- Onboard a new service to devstack (helm chart, ephemeral DB/cache/SQS/SNS, secrets)
- Deploy one or more services with a devstack label
- Debug crashing pods (CrashLoopBackOff, ImagePullBackOff, OOMKilled)
- Validate helmfile/helm chart configuration before deploying
- Set up devspace for live code sync
- Create a Spinnaker base pod pipeline for a service

## Quick Start

```
# First-time developer setup
/devstack  →  Setup devstack on my machine for the first time

# Onboard a new service
/devstack  →  Onboard payment-service to devstack with ephemeral database

# Deploy
/devstack  →  Deploy pg-router with abc123, api with def456

# Debug
/devstack  →  Why are pods crashing in namespace payment-service with label john

# Base pod pipeline
/devstack  →  Create base pod pipeline for payment-service
```

## Deployment Workflow (Mandatory Steps)

Every deployment MUST follow these steps in order. **Do NOT skip image validation.**

1. Locate and uncomment target service(s) in helmfile.yaml
2. Update image field if new commit specified; comment out all other services
3. Validate chart configuration and render templates (`helmfile template`)
4. **Validate ALL container images via Harbor API** (`https://harbor-image-checker.dev.razorpay.in/check-images`) — extract images from template output, verify they exist and support `linux/amd64`
5. **⛔ Deployment Gate** — only proceed if image validation passed or Harbor API was unreachable (warn user)
6. Clean deploy: `helmfile delete || true` then `helmfile sync`
7. Monitor pods and debug failures

> Skip image validation ONLY if `skip_image_validation: true` in config.json.
> Missing `linux/amd64` support is a **deployment blocker** (devstack nodes are amd64-only).

See [Deployment Subskill](subskills/deployment.md) for the full workflow.

## Subskills

| Subskill | Use when |
|---|---|
| [Deployment](subskills/deployment.md) | Deploying or updating services |
| [Debugging](subskills/debugging.md) | Troubleshooting failing or crashing pods |
| [Validation](subskills/validation.md) | Checking configurations before deployment |
| [Monitoring](subskills/monitoring.md) | Post-deployment health checks and log streaming |
| [Onboarding](subskills/onboarding.md) | Onboarding new services; adding ephemeral DB/cache/SQS/SNS; automatic PR creation |
| [User Onboarding](subskills/user-onboarding.md) | First-time devstack setup for a developer |
| [Devspace Code Sync](subskills/devspace.md) | Live code sync to running pods without rebuilding images |
| [Base Pod Readiness](subskills/base-pod-readiness.md) | Validating and auto-fixing a helm chart for base pod deployment |
| [Base Pod Pipeline](subskills/base-pod-pipeline.md) | Creating the Spinnaker pipeline to deploy a service as a base pod |

## Reference Documentation

| Reference | Contents |
|---|---|
| [Configuration](references/configuration.md) | Helmfile path setup, PR creation, autonomous behavior, output format |
| [FAQ](references/faq.md) | Common questions — TTL extension, external endpoints, NAT IPs, ImagePullBackOff |
| [Error Patterns](references/error-patterns.md) | CrashLoopBackOff, OOMKilled, ImagePullBackOff, secret errors and fixes |
| [Auto-Fix Strategies](references/auto-fix-strategies.md) | What the skill fixes automatically vs. what needs manual action |
| [Recovery Workflows](references/recovery-workflows.md) | Step-by-step recovery for common failure scenarios |
| [Helm Chart Templates](references/helm-chart-templates.md) | Template index for core, ephemeral DB/cache/SQS/SNS, secret management |
| [SNS Configurator Guide](references/sns-configurator-guide.md) | SNS topics, SQS subscriptions, localstack setup |
| [Path Detection](references/path-detection.md) | Helmfile directory auto-detection logic |

## Limitations

- Cannot auto-create secrets (security restriction) — add to credstash manually
- Cannot fix application-level bugs
- Requires `kubectl` access to the devstack cluster
- Limited to helmfile-based deployments

## Version

**Current**: 1.0.7 (2026-04-30) — See [CHANGELOG.md](CHANGELOG.md)

## Support

- Slack: `#platform-devstack`
- Docs: https://alpha.razorpay.com/repo/devstack-docs
- FAQ: [references/faq.md](references/faq.md)
- See [Maxwell CDC SOP](references/maxwell-cdc-sop.md) for ephemeral MySQL CDC setup
- See [Debezium CDC SOP](references/debezium-cdc-sop.md) for ephemeral PostgreSQL CDC setup

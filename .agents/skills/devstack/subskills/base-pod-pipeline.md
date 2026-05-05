# Base Pod Pipeline

## Purpose

Creates a Spinnaker pipeline in the `spinacode` repository to deploy a service as a long-running base pod on devstack. Base pods run the production commit, are monitored, and are deployed exclusively via Spinnaker for audit trail.

Orchestrates the full workflow:
1. Validates helm chart readiness (via [Base Pod Readiness](base-pod-readiness.md))
2. Reads validation output to build dynamic pipeline overrides
3. Generates the pipeline JSON config
4. Hands off to `/spinnaker-ops-assistant` skill + Spinnaker MCP to create the pipeline and raise the PR

## When to Use

- Setting up a new service to run as a base pod on devstack
- The service's helm chart already exists in `kube-manifests/helmfile/charts/<service>/`

## Prerequisites

### 1. spinnaker-ops-assistant skill

Check if the skill is installed:
```bash
ls ~/.claude/skills/spinnaker-ops-assistant/
```

If not found, stop and inform the user:
```
❌ The `spinnaker-ops-assistant` skill is required to raise the spinacode PR.

Install it from the agent-skills repo:
  cd <agent-skills-repo>
  make install SKILL=spinnaker-ops-assistant

Then re-run this flow.
```

### 2. Spinnaker MCP

Check if the Spinnaker MCP server is configured:
```bash
cat ~/.claude/settings.json | grep -i spinnaker
```

If not found, stop and inform the user:
```
❌ The Spinnaker MCP server is required for spinnaker-ops-assistant to work.

Add the following to your Claude settings (Settings → MCP Servers):

{
  "mcpServers": {
    "spinnaker-prod-mcp": {
      "name": "spinnaker-prod-mcp",
      "transport": "streamable-http",
      "url": "https://spinnaker-mcp.razorpay.com",
      "streamable": true
    }
  }
}

Then restart Claude and re-run this flow.
```

> Check both prerequisites before proceeding. Do not continue if either is missing.

---

## Inputs

Collect the following from the user before proceeding. Show defaults where available:

| Field | Description | Default |
|---|---|---|
| `service_name` | Service to onboard (must match helm chart directory name) | — |
| `github_repo_name` | GitHub repository name | Same as `service_name` |
| `namespace` | Kubernetes namespace | Same as `service_name` |
| `commit_txt_host` | Hostname for service | `<service>.dev.razorpay.in` |
| `slack_channel` | Slack channel for deployment notifications | `tech_deployments` |
| `slack_group_id` | Slack group ID for mentions (e.g., `S12345678`) | — |
| `slack_group_handle` | Slack group handle without `@` (e.g., `team-payments`) | — |

Ask for all fields in a single prompt. Confirm defaults with user before proceeding.

---

## Workflow

### Phase 1: Collect Inputs

Prompt the user for the fields in the table above. Present defaults inline. Example prompt:

```
To create the base pod pipeline for <service>, I need a few details:

Required:
- service_name: [provided]
- slack_group_id: (e.g. S12345678)
- slack_group_handle: (e.g. team-payments, without the @)

Using defaults (confirm or override):
- github_repo_name: <service>
- namespace: <service>
- service_host: <service>.dev.razorpay.in
- slack_channel: tech_deployments
```

---

### Phase 2: Invoke Base Pod Validation

Run the [Base Pod Readiness](base-pod-readiness.md) subskill for the service.

**After validation completes, read the Structured Result Block** — specifically these fields:

```
replica_overrides: "<value>"      # e.g., "web_replicas=2,worker_replicas=2" or ""
kube_manifests_pr: "<value>"      # e.g., PR URL or "none"
overall: <READY|BLOCKED>
```

- If `overall = BLOCKED` → **abort**. The chart does not exist; direct user to run `/devstack Onboarding` first.
- If `overall = READY` → proceed with pipeline creation.

---

### Phase 3: Build `default_overrides` String

Construct the `default_overrides` value for the pipeline JSON:

**Base (always included)**:
```
devstack_label=base,ttl=forever,image=${ parameters.image_tag }
```

**If `replica_overrides` is non-empty** (from the validation result block), append:
```
,<replica_overrides>
```

**Examples**:

| replica_overrides from validation | Resulting default_overrides |
|---|---|
| `""` (empty) | `devstack_label=base,ttl=forever,image=${ parameters.image_tag }` |
| `"web_replicas=2"` | `devstack_label=base,ttl=forever,image=${ parameters.image_tag },web_replicas=2` |
| `"web_replicas=2,worker_replicas=2"` | `devstack_label=base,ttl=forever,image=${ parameters.image_tag },web_replicas=2,worker_replicas=2` |

---

### Phase 4: Build Pipeline JSON

Construct the pipeline JSON with all substitutions applied:

```json
{
    "application": "devstack-mum-rspl-<service_name>",
    "exclude": [],
    "id": "<generated-uuid>",
    "index": 0,
    "keepWaitingPipelines": false,
    "limitConcurrent": true,
    "locked": {
        "ui": true,
        "allowUnlockUi": false,
        "description": "Note: Pipeline Templates govern this pipeline. UI edits are blocked for consistency."
    },
    "name": "Deploy base pods on DevStack",
    "notifications": [],
    "parameterConfig": [],
    "stages": [],
    "type": "templatedPipeline",
    "schema": "v2",
    "template": {
        "artifactAccount": "front50ArtifactCredentials",
        "reference": "spinnaker://75cb9c2f-bb26-42e2-91f8-2b16c5445a3b:latest",
        "type": "front50/pipelineTemplate"
    },
    "triggers": [],
    "variables": {
        "application": "devstack-mum-rspl-<service_name>",
        "commit_txt_host": "<commit_txt_host>",
        "default_overrides": "<default_overrides_string>",
        "github_repo_name": "<github_repo_name>",
        "helm_chart_overrides_file": "razorpay/kube-manifests/contents/helmfile/charts/<service_name>/values.yaml",
        "helm_chart_path_prefix": "helmfile/<service_name>/<service_name>-1-",
        "helm_release_name_override": "<service_name>-base",
        "kube_manifests_bucket_names": "{\"Mumbai\":[{\"value\":\"rzp-kube-manifests\"}]}",
        "logs_host": "https://razorpay-non-prod.app.coralogix.in",
        "namespace": "<namespace>",
        "pre_deploy_notification_text": "<!subteam^<slack_group_id>|@<slack_group_handle>>\nDev-Serve <service_name> Deployment initiated by ${ trigger.user }\n Old Commit: ${ old_commit }\n New Commit : ${ parameters.image_tag }",
        "region": "Mumbai",
        "service_name": "<service_name>",
        "slack_channel": "<slack_channel>",
        "sleeve_host": "http://spin-sleeve.spinnaker:8080",
        "user_groups_mentions": "<!subteam^<slack_group_id>|@<slack_group_handle>>"
    }
}
```

**Substitution map**:

| Placeholder | Value |
|---|---|
| `<service_name>` | User-provided `service_name` |
| `<generated-uuid>` | Generate via `python3 -c "import uuid; print(uuid.uuid4())"` |
| `<commit_txt_host>` | User-provided `commit_txt_host` |
| `<default_overrides_string>` | String built in Phase 3 |
| `<github_repo_name>` | User-provided `github_repo_name` |
| `<namespace>` | User-provided `namespace` |
| `<slack_group_id>` | User-provided `slack_group_id` |
| `<slack_group_handle>` | User-provided `slack_group_handle` (without `@` — the `@` is added in the template) |
| `<slack_channel>` | User-provided `slack_channel` |

---

### Phase 5: Hand off to spinnaker-ops-assistant

Invoke the `/spinnaker-ops-assistant` skill and pass it:
- The Spinnaker application name: `devstack-mum-rspl-<service_name>`
- The target path in spinacode: `v3/<service_name>/dev-serve/mum-rspl/deploy-to-devserve.json`
- The full pipeline JSON built in Phase 4
- The instruction to raise a PR to `razorpay/spinacode`

The spinnaker-ops-assistant skill + Spinnaker MCP will handle creating the pipeline entry and raising the PR.

---

## Output Report

```
## ✅ Base Pod Pipeline Created: <service-name>

### What I Did
1. ✅ Collected inputs from user
2. ✅ Validated helm chart (base-pod-readiness)
3. ✅ Built default_overrides with replica overrides: <value or "none needed">
4. ✅ Built pipeline JSON
5. ✅ Handed off to spinnaker-ops-assistant — spinacode PR raised

### Pipeline Configuration
- Application: devstack-mum-rspl-<service>
- Namespace: <namespace>
- default_overrides: <value>

### Pull Requests
- spinacode PR: <URL from spinnaker-ops-assistant>
- kube-manifests PR: <URL or "none">

### Next Steps
- [ ] Get spinacode PR reviewed and merged
- [ ] If kube-manifests PR was raised, get it merged first
- [ ] After both PRs are merged, trigger the pipeline in Spinnaker:
      https://deploy.razorpay.com/#/applications/devstack-mum-rspl-<service>/executions
- [ ] Verify base pods are running: kubectl --context dev-serve get pods -n <namespace> -l devstack_label=base
```

---

## Error Cases

| Error | Cause | Resolution |
|---|---|---|
| `BLOCKED: chart not found` | Helm chart doesn't exist | Run `/devstack Onboarding` first |
| `spinnaker-ops-assistant not installed` | Skill missing | Install from agent-skills repo |
| `Spinnaker MCP not configured` | MCP server missing | Add MCP config and restart Claude |
| `Pipeline already exists` | Duplicate creation attempt | Review existing pipeline in Spinnaker UI |

---

## Related Subskills

- [Base Pod Readiness](base-pod-readiness.md) — Called by this subskill; validates chart and provides replica overrides
- [Onboarding](onboarding.md) — Creates the helm chart before this pipeline can be set up
- [Deployment](deployment.md) — For ephemeral pod deployments (not base pods)

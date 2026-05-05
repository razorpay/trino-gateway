# Devstack Skill Configuration

## Helmfile Directory

The skill needs to know where your `kube-manifests/helmfile` directory is. Three options:

### Option 1: Auto-Detection (Recommended)

The skill automatically searches for `kube-manifests/helmfile` from your repository root. No config needed if your project follows this structure. If the repo is not found locally, the skill will auto-clone it (see [Path Detection](path-detection.md)).

### Option 2: Update config.json

Edit `agent-skills/infrastructure/skills/devstack/config.json`:

```json
{
  "helmfile_directory": "/path/to/your/kube-manifests/helmfile",
  "auto_detect": true
}
```

### Option 3: Quick Setup Command

```bash
# From your repository root
echo "{\"helmfile_directory\": \"$(pwd)/kube-manifests/helmfile\", \"auto_detect\": true}" > agent-skills/infrastructure/skills/devstack/config.json
```

To verify: the skill reports the path it's using when you run any deployment command.

---

## PR Creation (for kube-manifests changes)

After onboarding a service or making chart changes, the skill attempts to raise a PR automatically. Priority order:

1. **GitHub MCP server** *(preferred)* — add to MCP config and set `GITHUB_PERSONAL_ACCESS_TOKEN`:
   ```json
   {"type": "stdio", "command": "npx", "args": ["-y", "@modelcontextprotocol/server-github"]}
   ```
2. **`gh` CLI** *(fallback)* — install from https://cli.github.com/ and run `gh auth login`
3. **Git only** — branch is pushed, manual PR link provided
4. **No git** — files created locally, all git operations skipped

---

## How It Works

### Autonomous Behavior

- **No permission needed** for diagnostic commands (logs, events, describe)
- **Auto-fixes simple issues** — resource limits, TTL, probes
- **Only asks for input** when truly ambiguous (e.g. which service to deploy)

### Clean Deployments (Default)

All deployments use delete-before-sync:
- Deletes existing deployment before sync (prevents stale resource conflicts)
- Re-runs hooks — DB/SQS/SNS configurators execute fresh
- Regenerates secrets with latest values

To skip: say "update existing" or set `delete_before_sync: false` in config.json.

### Output Format

All operations produce a structured report:
- **Status**: SUCCESS / FAILED / PARTIAL
- **What I Did**: numbered action list with ✅ / ❌
- **Issues Found**: root cause + evidence + fix applied or fix needed
- **Resources Deployed**: pods, services, access URLs
- **Next Steps**: actionable checklist
- **Debug Commands**: exact kubectl commands for manual verification

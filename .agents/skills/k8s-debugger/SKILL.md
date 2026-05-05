---
name: k8s-debugger
description: Comprehensive Kubernetes pod debugging and log analysis using Friday MCP server. Use when troubleshooting K8s pod issues including (1) Pod startup failures or crashes, (2) High restart counts or instability, (3) Performance problems or resource issues, (4) Application errors and log analysis, (5) Namespace-wide health checks, (6) Root cause analysis of K8s incidents.
metadata:
  mcp_servers:
    - friday-mcp
---

# K8s Debugger

Kubernetes pod debugging and analysis using the **Friday MCP server**. Use the MCP tools to execute kubectl commands and inspect pods, deployments, namespaces, and logs — then synthesize findings into a root cause and recommended fix.

## Approach

Always follow this sequence:

1. **Get pod status** — check phase, restart count, conditions, and recent events
2. **Get logs** — look at current and previous container logs for errors or crash signals
3. **Check resource constraints** — CPU/memory requests, limits, and actual usage
4. **Synthesize** — identify the root cause and propose a concrete remediation

Use the Friday MCP server's `execute` tool to run kubectl commands. All commands are read-only and safe.

## Common Scenarios

### Pod Won't Start
- Get pod description and events to check for image pull errors, insufficient resources, or node affinity issues
- Get init container logs if init containers are defined
- Check if the namespace has resource quotas that are exhausted

### High Restart Count / CrashLoopBackOff
- Get logs from the **previous** container instance (`--previous`) to see the crash reason
- Check liveness probe configuration — misconfigured probes are a frequent cause
- Look for OOMKilled in the pod status (memory limit hit)

### Performance / Latency Issues
- Check current CPU and memory usage against limits
- Look for throttling indicators in pod events
- Check HPA status if autoscaling is configured

### Namespace-Wide Health Check
- List all pods and flag any not in Running/Completed state
- Summarize restart counts across the namespace
- Check for pending pods and identify the scheduling blocker

## Using Friday MCP

The Friday MCP server provides an `execute` tool for running kubectl commands. All operations are read-only and safe.

**Required parameters:**
- `command`: "kubectl"
- `subcommand`: The kubectl subcommand (get, describe, logs, etc.)
- `parameters`: Additional arguments (pod name, namespace, flags)
- `intent`: Description of what you're trying to find
- `filter`: "grep", "head", "tail", or "none"
- `filter_value`: Value for the filter
- `account_alias`: AWS account (ask user if not specified)

**Example:**
```
command: "kubectl"
subcommand: "describe"
parameters: "pod my-pod -n production"
intent: "Get detailed pod information to diagnose crash"
filter: "grep"
filter_value: "Events"
account_alias: "prod"
```

See `references/friday-mcp-kubectl.md` for detailed examples and `references/debugging-workflows.md` for complete investigation workflows.

## What to Report

For every investigation, include:
- **What is happening** — observed symptoms (pod state, restart count, error messages)
- **Why it is happening** — root cause identified from logs/events
- **How to fix it** — specific remediation steps (config change, resource adjustment, image fix, etc.)

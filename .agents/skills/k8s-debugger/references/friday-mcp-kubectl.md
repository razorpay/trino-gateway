# Using Friday MCP for Kubernetes Debugging

## Overview

The Friday MCP server provides an `execute` tool that runs AWS CLI commands. While primarily for AWS operations, you can use it to execute `kubectl` commands for Kubernetes debugging when working with EKS clusters.

## Available Tool

### Friday MCP `execute` Tool

Execute read-only AWS CLI and kubectl commands for infrastructure inspection.

**Tool name:** Use the Friday MCP server's execute tool (the exact name depends on your MCP configuration - typically `mcp__friday-mcp__execute` or `mcp__remote-friday-mcp-server__execute`)

**Parameters:**
- `options`: AWS CLI options (e.g., "--region us-east-1 --output json")
- `command`: Main command (e.g., "eks", "kubectl")
- `subcommand`: Specific operation (e.g., "describe-cluster", "get pods")
- `parameters`: Additional parameters
- `intent`: Description of what you're trying to accomplish
- `filter`: Optional filter ("head", "tail", "grep", "none")
- `filter_value`: Value for the filter
- `account_alias`: AWS account to use

## Common Kubectl Operations

### Get Pod Status

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "get"
  parameters: "pods -n <namespace> -o wide"
  intent: "Check status of all pods in namespace <namespace>"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

### Describe Specific Pod

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "describe"
  parameters: "pod <pod-name> -n <namespace>"
  intent: "Get detailed information about pod <pod-name> including events and status"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

### Get Pod Logs

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "logs"
  parameters: "<pod-name> -n <namespace> --tail=100"
  intent: "Get last 100 lines of logs from pod <pod-name>"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

### Get Previous Pod Logs (for crashed containers)

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "logs"
  parameters: "<pod-name> -n <namespace> --previous --tail=200"
  intent: "Get logs from previous container instance to see crash reason"
  filter: "grep"
  filter_value: "error"
  account_alias: "prod"
```

### Check Pod Resource Usage

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "top"
  parameters: "pod <pod-name> -n <namespace>"
  intent: "Check CPU and memory usage for pod <pod-name>"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

### Get Pod Events

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "get"
  parameters: "events -n <namespace> --field-selector involvedObject.name=<pod-name>"
  intent: "Get all events related to pod <pod-name> to see scheduling and lifecycle issues"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

### List All Failing Pods

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "get"
  parameters: "pods -n <namespace> --field-selector=status.phase!=Running,status.phase!=Succeeded"
  intent: "Find all pods that are not running or completed in namespace <namespace>"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

### Check Deployment Status

```
Tool: Friday MCP execute
Parameters:
  command: "kubectl"
  subcommand: "get"
  parameters: "deployment <deployment-name> -n <namespace> -o yaml"
  intent: "Get deployment configuration and status for <deployment-name>"
  filter: "none"
  filter_value: ""
  account_alias: "prod"
```

## Debugging Workflows

### Workflow 1: Pod Won't Start

1. **Get pod status**
   ```
   kubectl get pod <pod-name> -n <namespace> -o wide
   ```

2. **Describe pod for events**
   ```
   kubectl describe pod <pod-name> -n <namespace>
   ```
   Filter: grep "Events" to focus on relevant section

3. **Check init container logs** (if applicable)
   ```
   kubectl logs <pod-name> -n <namespace> -c <init-container-name>
   ```

4. **Check namespace resource quotas**
   ```
   kubectl get resourcequota -n <namespace>
   ```

### Workflow 2: High Restart Count / CrashLoopBackOff

1. **Get pod status to see restart count**
   ```
   kubectl get pod <pod-name> -n <namespace>
   ```

2. **Get previous container logs**
   ```
   kubectl logs <pod-name> -n <namespace> --previous --tail=200
   ```
   Filter: grep "error\|fatal\|panic\|exception"

3. **Describe pod to check liveness probes**
   ```
   kubectl describe pod <pod-name> -n <namespace>
   ```
   Filter: grep "Liveness\|OOMKilled"

4. **Check resource limits**
   ```
   kubectl describe pod <pod-name> -n <namespace>
   ```
   Filter: grep "Limits\|Requests"

### Workflow 3: Performance / Latency Issues

1. **Check resource usage**
   ```
   kubectl top pod <pod-name> -n <namespace>
   ```

2. **Get full pod spec with limits**
   ```
   kubectl get pod <pod-name> -n <namespace> -o yaml
   ```
   Filter: grep "resources\|limits\|requests"

3. **Check HPA status** (if autoscaling enabled)
   ```
   kubectl get hpa -n <namespace>
   ```

4. **Look for throttling events**
   ```
   kubectl get events -n <namespace> --field-selector involvedObject.name=<pod-name>
   ```
   Filter: grep "throttl\|OOM\|evict"

### Workflow 4: Namespace-Wide Health Check

1. **List all pods with status**
   ```
   kubectl get pods -n <namespace> -o wide
   ```

2. **Count pods by phase**
   ```
   kubectl get pods -n <namespace> --no-headers
   ```
   Filter: grep to count Running, Pending, Failed, etc.

3. **Get all failing pods**
   ```
   kubectl get pods -n <namespace> --field-selector=status.phase!=Running,status.phase!=Succeeded
   ```

4. **Check recent events**
   ```
   kubectl get events -n <namespace> --sort-by='.lastTimestamp'
   ```
   Filter: tail 50 to get most recent

## Available AWS Accounts

When using the Friday MCP server, specify one of these account aliases:

- **ai-prod**: Account ID 485652168265, Region ap-south-1
- **ezetap-prod**: Account ID 192870797902, Region ap-south-1
- **perf-cell1**: Account ID 098596092361, Region ap-south-2
- **perf-cell2**: Account ID 585563982108, Region ap-south-2
- **prod**: Account ID 141592612890, Region ap-south-1
- **qubole**: Account ID 164294033593, Region ap-south-1
- **stage**: Account ID 101860328116, Region ap-south-1
- **upi-switch-apb-prod**: Account ID 459164390929, Region ap-south-1
- **upi-switch-axis-prod**: Account ID 533267024078, Region ap-south-1
- **upi-switch-axis-stage**: Account ID 600275855853, Region ap-south-1

**Note:** Always ask the user which account to use if not specified.

## Filter Usage Tips

### Using grep filter
When you want to highlight errors or specific patterns:
```
filter: "grep"
filter_value: "error\|warn\|fail"
```
Returns matching lines with 25 lines of context above and below.

### Using head/tail
When output is large and you want first/last N lines:
```
filter: "head"
filter_value: "50"  # First 50 lines
```

```
filter: "tail"
filter_value: "100"  # Last 100 lines
```

### No filter
For complete output (up to size limits):
```
filter: "none"
filter_value: ""
```

## Important Notes

1. **Read-only operations**: The Friday MCP server only allows read operations. No write, update, or delete commands.

2. **Account context**: Always specify the correct AWS account alias based on which cluster you're debugging.

3. **Timeouts**: Complex queries may timeout. If that happens, simplify the query or use filters to reduce output.

4. **Output limits**: Very large outputs may be summarized. Use filters to get specific data.

5. **Intent field**: Always provide a clear intent describing what you're trying to find. This helps create better summaries when output is large.

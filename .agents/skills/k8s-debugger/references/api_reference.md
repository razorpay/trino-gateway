# K8s Debugger - Friday MCP Commands Reference

## Quick Command Reference

Use the Friday MCP server's `execute` tool to run these kubectl commands.

### Pod Status Checks

**Check specific pod:**
```bash
kubectl get pod <pod-name> -n <namespace> -o wide
```

**Check all pods in namespace:**
```bash
kubectl get pods -n <namespace> -o wide
```

**Describe specific pod (includes events):**
```bash
kubectl describe pod <pod-name> -n <namespace>
```

### Log Analysis

**Get pod logs:**
```bash
kubectl logs <pod-name> -n <namespace> --tail=100
```

**Get previous container logs (for crashed pods):**
```bash
kubectl logs <pod-name> -n <namespace> --previous --tail=200
```

**Get logs with timestamps:**
```bash
kubectl logs <pod-name> -n <namespace> --timestamps --tail=100
```

### Comprehensive Debugging

**Full pod analysis (status + events + logs):**
1. Get pod status: `kubectl get pod <pod-name> -n <namespace> -o wide`
2. Describe pod: `kubectl describe pod <pod-name> -n <namespace>`
3. Get logs: `kubectl logs <pod-name> -n <namespace> --tail=200`
4. Get events: `kubectl get events -n <namespace> --field-selector involvedObject.name=<pod-name>`

**Namespace-wide issue analysis:**
1. List all pods: `kubectl get pods -n <namespace> -o wide`
2. Find failing pods: `kubectl get pods -n <namespace> --field-selector=status.phase!=Running,status.phase!=Succeeded`
3. Get recent events: `kubectl get events -n <namespace> --sort-by='.lastTimestamp' | tail -50`

## Common Use Cases

### 1. Pod Won't Start

**Investigation sequence:**
1. `kubectl get pod <pod-name> -n <namespace>` - Check current status
2. `kubectl describe pod <pod-name> -n <namespace>` - Look for image pull errors, scheduling issues
3. `kubectl logs <pod-name> -n <namespace>` - Check container logs
4. `kubectl get events -n <namespace> --field-selector involvedObject.name=<pod-name>` - Review events

**Common causes:** Image pull errors, resource constraints, invalid configuration

### 2. High Restart Count / CrashLoopBackOff

**Investigation sequence:**
1. `kubectl get pod <pod-name> -n <namespace>` - Check restart count
2. `kubectl logs <pod-name> -n <namespace> --previous` - Get logs from crashed container
3. `kubectl describe pod <pod-name> -n <namespace>` - Check liveness probes, resource limits
4. Look for OOMKilled status indicating memory limits hit

**Common causes:** Application crashes, failed liveness probes, OOM kills

### 3. Namespace Issues

**Investigation sequence:**
1. `kubectl get pods -n <namespace>` - List all pod statuses
2. `kubectl get pods -n <namespace> --field-selector=status.phase!=Running,status.phase!=Succeeded` - Find failing pods
3. `kubectl get events -n <namespace> --sort-by='.lastTimestamp'` - Check recent events
4. `kubectl top pods -n <namespace>` - Check resource usage

**Common causes:** Resource quotas, node issues, deployment problems

### 4. Performance Problems

**Investigation sequence:**
1. `kubectl top pod <pod-name> -n <namespace>` - Check CPU/memory usage
2. `kubectl describe pod <pod-name> -n <namespace>` - Review resource limits/requests
3. `kubectl get hpa -n <namespace>` - Check autoscaling status
4. `kubectl logs <pod-name> -n <namespace>` - Look for performance-related errors

**Common causes:** Resource constraints, CPU throttling, memory pressure

## Friday MCP Command Output Formats

### kubectl get pods
```
NAME                     READY   STATUS    RESTARTS   AGE
api-gateway-abc123-xyz   1/1     Running   0          2d
payment-service-def456   0/1     Pending   5          1h
```

### kubectl describe pod
```
Name:         api-gateway-abc123-xyz
Namespace:    production
Status:       Running
IP:           10.0.1.23
Conditions:
  Ready       True
Events:
  Normal  Scheduled  2d   Successfully assigned pod
  Normal  Pulled     2d   Container image pulled
  Normal  Started    2d   Started container
```

### kubectl logs
```
2024-02-19T10:23:45Z INFO Starting application
2024-02-19T10:23:46Z ERROR Failed to connect to database
2024-02-19T10:23:47Z FATAL Application crashed
```

### kubectl get events
```
LAST SEEN   TYPE      REASON       OBJECT                MESSAGE
1m          Warning   BackOff      pod/payment-service   Back-off restarting failed container
5m          Warning   Failed       pod/payment-service   Error: ImagePullBackOff
```

## Error Handling

### MCP Connection Issues
If Friday MCP server is unavailable:
```
ERROR: Cannot connect to Friday MCP server
Check MCP server configuration and connectivity
```

### Account Access Issues
```
ERROR: Access denied to AWS account
Verify account_alias and permissions
```

### Command Timeouts
```
TIMEOUT: Command took too long to complete
Try simplifying the query or using filters to reduce output size
```

### Forbidden Operations
```
FORBIDDEN: Write operations are not allowed
Friday MCP server only supports read-only commands
```

## Advanced Usage Patterns

### Resource Analysis
```bash
# Check pod resource requests vs limits
kubectl get pods -n <namespace> -o json | \
  jq '.items[] | {name: .metadata.name, requests: .spec.containers[].resources.requests, limits: .spec.containers[].resources.limits}'

# Get pods sorted by restart count
kubectl get pods -n <namespace> --sort-by='.status.containerStatuses[0].restartCount'

# Find pods with high memory usage
kubectl top pods -n <namespace> --sort-by=memory
```

### Log Analysis with Filters
```bash
# Find errors in logs (use Friday MCP grep filter)
kubectl logs <pod-name> -n <namespace> --tail=500
# Filter: "grep"
# Filter value: "ERROR\|FATAL\|exception"

# Get logs from specific time range
kubectl logs <pod-name> -n <namespace> --since=1h --timestamps

# Get logs from all containers in a pod
kubectl logs <pod-name> -n <namespace> --all-containers
```

### Batch Namespace Checks
```bash
# Check multiple namespaces
for ns in production staging development; do
  echo "Checking $ns:"
  kubectl get pods -n $ns --field-selector=status.phase!=Running
done

# Get resource usage across namespaces
kubectl top pods -A --sort-by=memory | head -20
```
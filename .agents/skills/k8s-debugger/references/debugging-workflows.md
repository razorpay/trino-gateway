# K8s Debugging Workflows with Friday MCP

This document outlines proven debugging workflows using the Friday MCP server for Kubernetes troubleshooting.

## Workflow Selection Guide

### Single Pod Investigation
Use when investigating a specific pod that's failing or behaving unexpectedly.

**Commands needed:**
- `kubectl get pod` - Current status
- `kubectl describe pod` - Events and configuration
- `kubectl logs` - Application logs
- `kubectl get events` - Related events

### Namespace Health Check
Use when checking overall health of a namespace or finding all problematic pods.

**Commands needed:**
- `kubectl get pods` - List all pods
- `kubectl get pods --field-selector` - Filter by status
- `kubectl get events` - Recent namespace events
- `kubectl top pods` - Resource usage

### Performance Analysis
Use when investigating latency, high resource usage, or performance degradation.

**Commands needed:**
- `kubectl top pod` - Current resource usage
- `kubectl describe pod` - Resource limits and requests
- `kubectl get hpa` - Autoscaling status
- `kubectl logs` - Performance-related logs

### Deployment Investigation
Use when a deployment fails or causes issues.

**Commands needed:**
- `kubectl get deployment` - Deployment status
- `kubectl describe deployment` - Deployment events
- `kubectl rollout status` - Rollout progress
- `kubectl rollout history` - Previous versions

## Multi-Step Debugging Workflows

### Workflow 1: Pod Won't Start

**Step 1: Check pod status**
```bash
kubectl get pod <pod-name> -n <namespace> -o wide
```
Look for: Phase (Pending/Failed), Node assignment

**Step 2: Get pod description**
```bash
kubectl describe pod <pod-name> -n <namespace>
```
Look for: Events section (image pull errors, scheduling failures, resource constraints)

**Step 3: Check pod events**
```bash
kubectl get events -n <namespace> --field-selector involvedObject.name=<pod-name> --sort-by='.lastTimestamp'
```
Look for: FailedScheduling, FailedMount, ImagePullBackOff

**Step 4: Check logs (if container started)**
```bash
kubectl logs <pod-name> -n <namespace>
```
Look for: Application startup errors

**Common causes:**
- Image doesn't exist or registry inaccessible
- Insufficient resources (CPU/memory)
- Node selector/affinity preventing scheduling
- Missing secrets or ConfigMaps
- Volume mount failures

### Workflow 2: CrashLoopBackOff / High Restart Count

**Step 1: Check restart count**
```bash
kubectl get pod <pod-name> -n <namespace>
```
Note: RESTARTS column shows count

**Step 2: Get previous container logs**
```bash
kubectl logs <pod-name> -n <namespace> --previous --tail=200
```
Look for: Error messages, stack traces, fatal errors

**Step 3: Check liveness/readiness probes**
```bash
kubectl describe pod <pod-name> -n <namespace>
```
Look for: Liveness/Readiness probe configuration and failures

**Step 4: Check resource limits**
```bash
kubectl get pod <pod-name> -n <namespace> -o yaml
```
Look for: OOMKilled status, memory/CPU limits

**Step 5: Check events for patterns**
```bash
kubectl get events -n <namespace> --field-selector involvedObject.name=<pod-name>
```
Look for: Killing, OOMKilling, Unhealthy

**Common causes:**
- Application crashes on startup
- Misconfigured liveness probes (too aggressive)
- OOM kills (memory limit too low)
- Missing environment variables or configuration
- Database/dependency connection failures

### Workflow 3: Performance / Latency Issues

**Step 1: Check current resource usage**
```bash
kubectl top pod <pod-name> -n <namespace>
```
Compare CPU/Memory usage against limits

**Step 2: Get resource limits and requests**
```bash
kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.spec.containers[*].resources}'
```
Look for: Under-provisioned limits, missing requests

**Step 3: Check for throttling events**
```bash
kubectl get events -n <namespace> --field-selector involvedObject.name=<pod-name>
```
Look for: Evicted, OOMKilling, BackOff

**Step 4: Check HPA status (if applicable)**
```bash
kubectl get hpa -n <namespace>
```
Look for: Current replicas vs desired, scaling behavior

**Step 5: Analyze application logs**
```bash
kubectl logs <pod-name> -n <namespace> --tail=500
```
Look for: Slow query warnings, timeout errors, performance degradation messages

**Common causes:**
- CPU throttling (limits too low)
- Memory pressure
- Slow external dependencies
- Inefficient application code
- Network latency

### Workflow 4: Namespace-Wide Health Check

**Step 1: List all pods**
```bash
kubectl get pods -n <namespace> -o wide
```
Identify: Non-Running pods, high restart counts

**Step 2: Get failing pods**
```bash
kubectl get pods -n <namespace> --field-selector=status.phase!=Running,status.phase!=Succeeded
```
Focus: Pods needing attention

**Step 3: Check recent events**
```bash
kubectl get events -n <namespace> --sort-by='.lastTimestamp' | tail -50
```
Look for: Patterns, repeated errors

**Step 4: Check resource usage**
```bash
kubectl top pods -n <namespace> --sort-by=memory
```
Identify: Resource-constrained pods

**Step 5: Get namespace resource quotas**
```bash
kubectl get resourcequota -n <namespace>
```
Check: Quota limits and usage

**Common issues:**
- Namespace resource quota exhausted
- Systemic deployment issues
- Node-level problems affecting multiple pods
- ConfigMap/Secret changes breaking pods

### Workflow 5: Deployment Rollout Issues

**Step 1: Check deployment status**
```bash
kubectl get deployment <deployment-name> -n <namespace>
```
Look for: Ready replicas vs desired

**Step 2: Check rollout status**
```bash
kubectl rollout status deployment/<deployment-name> -n <namespace>
```
Look for: Stuck rollout, waiting for replicas

**Step 3: Get deployment events**
```bash
kubectl describe deployment <deployment-name> -n <namespace>
```
Look for: ReplicaSet creation, scaling events, errors

**Step 4: Check ReplicaSet status**
```bash
kubectl get rs -n <namespace> -l app=<app-label>
```
Identify: Old vs new ReplicaSets

**Step 5: Check pod status in new ReplicaSet**
```bash
kubectl get pods -n <namespace> -l pod-template-hash=<new-rs-hash>
```
Then describe/logs failing pods

**Common causes:**
- Image doesn't exist for new version
- Configuration errors in new deployment
- Readiness probes failing
- Resource constraints preventing scale-up

## Common Error Patterns

### ImagePullBackOff
1. `kubectl describe pod` → Check image name
2. Verify: Registry URL, image tag exists, pull secrets configured
3. Check: Network connectivity to registry

### Pending (FailedScheduling)
1. `kubectl describe pod` → Check scheduling events
2. Verify: Node capacity, resource requests, node selectors
3. Check: Taints/tolerations, pod affinity rules

### OOMKilled
1. `kubectl describe pod` → Check OOMKilled status
2. `kubectl get pod -o yaml` → Review memory limits
3. `kubectl logs --previous` → Check memory usage patterns
4. Action: Increase memory limits or optimize application

### Error: ErrImagePull
1. `kubectl describe pod` → Get exact error message
2. Verify: Image exists in registry
3. Check: Pull secrets, registry authentication
4. Verify: Network policies allow registry access

## Best Practices

### Query Construction
- Always specify namespace explicitly with `-n`
- Use `--field-selector` to filter results
- Add `--sort-by` for ordered output
- Use `-o wide` or `-o yaml` for detailed info

### Information Gathering
1. Start broad: Get overview (`kubectl get pods`)
2. Focus: Describe specific resources
3. Deep dive: Logs and events
4. Correlate: Compare before/after states

### Using Friday MCP Filters
- **grep filter**: Use for error hunting in logs (pattern: `"ERROR\|FATAL\|exception"`)
- **tail filter**: Get last N lines of large outputs
- **head filter**: Get first N lines for quick checks
- **none filter**: Get complete output (within limits)

### Synthesis
After gathering data:
1. Identify the symptom (what's failing)
2. Determine the root cause (why it's failing)
3. Propose remediation (how to fix it)
4. Include prevention steps (avoid future recurrence)

## Example Complete Investigation

**Scenario:** Pod `payment-api-abc123` in namespace `production` keeps restarting

**Step-by-step:**

1. **Check current status**
   ```bash
   kubectl get pod payment-api-abc123 -n production
   ```
   Result: CrashLoopBackOff, 15 restarts

2. **Get previous logs**
   ```bash
   kubectl logs payment-api-abc123 -n production --previous --tail=100
   ```
   Friday MCP filter: grep, value: "ERROR\|FATAL"
   Result: "FATAL: Cannot connect to database: connection refused"

3. **Check pod configuration**
   ```bash
   kubectl describe pod payment-api-abc123 -n production
   ```
   Result: Environment variable `DB_HOST` = "localhost"

4. **Check service configuration**
   ```bash
   kubectl get service database-service -n production
   ```
   Result: Service exists with correct ClusterIP

**Root cause:** Pod configured with `DB_HOST=localhost` instead of `database-service`

**Remediation:** Update deployment to set `DB_HOST=database-service`

**Prevention:** Add configuration validation in deployment pipeline

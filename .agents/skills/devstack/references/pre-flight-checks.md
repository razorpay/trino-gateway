# Pre-Flight Checks Reference

Checks to run BEFORE `helmfile sync` to prevent hard deployment failures.

## 1. devstack_label Validation

**Check**: Does the `devstack_label` in helmfile.yaml match what the user requested?

```bash
grep "devstack_label:" helmfile.yaml | head -3
```

Expected location in file:
```yaml
environments:
  default:
    values:
      - devstack_label: <label>   # ← Must match user's requested label
```

**Fix**: Update with Edit tool if mismatched.

---

## 2. ServiceAccount Conflict Detection

**Check**: Are any ServiceAccounts in target namespaces owned by a different Helm release?

```bash
# Run for each namespace being deployed to
for ns in <space-separated list of namespaces being deployed to>; do
  echo "=== $ns ===";
  kubectl get serviceaccount -n $ns -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.meta\.helm\.sh/release-name}{"\n"}{end}' 2>/dev/null | grep -v "^$"
done
```

**Conflict detected if**: A SA's release-name annotation doesn't match the release you're about to create.

**Fix options** (in order of preference):
1. Check for `create_sa` flag in chart: `grep "create_sa" charts/<service>/values.yaml`
   - If found, add `- create_sa: true` to helmfile values
2. If no flag exists, check if SA is safe to delete (no other pods using it), then delete

---

## 3. Missing ServiceAccount Detection

**Check**: Does a required SA exist when `create_sa` defaults to false?

```bash
# Check if chart has create_sa pattern
grep "create_sa" charts/<service>/values.yaml charts/<service>/templates/*.yaml 2>/dev/null
```

If chart has `create_sa: false` default AND the SA doesn't exist in the namespace:
```bash
kubectl get serviceaccount <sa-name> -n <namespace> 2>&1
```

**Fix**: Add `- create_sa: true` to service values in helmfile.yaml.

---

## 4. Chart Default vs Base Deployment Resource Drift

**Check**: Are the chart's default resource limits lower than what the base deployment runs?

```bash
# Get base deployment resources for a service
kubectl get deployment -n <namespace> -l devstack_label=base \
  -o jsonpath='{.items[0].spec.template.spec.containers[0].resources}' | python3 -m json.tool
```

Compare with chart defaults:
```bash
grep -E "limits_memory|requests_memory|limits_cpu|requests_cpu" charts/<service>/values.yaml
```

**Common drift patterns**:
- Chart default 500Mi memory, base runs 600Mi → OOMKill on startup
- Chart default 400m CPU limit → throttling (all CPU limits should be removed)

**Fix**: Override in helmfile values:
```yaml
- <service>_limits_memory: <base_deployment_value>
```

---

## 5. CPU Limits Audit

**Check**: Do any charts set CPU limits on main service pods?

```bash
# Check all charts being deployed
for svc in <space-separated list of services being deployed>; do
  echo "=== $svc ===";
  grep -n "limits_cpu\|cpu:" charts/$svc/templates/*.yaml 2>/dev/null | grep -v "requests\|#\|init\|sec-\|secret\|cloner" | head -5;
done
```

**Fix**: Remove CPU limits via values override or template patch (see auto-fix-strategies.md).

---

## 6. Init Container Dependency Check

**Check**: Will any pods have init containers that wait for services not being deployed?

```bash
# After helmfile template, check for init containers
helmfile template 2>/dev/null | grep -B2 -A10 "initContainers"
```

**Problematic pattern**: `wait-for-<service>` init containers where `<service>` is not in your deployment.

**Fix**: Ensure `link_services` in helmfile values does NOT include the dependency service. Verify with `helmfile template 2>/dev/null | grep "wait-for"`.

---

## Quick Pre-Flight Script

Run all checks at once before deploying:

```bash
LABEL="<your-label>"
SERVICES="<space-separated list of services being deployed>"

echo "=== 1. devstack_label check ==="
grep "devstack_label:" helmfile.yaml | head -2

echo "=== 2. ServiceAccount conflicts ==="
for svc in $SERVICES; do
  ns=$(grep -A2 "name: $svc-" helmfile.yaml | grep namespace | awk '{print $2}' | head -1)
  [ -z "$ns" ] && continue
  kubectl get serviceaccount -n $ns -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.meta\.helm\.sh/release-name}{"\n"}{end}' 2>/dev/null | grep -v "^$\|$LABEL" && echo "  ^^^ CONFLICT in $ns" || true
done

echo "=== 3. Init container check ==="
helmfile template 2>/dev/null | grep "wait-for" | sort -u

echo "=== 4. CPU limits ==="
for svc in $SERVICES; do
  grep -n "cpu:" charts/$svc/templates/*.yaml 2>/dev/null | grep -v "requests\|#\|init\|sec\|cloner\|cache" | head -3
done
```

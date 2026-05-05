# Infrastructure & Deployment Knowledge (Persistent Context)

This is reusable platform knowledge. Do NOT rediscover this on every run.

## Razorpay Deployment Architecture

### Helmfile is Centralized

- Helmfiles are **NOT** in service repos. They live in the `kube-manifests` repo.
- Path: `kube-manifests/helmfile/charts/<service>/`
- Main helmfile: `kube-manifests/helmfile/helmfile.yaml`
- The `kube-manifests` repo may use sparse checkout — run `git sparse-checkout add helmfile/charts/<service> helmfile` to get chart files.

### kube-manifests Structure

```
kube-manifests/
  helmfile/
    helmfile.yaml                    # Main helmfile (all services, mostly commented out)
    charts/<service>/                # Per-service helm chart
      Chart.yaml
      values.yaml                    # Default values (image tags, replicas, env)
      templates/
        deployment.yaml              # Deployment specs
        svc.yaml                     # Service + devstack_label routing
        preview-url.yaml             # IngressRoute for devstack access
        secret_cloner.yaml           # Clones base secret for custom pods
        sqs-configmap.yaml           # SQS queue config (localstack)
        sqs-configurator.yaml        # Creates SQS queues on localstack
  templates/<service>/               # Standard stage/prod helm charts (Spinnaker)
  stage/<service>/values.yaml        # Stage-specific overrides
```

### Deploying a Custom Pod

1. **Edit** `helmfile/helmfile.yaml` — uncomment the service block, set image SHA
2. **Image tag format**: Just the commit SHA (no `service:` prefix for helmfile)
3. **Run**: `helmfile --state-values-set devstack_label=<label> --state-values-set ttl=8h sync`
4. **Access**: `https://<service>-<label>.dev.razorpay.in` OR `https://<service>.dev.razorpay.in` with header `rzpctx-dev-serve-user: <label>`

### Image Tag Format

| Context | Format | Example |
|---------|--------|---------|
| helmfile values.yaml | `batch:<sha>` | `sqs_image_tag: batch:29d60dec...` |
| helmfile release block | Just SHA | `image: 29d60dec...` |
| Docker registry | `c.rzp.io/razorpay/<service>:<sha>` | `c.rzp.io/razorpay/batch:29d60dec...` |

### Devstack Label Patterns

| Label | Purpose | Properties |
|-------|---------|------------|
| `base` | Shared persistent pod | Protected, backup labels, dedicated nodes |
| `<custom>` (e.g., `harsh-e2e-2412`) | Ephemeral test pod | TTL-based auto-cleanup, generic worker nodes |

### Resources Created per Devstack Deploy

- Deployment: `<service>-web-<label>`, `<service>-sqs-<label>`
- Service: `<service>-<label>` (ClusterIP, port 80)
- IngressRoute: `<service>-<label>.dev.razorpay.in`
- Secret: `<service>-<label>` (cloned from base)
- SQS Queues: `devstack-<service>-<label>` (on localstack)

## CI/CD Rules

### CI Trigger Conditions

- **Batch CI**: Triggers on `pull_request` to `master` and `push` to `master` only
- Branch-to-branch PRs do NOT trigger build workflows
- **CRITICAL**: Merge conflicts silently prevent CI from triggering
- Always check: `gh pr view <URL> --json mergeable,mergeStateStatus`

### CI Not Triggering Checklist

1. Check merge conflicts FIRST
2. Check workflow trigger config (`.github/workflows/`)
3. Check branch filters (`branches: [master]`)
4. Check concurrency groups (previous run might be blocking)
5. Try: resolve conflicts → push → verify

### Docker Registry Migration

- Old: `harbor.razorpay.com/razorpay/` (may return 401)
- New: `c.rzp.io/razorpay/`
- If Dockerfile uses old registry, update to new one

## Safety Constraints (STRICT)

### Never Scale Down Base Pods

Base pods (`-base` suffix) are shared infrastructure:
- Used by ALL developers on devstack
- Scaling down breaks other workflows
- NEVER modify base deployments, services, or secrets
- Only interact with your own custom-labeled resources

### Never Modify Base Secrets

- Base secrets contain shared credentials
- Custom pod secrets are cloned from base during deploy
- Override values via `kubectl set env` on your custom deployment only

## Cross-Namespace Service Discovery

Services in different K8s namespaces require FQDN:

```
http://<service>-<label>.<namespace>.svc.cluster.local:<port>
```

Common ports:
- Go services: typically 8000 or 8080 (check with `netstat -tlnp` inside pod)
- Java/Spring: typically 8080
- Nginx reverse proxy: 80 (may not proxy all paths)
- Edge sidecar: 8000

### Scrooge Specifics

- Service `scrooge-base` in namespace `scrooge`
- Nginx on port 80 (doesn't proxy internal API paths)
- Go app on port 8000 (serves all API routes including `/v1/internal/*`)
- Edge service: `scrooge-edge-base` (ExternalName → `edge-base.edge.svc.cluster.local:8000`)

## Environment Drift Awareness

Always compare devstack vs prod configs for:

| Check | Where |
|-------|-------|
| Base paths | `application-devstack.properties` vs `application-prod.properties` |
| Feature flags | `splitz.mock` (true in dev, false in prod) |
| DNS/hostnames | Short names vs FQDN |
| Ports | Direct app port vs reverse proxy port |
| Auth credentials | Env var references vs hardcoded defaults |

## Experiment/Feature Flag Testing on Devstack

### Priority Order in SplitzClientService

1. `X-Splitz-Override` header (ThreadLocal, HTTP request only — does NOT propagate to async/SQS)
2. `SPLITZ_MOCK=true` + `MOCK_SPLITZ_<EXPERIMENT>=true` env vars (requires pod restart)
3. Redis cache: `SET splitz:<experiment>:id:<entity_id> true EX 3600` (works across all workers)
4. Actual Splitz API call (production path)

### For SQS/Async Workers

`X-Splitz-Override` header does NOT work — use env vars or Redis cache approach.

```bash
# Set env vars on deployment
kubectl set env deployment/<deploy> -n <ns> SPLITZ_MOCK=true MOCK_SPLITZ_<EXPERIMENT>=true

# OR set Redis cache (works for all pods sharing Redis)
kubectl exec deployment/<deploy> -n <ns> -- sh -c \
  'echo "SET splitz:<experiment>:id:<entity_id> true EX 3600" | nc -w3 $REDIS_HOST $REDIS_PORT'
```

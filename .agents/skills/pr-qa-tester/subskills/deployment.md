# Phases 3-4: Service Enumeration & Deployment

**IMPORTANT**: Load [references/infra-deployment.md](../references/infra-deployment.md) for persistent platform knowledge before proceeding. Do NOT rediscover helmfile architecture, kube-manifests structure, or CI/CD patterns.

## Pre-Deployment Checks (MANDATORY)

1. **Merge conflicts**: `gh pr view <URL> --json mergeable` — conflicts silently block CI image builds
2. **CI image built**: `gh pr checks <URL>` — look for build jobs (e.g., "Build Batch")
3. **Helmfile exists**: Check `kube-manifests/helmfile/charts/<service>/` — helmfiles are NOT in service repos
4. **Cross-namespace DNS**: If service calls other services, verify FQDN resolution (not short names)
5. **Config drift**: Compare `application-devstack.properties` vs `application-prod.properties` for base paths, ports, auth

## Service Enumeration

For each impacted service, determine the deployment strategy:

| Service Role | Image Source | Deployment Method |
|-------------|-------------|-------------------|
| PR service (primary) | PR branch commit SHA | CI build → helmfile sync OR devspace dev |
| Downstream dependency | Latest master | Base pod (already running) OR helmfile sync |
| Infrastructure (DB, cache, queue) | N/A | Ephemeral resources via helm hooks |

### Check Existing Custom Pods

```bash
# Check if user already has a custom pod for the service
kubectl get pods -n <namespace> -l devstack_label=<user_label> --no-headers
```

- **Pod exists**: Use `devspace dev` for hot-reload (preferred)
- **No pod**: Need first-time setup via CI/CD → helmfile sync

### Get PR Commit SHA (Image Tag)

```bash
gh pr view <PR_URL> --json headRefOid -q '.headRefOid'
# Or use short SHA:
gh pr view <PR_URL> --json headRefOid -q '.headRefOid[:7]'
```

### Verify CI Image is Built

```bash
# Check GitHub Actions build status
gh pr checks <PR_URL> --json name,state,conclusion | jq '.[] | select(.name | contains("build"))'
```

If build is not complete, inform user and wait or prompt to trigger.

## Deployment

### Delegate to /devstack Skill

Use the existing `/devstack` skill for actual deployment:

```
/devstack
Deploy <service> with commit <sha> and devstack label <label>
```

For multi-service deployment:
```
/devstack
Deploy <svc1> with <sha1>, <svc2> with <sha2>, <svc3> with existing
```

### First-Time Pod Setup (CI/CD Path)

If no custom pod exists:

1. Ensure the PR branch CI has built the image (check GitHub Actions)
2. Invoke `/devstack` with the commit SHA
3. The devstack skill handles helmfile.yaml editing, validation, and `helmfile sync`
4. Wait for pods to become ready

### Hot-Reload Path (Devspace)

If custom pod already exists:

1. Clone the PR branch locally:
   ```bash
   gh pr checkout <PR_URL>
   ```
2. Invoke devspace:
   ```
   /devstack
   Setup devspace for <service> with label <label>
   ```
3. The devstack skill handles devspace.yaml configuration and `devspace dev`
4. Code changes auto-sync to the running pod

### Devspace Fallback

If devspace fails (hot-reload not supported by the repo):

Ask user:
- **Option A**: Fix devspace configuration (inspect errors, update devspace.yaml)
- **Option B**: Fall back to CI/CD — push changes, wait for image build, redeploy via helmfile

### Post-Deployment Verification

After deployment, verify all services are healthy:

```bash
kubectl get pods -n <namespace> -l devstack_label=<label> -o wide
kubectl logs -n <namespace> -l name=<service>-<label> --tail=20
```

Check for:
- All pods in `Running` state
- No `CrashLoopBackOff` or `ImagePullBackOff`
- Health check endpoints responding

If pods are unhealthy, delegate to `/devstack` debugging:
```
/devstack
Debug pods in namespace <namespace> with label <label>
```

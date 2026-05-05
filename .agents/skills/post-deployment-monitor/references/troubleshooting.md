# Troubleshooting Guide

Common issues encountered during post-deployment monitoring and their solutions.

## Image Tag Parsing Issues

### Issue: Cannot extract commit SHA from image tag

**Symptoms:**
- `parse_image_tag.py` returns error
- Image tag format not recognized

**Common causes:**
1. Non-standard image tag format
2. Image tag doesn't contain commit SHA
3. Commit SHA format unexpected (not hex)

**Solutions:**

```bash
# Check the actual image tag format
kubectl get deployment <name> -n <namespace> -o jsonpath='{.spec.template.spec.containers[0].image}'

# Common formats:
# registry.com/app:v1.2.3-abc123def  ✅ Supported
# registry.com/app:abc123def         ✅ Supported
# registry.com/app:latest            ❌ No commit SHA
# registry.com/app:v1.2.3            ❌ No commit SHA

# Manual extraction if script fails
IMAGE="registry.com/app:v1.2.3-abc123def"
COMMIT=$(echo $IMAGE | grep -oE '[a-f0-9]{7,40}$' | tail -1)
```

**Workaround:**
- Ask user for commit SHAs directly
- Check CI/CD pipeline for commit information
- Use deployment annotations if they contain commit info

### Issue: ReplicaSet not found for previous image

**Symptoms:**
- Cannot get previous deployment image
- No ReplicaSet history

**Solutions:**

```bash
# Check all ReplicaSets for the deployment
kubectl get rs -n <namespace> -l app=<deployment> --sort-by=.metadata.creationTimestamp

# If only one RS exists, deployment may be new
# Alternative: Check deployment rollout history
kubectl rollout history deployment/<name> -n <namespace>

# Get specific revision
kubectl rollout history deployment/<name> -n <namespace> --revision=<N>
```

**Workaround:**
- First deployment: Compare against empty baseline
- Use git log to find previous release tag

## Repository Access Issues

### Issue: Cannot find repository for deployment

**Symptoms:**
- Unknown which GitHub repo corresponds to deployment
- Repository name doesn't match deployment name

**Solutions:**

1. Check deployment labels/annotations:
```bash
kubectl get deployment <name> -n <namespace> -o yaml | grep -E "(labels|annotations)" -A 5
```

2. Common naming patterns:
   - Deployment name = repo name
   - Namespace = repo name
   - Label `app.kubernetes.io/name`
   - Annotation `repo.url`

3. Ask user for repository name

### Issue: Commit not found in repository

**Symptoms:**
- GitHub API returns 404 for commit SHA
- Commit exists locally but not on remote

**Solutions:**

```bash
# Verify commit exists
git log --all --grep=<commit-sha>

# Check if commit is on remote
git branch -r --contains <commit-sha>

# Possible causes:
# - Commit from different branch not yet merged
# - Commit from fork not in main repo
# - Short SHA conflict (use full SHA)
```

**Workaround:**
- Use `git diff` locally if repo is checked out
- Ask user to push commits to remote

## Skill Loading Issues

### Issue: No Claude skills found in repository

**Symptoms:**
- `.claude/` directory doesn't exist
- No skill files in expected location

**Solutions:**

```bash
# Check for skills
ls -la .claude/skills/

# Common locations:
# .claude/skills/<service>/SKILL.md
# .claude/skills/<service>-skill/SKILL.md
# skills/<service>/  (alternative)
```

**Workaround:**
- Skip skill-based analysis
- Use generic impact analysis on diff
- Manually identify affected flows from code

### Issue: Skill files exist but no flows documented

**Symptoms:**
- Skills loaded but no flows.md found
- Flows exist but don't document routes

**Solutions:**

1. Check skill structure:
```bash
find .claude/skills/ -name "flows.md"
find .claude/skills/ -name "*.md" | grep -i flow
```

2. Use alternative documentation:
   - API specs (OpenAPI/Swagger)
   - README files
   - Integration documentation
   - Code comments

**Workaround:**
- Extract routes from code directly
- Use `scripts/extract_routes.py` on all .md files
- Manually trace affected routes from diff

## Route Extraction Issues

### Issue: No routes extracted from flows

**Symptoms:**
- `extract_routes.py` returns empty
- Routes exist but not in expected format

**Solutions:**

1. Check route format in flows:
```markdown
# Supported formats:
POST /v1/offers           ✅
Route: /offers/create     ✅
Endpoint: GET /v1/data    ✅
`DELETE /v1/items/{id}`   ✅

# Unsupported formats:
/offers (no method)       ⚠️ Extracted without method
offers/create (no /)      ❌
```

2. Manual extraction:
```bash
# Grep for routes
grep -rE "(GET|POST|PUT|DELETE|PATCH)\s+/[a-z]" .claude/skills/ --include="*.md"
```

**Workaround:**
- Manually list routes from API documentation
- Check OpenAPI spec if available
- Ask user for affected routes

## Grafana Query Issues

### Issue: No data returned for route

**Symptoms:**
- Grafana query returns empty result
- Metrics exist but route label doesn't match

**Solutions:**

1. Check actual metric labels:
```promql
# List all available labels
http_requests_total

# Check what route values exist
group(http_requests_total) by (route)
```

2. Route label variations:
```promql
# Different label names:
http_requests_total{route="/v1/offers"}      # Standard
http_requests_total{path="/v1/offers"}       # Alternative
http_requests_total{endpoint="/v1/offers"}   # Alternative
http_requests_total{uri="/v1/offers"}        # Alternative

# Route normalization:
/v1/offers/{id}         # Template
/v1/offers/123          # Actual value
/v1/offers/:id          # Alternative template
```

**Solutions:**
- Ask user for correct label name
- Check Grafana dashboards for examples
- Use `list_prometheus_label_names` to discover labels

### Issue: Baseline comparison shows huge change

**Symptoms:**
- 1000%+ increase in errors or latency
- Baseline period had no traffic

**Causes:**
- Baseline period (24h ago) was different:
  - Deployment was down
  - Route didn't exist yet
  - Weekend vs weekday traffic
  - Off-peak vs peak hours

**Solutions:**

```promql
# Check if baseline had traffic
sum(rate(http_requests_total{route="<route>"}[5m] offset 24h))
# If returns 0, baseline is invalid

# Use alternative baseline:
# - 1 week ago (same day of week)
offset 168h

# - 1 hour ago (recent stable period)
offset 1h

# - Absolute threshold instead of relative
sum(rate(http_requests_total{status=~"5..", route="<route>"}[5m])) > 0.1
```

**Recommendation:**
- Always check baseline has traffic first
- Use absolute thresholds for new routes
- Compare multiple time windows

## Kubernetes Access Issues

### Issue: Cannot access cluster

**Symptoms:**
- `kubectl_execute` fails
- Authentication error
- Cluster not found

**Solutions:**

```bash
# Verify cluster access
kubectl config get-contexts

# Check current context
kubectl config current-context

# Use KubeJit if available
# Request temporary access to cluster
request_cluster_access:
  cluster: <cluster-name>
  duration: 3600
```

**Workaround:**
- Ask user to provide kubectl output
- Use alternative observability tools
- Check deployment status in Spinnaker

### Issue: Namespace not found

**Symptoms:**
- Deployment not found in namespace
- Namespace doesn't exist

**Solutions:**

```bash
# List all namespaces
kubectl get namespaces

# Search for deployment across namespaces
kubectl get deployment --all-namespaces | grep <name>
```

## Analysis Interpretation Issues

### Issue: Unclear impact from diff

**Symptoms:**
- Cannot determine affected flows from code changes
- Changes are complex or indirect

**Solutions:**

1. Focus on key indicators:
   - Changed API handlers → Affected route
   - Changed business logic → Affected flow
   - Changed database queries → Performance impact
   - Changed external calls → Integration impact

2. Use grep to find flow references:
```bash
# Find which flows mention changed files
for file in $(git diff --name-only <old>..<new>); do
  grep -r "$(basename $file)" .claude/skills/ --include="*.md"
done
```

3. Ask user for guidance:
   - "Which flows are affected by changes in <file>?"
   - "What are the critical routes for this service?"

**Workaround:**
- Monitor all documented routes
- Focus on routes with highest traffic
- Use broad monitoring if uncertain

## General Best Practices

### When Things Go Wrong

1. **Collect more context:**
   - Full error messages
   - Actual kubectl output
   - Repository structure
   - Available Grafana dashboards

2. **Simplify the problem:**
   - Skip skill-based analysis if skills unavailable
   - Use manual route list if extraction fails
   - Monitor key metrics even if can't compare to baseline

3. **Communicate clearly:**
   - Explain what's missing
   - Suggest alternatives
   - Ask user for specific information

4. **Provide partial results:**
   - Report what you can determine
   - Mark uncertain items clearly
   - Suggest manual verification steps

### Recovery Steps

If monitoring completely fails:

1. **Manual checklist:**
```markdown
Provide user with manual monitoring checklist:

□ Check deployment rollout status
□ Verify all pods are running
□ Check pod logs for errors
□ Review Grafana dashboard for service
□ Check error rates in last 30 minutes
□ Compare latency to baseline
□ Review recent alerts
□ Check Sift for anomalies
```

2. **Fallback to basic health checks:**
   - Pod status
   - Restart count
   - Resource usage
   - Recent alerts

3. **Request user action:**
   - Manual Grafana review
   - Check service-specific dashboards
   - Review logs in Coralogix
   - Consult on-call runbook

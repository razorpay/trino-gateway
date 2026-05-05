# Troubleshooting Guide - Alert Coverage Analyzer

This guide covers common issues when running the alert coverage analyzer workflow.

## PR Creation Failures

### Step 1: Diagnose the Failure

Check which operation failed:
```bash
# Verify branch push status
cd <application-repo>
git branch -r | grep add-metrics-<service-name>

cd <alert-rules-path>
git branch -r | grep add-alerts-<service-name>
```

### Step 2: Common Failure Scenarios

#### Scenario A: Authentication Failure

```bash
# Check authentication status
gh auth status

# If expired or invalid:
gh auth refresh

# If completely broken:
gh auth logout
gh auth login
```

**Error message:** `HTTP 401: Bad credentials` or `authentication required`

**Fix:** Re-authenticate with `gh auth login`

#### Scenario B: Branch Already Exists Remotely

```bash
# Check if branch exists
gh api repos/razorpay/<repo>/branches/add-metrics-<service-name>

# If exists, delete and retry:
git push origin --delete add-metrics-<service-name>
git push -u origin add-metrics-<service-name>
```

**Error message:** `remote ref already exists` or `rejected - non-fast-forward`

**Fix:** Delete remote branch and re-push, or use a different branch name

#### Scenario C: No Permission to Create PRs

```bash
# Check repository access
gh api repos/razorpay/<repo>/collaborators/<username>/permission

# Expected: "permission": "write" or "admin"
```

**Error message:** `Resource not accessible by integration` or `403 Forbidden`

**Fix:** Contact repository admin to grant write access

#### Scenario D: PR Already Exists

```bash
# Check existing PRs
gh pr list --head add-metrics-<service-name>
```

**Error message:** `A pull request already exists`

**Fix:** View existing PR with `gh pr view <number>` and update it, or close and create new

#### Scenario E: Network/Connectivity Issues

```bash
# Test GitHub connectivity
curl -I https://api.github.com

# Retry PR creation with verbose output
gh pr create --title "..." --body "..." --debug
```

**Error message:** `dial tcp: lookup api.github.com` or timeout errors

**Fix:** Check internet connection, retry after network stabilizes

### Step 3: Retry Logic

If push succeeded but PR creation failed, retry PR creation:

```bash
# Application repo PR
cd <application-repo>
gh pr create --title "Add missing metrics for <service-name>" \
  --body "$(cat <<'EOF'
## Summary
- Added <count> new metrics

## Metrics Added
...
EOF
)"

# Alert-rules repo PR
cd <alert-rules-path>
gh pr create --title "Add alert rules for <service-name>" \
  --body "$(cat <<'EOF'
## Summary
- Added <count> alert rules

## Alerts Added
...
EOF
)"
```

### Step 4: Manual Fallback

If automated PR creation continues to fail after debugging, provide manual instructions:

```
=== MANUAL PR CREATION REQUIRED ===

Application Repo:
1. Branch: add-metrics-<service-name>
2. URL: https://github.com/razorpay/<repo>/compare/add-metrics-<service-name>
3. Click "Create pull request"
4. Copy PR template below:

   Title: Add missing metrics for <service-name>

   Body:
   ## Summary
   - Added <count> new metrics to track critical business flows

   ## Metrics Added
   - `<metric1>` - <description>
   - `<metric2>` - <description>

   ## Business Impact
   - Enables monitoring of <flow>
   - Prevents <incident type>

   🤖 Generated with Claude Code

Alert Rules Repo:
1. Branch: add-alerts-<service-name>
2. URL: https://github.com/razorpay/alert-rules/compare/add-alerts-<service-name>
3. Click "Create pull request"
4. Copy PR template below:

   Title: Add alert rules for <service-name>

   Body:
   ## Summary
   - Added <count> alert rules for new <service-name> metrics

   ## Alerts Added
   - <alert1> - Threshold: <value>
   - <alert2> - Threshold: <value>

   ## Related PR
   Application repo: <link-to-metrics-pr>

   🤖 Generated with Claude Code
```

**Note:** Both branches are already pushed and ready for PR creation

## Service Identification Issues

### Service Name Unknown

If automatic detection fails:

```bash
# Check common locations:
# 1. Go modules
cat go.mod | grep "module"

# 2. Package.json
cat package.json | jq -r '.name'

# 3. Metric definitions
grep -r "Namespace:" app/metric/ internal/metrics/
```

If still unknown, ask user directly for service name.

### Not a Razorpay Repository

**Error:** `This is not a Razorpay repository`

**Cause:** Git remote is not `razorpay/*`

**Fix:** This skill only works with Razorpay repositories. Navigate to a Razorpay repo.

## Alert-Rules Repository Issues

### Repository Not Found Locally

If user-provided path fails verification:

```bash
cd <user-provided-path>
git remote -v | grep "razorpay/alert-rules"
```

**Fix:** Clone to `/tmp`:
```bash
cd /tmp && gh repo clone razorpay/alert-rules
```

### Missing Alert Files

If no alert files found for service:

```bash
find <alert-rules-path> -name "*<service-name>*" -type f
```

**Possible causes:**
1. Service name mismatch (e.g., `payment-links` vs `payment_links`)
2. Alerts in different directory structure
3. No alerts exist yet (first-time setup)

**Fix:** Try alternative naming patterns or confirm this is first-time alert setup.

## Metric Scanning Issues

### No Metrics Found

If metric scan returns empty:

```bash
# Check alternative locations
find . -name "*metric*.go" -o -name "*metrics*.js"

# Search for Prometheus patterns
grep -r "prometheus.New" .
grep -r "NewCounterVec\|NewHistogramVec\|NewGaugeVec" .
```

**Possible causes:**
1. Metrics in non-standard location
2. Different metric library used
3. New service with no metrics yet

**Fix:** Ask user where metrics are defined, or confirm this is a new service.

## GitHub CLI Issues

### gh Not Installed

```bash
which gh
# Output: gh not found
```

**Fix:**
```bash
# macOS
brew install gh

# Linux
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install gh
```

### gh Not Authenticated

```bash
gh auth status
# Output: You are not logged into any GitHub hosts
```

**Fix:**
```bash
gh auth login
# Follow interactive prompts
```

## Repo Skill Issues

### Repo Skill Not Found

If `.agents/skills/` is empty:

```bash
ls -la .agents/skills/
# Output: No such file or directory
```

**Fix:** This is optional. Proceed without repo skill (lower accuracy expected).

**Recommendation:** Create repo skill for better analysis:
```bash
# Use swe-agent or similar tool to create repo skill
npx skills add razorpay/agent-skills --skill swe-agent
```

### Observability Section Missing

If repo skill exists but has no observability docs:

```bash
ls .agents/skills/<service>-skill/modules/technical/observability/
# Output: No such file or directory
```

**Fix:** Proceed without observability section. Consider adding it to repo skill for future audits.

## Common Warnings

### High Cardinality Labels Detected

**Warning:** Metric uses unbounded label (`merchant_id`, `terminal_id`, etc.)

**Impact:** Will cause Prometheus memory exhaustion

**Fix:** Remove high-cardinality label. Use logs for per-entity debugging instead.

**Example:**
```diff
- // WRONG
- prometheus.NewCounterVec(
-     prometheus.CounterOpts{Name: "payments_total"},
-     []string{"merchant_id", "status"},  // ❌ Unbounded
- )

+ // CORRECT
+ prometheus.NewCounterVec(
+     prometheus.CounterOpts{Name: "payments_total"},
+     []string{"status", "gateway"},  // ✅ Bounded
+ )
```

### External Service Alerts

**Warning:** Recommended alert for external service latency/errors

**Clarification:** External services own their alerts. Only add **metrics** for dependency tracking, not alerts.

**Exception:** Add latency metrics (p99, p95, p50) and 5XX error metrics for downstream service calls.

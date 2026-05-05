# Deployment Subskill

Autonomous deployment workflow for helmfile-based services.

## Purpose

Handles end-to-end deployment of services including pre-validation, helmfile sync, and initial health checks.

## ⚠️ Kubernetes Tool Priority

For all `kubectl` operations, try **Friday Kubernetes MCP** (`kubectl_execute`) first, then fall back to local `kubectl --context dev-serve` if Friday MCP is unavailable or the specific operation fails. See [SKILL.md](../SKILL.md) for full rules.

## CRITICAL DEPLOYMENT RULES

**⚠️ IMPORTANT CHANGES - READ CAREFULLY**:

1. **`--kube-context dev-serve` is MANDATORY on every helmfile and helm command**:
   - ❌ WRONG: `helmfile -f helmfile.yaml sync`
   - ✅ CORRECT: `helmfile --kube-context dev-serve -f helmfile.yaml sync`
   - This applies to ALL helmfile operations: `template`, `delete`, `sync`, and all `helm` commands
   - Omitting it is unsafe — if the active kubectl context is wrong, you deploy to the wrong cluster silently
   - There are NO exceptions to this rule

2. **NO Selector Flags**: NEVER use `-l` selector flags in helmfile commands
   - ❌ WRONG: `helmfile --kube-context dev-serve -f helmfile.yaml -l name=service-label sync`
   - ✅ CORRECT: `helmfile --kube-context dev-serve -f helmfile.yaml sync`

2. **Service Selection via Uncommenting**:
   - To deploy specific service(s): UNCOMMENT only those services, COMMENT OUT all others
   - To deploy all services: UNCOMMENT all desired services
   - Helmfile deploys ALL uncommented services automatically

3. **Auto-Uncomment Required**:
   - ALWAYS check if requested services are commented
   - AUTOMATICALLY uncomment ALL services mentioned in deployment request
   - Report uncommenting action to user

4. **Image Validation Required**:
   - ALWAYS validate images via Harbor API before deployment (unless `skip_image_validation: true`)
   - Check images exist in registry
   - Verify linux/amd64 architecture support
   - BLOCK deployment if images are invalid or missing
   - WARN if images don't support amd64 (ask user to check CI workflow)

## When to Use

- Deploying new services
- Updating existing deployments
- Re-deploying after configuration changes
- Rolling out new image versions

## ⚠️ CRITICAL: "Base Commit" Phrase Recognition

The following phrases ALL mean the same thing: **fetch the image commit currently running in the base deployment pod via kubectl** — NOT the latest commit on git master.

> "latest base commit" · "base commit" · "base deployment commit" · "use base" · "same as base" · "what's running in base" · "base pod commit" · "current base" · "with base image"

**When ANY of these phrases appear in the user's request → NEVER use git/master/HEAD. ALWAYS run the kubectl command below to get the actual running commit from the base pod.**

```bash
kubectl --context dev-serve get deployment -n <namespace> -l devstack_label=base \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.image}{"\n"}{end}{end}'
```

Extract the commit hash from the image tag (e.g. `c.rzp.io/razorpay/pg-router:api_38a86d11` → commit is `38a86d11`) and use that as the image value in helmfile.yaml.

## Workflow

### Pre-Deployment Checklist

Before deploying, you MUST complete ALL these steps IN ORDER. Each step MUST be reported to the user with its status:

1. ✅ **Read helmfile.yaml once** into memory — reuse for all subsequent analysis (no repeated file reads)
2. ✅ **Locate service** in helmfile.yaml → Report: "Found service at line X"
3. ✅ **Uncomment service** if it's commented out → Report: "Uncommented service" or "Already uncommented"
4. ✅ **Validate/update `devstack_label`** in helmfile environments block → Report: "Label is X" or "Updated label to X"
5. ✅ **Update image field** — determine source from user's words (see image source table below) → Report: "Image set to X" or "Keeping existing image X" or "Fetched base commit X from running base pod"
6. ✅ **Comment out** all other services not being deployed → Report: "Commented out N other services"
7. ✅ **Pre-flight checks** (run in PARALLEL): namespace SA conflicts + base commit fetches for all services → Report results
8. ✅ **Render templates once** via `helmfile --kube-context dev-serve template` — save output, reuse for BOTH validation (step 8) AND image extraction (step 9). Do NOT run `helmfile template` twice.
9. ✅ **VALIDATE IMAGES VIA HARBOR API** using images extracted from step 8 output → Report: "All N images validated: X passed, Y failed, Z warnings"
10. ⛔ **DEPLOYMENT GATE** — Only proceed if step 9 passed OR user explicitly approved warnings/failures
11. ✅ **Check if release exists** before deleting — skip delete if no existing release
12. ✅ **Deploy** using `helmfile --kube-context dev-serve -f helmfile.yaml sync` (no selectors)
13. ✅ **Fetch notes** for all deployed releases in PARALLEL

> ⛔ **NEVER skip from step 8 to step 12.** Step 9 (image validation) is MANDATORY before deployment.
> ⚡ **Run helmfile template ONCE only** — cache the output and use it for both template validation and image extraction.

### Phase 1: Pre-Deployment Validation

#### 1. Read helmfile.yaml Once (Optimisation)

**Read the entire helmfile.yaml into memory at the start using the Read tool.** All subsequent analysis (finding services, checking label, checking images, listing uncommented services) should work from this in-memory content. Do NOT use separate `grep` shell calls for each check — that re-reads the file each time and wastes time.

```bash
# ONE read at the start covers all of: finding service, checking label, 
# checking image field, listing commented/uncommented services
```

#### 2. Change to Helmfile Directory

**Important**: Read the helmfile directory path from `../config.json`:
```bash
# The skill will read config.json to get the helmfile_directory path
# Example: cd /Users/parag.dudeja/Documents/Work/rzp-repos/harbor-action-tracking/kube-manifests/helmfile
cd <helmfile_directory from config.json>
```

**Auto-Detection Workflow**:
If `auto_detect: true` in config.json:
1. Try the configured `helmfile_directory` path
2. If not found, try fallback paths relative to repo root:
   - `kube-manifests/helmfile`
   - `helmfile`
   - `../kube-manifests/helmfile`
3. Report the path being used to the user
4. If none found, ask user to configure the path

**User-Specified Directory**:
If the user explicitly points to a different helmfile directory (e.g., "use charts from /path/to/helmfile"), use that path directly and update config.json accordingly.

#### 2. Locate Service in helmfile.yaml

Use Grep to find the service release entry:
```bash
grep -n "name: <service>-{{ .Values.devstack_label }}" helmfile.yaml
```

**CRITICAL - Service Comment Management**:
- For ALL services mentioned in the deployment request:
  - If commented (lines starting with #) → AUTOMATICALLY uncomment it
  - Verify the service entry includes all required fields
  - Check if image tag is specified
- **CRITICAL**: For ALL services NOT mentioned in deployment request:
  - If uncommented → AUTOMATICALLY comment it out
  - This prevents deploying unintended services
  - Only services explicitly mentioned should be uncommented

**Example**:
```
User request: "deploy pg-router and asv"

Actions:
1. Find pg-router → uncomment if needed ✓
2. Find asv → uncomment if needed ✓
3. Find ALL other uncommented services → comment them out ✓
4. Result: ONLY pg-router and asv are uncommented
```

#### 2.5. Validate devstack_label in helmfile.yaml

**CRITICAL**: Before deploying, verify the `devstack_label` in the helmfile environments block matches the user's requested label.

```bash
grep "devstack_label:" helmfile.yaml | head -5
```

The environments block looks like:
```yaml
environments:
  default:
    values:
      - devstack_label: parag   # ← THIS must match the requested label
```

**If it doesn't match**: Update it using the Edit tool before proceeding.

**Why this matters**: A wrong label deploys to the wrong namespace/release name, silently deploying with an unintended identity.

**CRITICAL - Image Update**:
- **ALWAYS check the `image:` field** in the service's values section in helmfile.yaml
- The image value is a **commit ID** (git commit hash)
- Update workflow:
  1. If user specifies image/commit: **UPDATE** the `image:` field to the new commit ID
  2. If user says "existing", "current", or "keep": **DO NOT UPDATE** - keep current image value
  3. If no image specified: **DO NOT UPDATE** - keep current image value
- Image field location in helmfile.yaml:
  ```yaml
  - name: service-{{ .Values.devstack_label }}
    namespace: service
    chart: ./charts/service
    values:
      - image: <COMMIT_ID_HERE>  # ← UPDATE THIS if new image specified
      - devstack_label: {{ .Values.devstack_label }}
      - ttl: {{ .Values.ttl }}
  ```

**Example**:
```yaml
# Before (commented)
# - name: payment-service-{{ .Values.devstack_label }}
#   namespace: payment-service
#   chart: ./charts/payment-service

# After (uncommented automatically)
- name: payment-service-{{ .Values.devstack_label }}
  namespace: payment-service
  chart: ./charts/payment-service
  values:
    - image: abc123def
    - devstack_label: {{ .Values.devstack_label }}
    - ttl: {{ .Values.ttl }}
```

#### 3. Update Image Values (If Specified)

**CRITICAL STEP - Update Commit IDs in helmfile.yaml**:

For each service in the deployment request:
1. **Read the current image value** from helmfile.yaml
2. **Determine image source** using this lookup table — match in order, stop at first match:

   | User said | Image source | Action |
   |---|---|---|
   | Any "base" phrase (see list above) | Running base pod | `kubectl get deployment -n <ns> -l devstack_label=base` → extract commit from image tag → UPDATE |
   | A specific commit hash (e.g. `abc123`) | User-provided | UPDATE to that commit |
   | "existing" / "current" / "keep" | Current helmfile.yaml value | NO UPDATE |
   | Nothing about image | Current helmfile.yaml value | NO UPDATE |

   **"base" phrases ALWAYS take priority and ALWAYS require a kubectl lookup. Never substitute git/master for "base".**

3. **Update the image field** if needed using Edit tool

**Example Updates**:

```yaml
# User request: "deploy pg-router with image abc123def"
# BEFORE:
- name: pg-router-{{ .Values.devstack_label }}
  values:
    - image: old456xyz  # ← Current image

# AFTER (UPDATE):
- name: pg-router-{{ .Values.devstack_label }}
  values:
    - image: abc123def  # ← Updated to new commit ID
```

```yaml
# User request: "deploy asv with existing image"
# BEFORE:
- name: account-service-{{ .Values.devstack_label }}
  values:
    - image: current789  # ← Keep this

# AFTER (NO CHANGE):
- name: account-service-{{ .Values.devstack_label }}
  values:
    - image: current789  # ← Unchanged, using existing
```

**Important**:
- Image values are **git commit hashes** (SHA-1, 40 characters)
- Shortened commit IDs (7-8 characters) are also valid
- **Always verify** the commit exists before deploying

#### Deploying with Base Deployment Commits

**What is a base deployment?**
A base deployment is a permanent, always-on deployment in each service's namespace with `devstack_label=base`. It is managed by Spinnaker and runs the latest staging/production commit. It acts as the shared fallback that all devstack users share — traffic routes to a user's personal deployment first, then falls back to base if none exists. Base deployments use `ttl=forever` and are never cleaned up.

When the user asks to deploy "with the commit running on their base deployment", they mean: use the same image tag currently running on the base pod as your deployment's image. This pattern ensures your personal deployment is on the same version as the shared environment.

Use this pattern to fetch base commits:

```bash
# Get the base deployment image for each service
# Base deployment = deployment with devstack_label=base in the service's namespace
kubectl get deployment -n <namespace> -l devstack_label=base \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.image}{"\n"}{end}{end}'
```

The main API image commit is extracted from the primary deployment (not workers). For example:
- `api-web-base` → `c.rzp.io/razorpay/api:<commit>` → use `<commit>` as the image value
- `pg-router-base` → `c.rzp.io/razorpay/pg-router:api_<commit>` → use `<commit>` as the image value

Run this for ALL services simultaneously (parallel tool calls in a single message) before editing helmfile.yaml.

#### 4. Comment Out Non-Target Services

**CRITICAL**: After uncommenting your target services and updating images, ensure ALL other services in helmfile.yaml are commented out. This prevents deploying unintended services.

```bash
# Verify only target services are uncommented
grep -n "^- " helmfile.yaml | grep "name:"
```

Only services explicitly mentioned in the deployment request should appear uncommented.

#### 4.5. Pre-Flight Checks (Run ALL in PARALLEL)

**Run all pre-flight checks simultaneously** — SA conflict checks across all target namespaces AND base commit fetches can all be issued in a single parallel batch. Do not run them sequentially.

**Purpose**: Detect resource ownership conflicts BEFORE deployment that would cause hard failures during `helmfile sync`.

**Step 1 — Check for conflicting ServiceAccounts**:
Some charts create ServiceAccounts (e.g., `reporting`). These are often shared across deployments in the same namespace. If a SA already exists owned by a different Helm release, the install will fail.

```bash
# For each service being deployed, check if a SA exists in its namespace
kubectl get serviceaccount -n <namespace> -o json | \
  python3 -c "import json,sys; sas=json.load(sys.stdin)['items']; \
  [print(f'SA: {s[\"metadata\"][\"name\"]} owned by {s[\"metadata\"].get(\"annotations\",{}).get(\"meta.helm.sh/release-name\",\"unknown\")}') for s in sas if 'meta.helm.sh/release-name' in s['metadata'].get('annotations',{})]"
```

**If a conflict is found**:
- The SA is owned by a DIFFERENT release → **WARN the user** before taking action
- ⚠️ **DANGER**: Deleting a SA affects ALL pods in that namespace that use it (including other users' deployments and base pods)
- Check if the chart has a `create_sa` flag: `grep -r "create_sa" charts/<service>/values.yaml`
- If `create_sa` flag exists and defaults to `false`, set `create_sa: true` in helmfile values — this creates the SA as part of THIS release without deleting the existing one

**Preferred fix** (avoids SA deletion):
```yaml
# In helmfile.yaml, add to the service's values:
- create_sa: true
```

**Only delete the SA if**: no other releases/pods in the namespace depend on it. Verify first:
```bash
# Check what's using the SA
kubectl get pods -n <namespace> -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.serviceAccountName}{"\n"}{end}' | grep <sa-name>
```

**Step 2 — Check for `create_sa` pattern in charts**:
If a chart has `create_sa: false` as default and the SA doesn't exist in the namespace, pods will fail with:
```
Error creating: pods is forbidden: error looking up service account <namespace>/<sa-name>: serviceaccount "<sa-name>" not found
```
Always check for this pattern when deploying a service for the first time or after a SA was deleted:
```bash
grep -r "create_sa" charts/<service>/values.yaml
```
If found and `create_sa: false`, add `- create_sa: true` to the service's helmfile values.

#### 5. Validate Chart Configuration

Read and validate these files:

**charts/<service-name>/values.yaml**:
- Check for required fields (see [../references/config-checklist.md](../references/config-checklist.md))
- Validate resource limits exist
- Verify TTL and devstack_label placeholders

**charts/<service-name>/templates/deployment.yaml**:
- Check for liveness/readiness probes
- Validate resource requests/limits
- Verify nodeSelector and labels
- Check DNS policy configuration
- **CRITICAL — nodeSelector must render exactly ONE key**: If the template has a conditional block (`{{ if eq .Values.devstack_label "base" }}`), verify that only one branch sets a nodeSelector key. Multiple nodeSelector keys cause pods to be stuck Pending indefinitely because Kubernetes requires ALL selectors to match.

**Automatic Fixes Applied**:
- Missing resource limits → Add defaults
- Missing TTL annotation → Add janitor/ttl
- Missing devstack_label → Add to labels
- Missing DNS policy → Add ClusterFirst

**CPU Limits Check**:
CPU limits cause throttling even when CPU is available on the node. Check if the chart sets CPU limits on main service pods:
```bash
grep -n "limits_cpu\|cpu:" charts/<service>/templates/deployment.yaml | grep -v "requests\|#\|init" | head -10
```

If CPU limits exist via a values variable (e.g., `cpu: {{ .Values.web_limits_cpu }}`):
- Override to empty in helmfile: `- web_limits_cpu: ""`
- Make the template conditional first: `{{- if .Values.web_limits_cpu }}cpu: {{ .Values.web_limits_cpu }}{{- end }}`

If CPU limits are hardcoded in the template, patch after deployment:
```bash
kubectl patch deployment <name> -n <namespace> --type='json' \
  -p='[{"op":"remove","path":"/spec/template/spec/containers/0/resources/limits/cpu"}]'
```

**Memory Limits vs Base Deployment**:
The chart's default memory limit may be lower than what the base deployment runs. Check:
```bash
# Get base deployment's actual memory limit
kubectl get deployment -n <namespace> -l devstack_label=base \
  -o jsonpath='{.items[0].spec.template.spec.containers[0].resources.limits.memory}'
```
If the chart default (from `values.yaml`) is lower, override in helmfile:
```yaml
- web_limits_memory: <base_deployment_value>
```

#### 6. Run Template Validation AND Extract Images (Single Command)

> ⚡ **Run `helmfile template` ONCE and reuse the output for BOTH steps 6 and 7.** Never run it twice.

```bash
# Run once, capture output
TEMPLATE_OUTPUT=$(helmfile --kube-context dev-serve -f helmfile.yaml template 2>&1)

# Use $TEMPLATE_OUTPUT for:
# (a) Checking for template errors (step 6)
# (b) Extracting images for Harbor validation (step 7)
```

**Check for template errors** in `$TEMPLATE_OUTPUT`:
- If output contains `Error:` or `error:` or exit code non-zero → attempt one automatic fix (add missing values, fix syntax)
- Re-run `helmfile template` ONCE after the fix
- If still errors after the second attempt → report to user and **STOP** (do not proceed to deployment)
- Maximum 2 render attempts total. If both fail, manual intervention is needed.

**If no errors** → proceed immediately to step 7 using the same `$TEMPLATE_OUTPUT`.

#### 7. Image Validation (Pre-Deployment Check)

> ⛔ **THIS STEP IS MANDATORY — Do NOT skip to Phase 2 (Deployment).**
> If you skip this step, deployments may fail with ImagePullBackOff.
> The ONLY way to skip is if `skip_image_validation: true` in config.json.
> **You MUST report the validation results to the user before proceeding.**

**CRITICAL**: Validate all container images exist in Harbor registry and support required architecture.

**Skip Validation**: Only if `skip_image_validation: true` in config.json

**Workflow**:

1. **Extract ALL Images from cached template output** (do NOT re-run `helmfile template`):
   ```bash
   # Extract from $TEMPLATE_OUTPUT captured in step 6
   # This pattern captures images at any indentation (main containers, initContainers, sidecars)
   echo "$TEMPLATE_OUTPUT" | grep -E '^\s+image:\s+' | sed 's/.*image:\s*//' | sed 's/[[:space:]]*$//' | grep -v '^#' | grep "c.rzp.io" | sort -u
   ```

   **CRITICAL**: This extracts:
   - Main service images (api, worker, etc.)
   - All worker images (notification_worker, payment_worker, etc.)
   - Init container images
   - Sidecar container images
   - Migration job images
   - **ALL images** used by the deployment

2. **Build Complete Image List**:
   - Extract all unique container image references
   - Remove quotes and clean up formatting
   - Format: `c.rzp.io/razorpay/service:tag` or `c.rzp.io/razorpay/service:commit-id`
   - **Include ALL containers** - typically 10-20 images per service
   - Example for pg-router: api, 10+ worker images, migration images

3. **Call Harbor Image Validation API**:
   ```bash
   # Build JSON array with ALL extracted images
   curl 'https://harbor-image-checker.dev.razorpay.in/check-images' \
     -H 'Content-Type: application/json' \
     -d '{
       "images": [
         "c.rzp.io/razorpay/pg-router:api_commit-abc123",
         "c.rzp.io/razorpay/pg-router:notification_worker_commit-abc123",
         "c.rzp.io/razorpay/pg-router:payment_worker_commit-abc123",
         "c.rzp.io/razorpay/pg-router:ledger_worker_commit-abc123",
         "c.rzp.io/razorpay/asv:api-commit-def456",
         "c.rzp.io/razorpay/asv:worker-commit-def456"
         ... (include ALL images from template output)
       ]
     }'
   ```

   **IMPORTANT**:
   - Validate **ALL** images extracted in step 1
   - Don't just validate the first 1-2 images
   - Typical deployments have 10-20+ images
   - Missing validation will cause ImagePullBackOff during deployment

4. **Analyze Response**:
   - Check `valid_count`, `invalid_count`, `skipped_count`
   - For each image in results:
     - `valid: false` → **BLOCK deployment** - invalid image
     - `exists: false` → **BLOCK deployment** - image not found
     - Check `architectures` array for `linux/amd64` → **WARN if missing**

5. **Validation Decision Table** (apply deterministically — no exceptions):

   | valid | exists | linux/amd64 present | Decision |
   |---|---|---|---|
   | true | true | ✅ yes | ✅ PASS — proceed |
   | true | true | ❌ no | ❌ BLOCK — amd64 missing, deployment will fail on devstack nodes |
   | true | false | any | ❌ BLOCK — image not in registry |
   | false | any | any | ❌ BLOCK — image invalid |

   **There is no "warn and proceed" for missing amd64.** Devstack nodes are amd64-only. An image without `linux/amd64` will always fail to schedule.

**Actions Based on Results**:

- **All images PASS** → Report "All N images validated ✅" and proceed to deployment gate
- **Any BLOCK — missing amd64** → Report "Image `{image}` exists but has no linux/amd64 support. Fix CI to build for amd64 (`--platform linux/amd64,linux/arm64`)." Stop.
- **Any BLOCK — not found** → Report "Image `{image}` not found. Verify the commit ID and that the CI build completed." Stop.
- **Any BLOCK — invalid** → Report "Image `{image}` failed validation: `{error}`." Stop.

**Configuration**:

To skip image validation, update `config.json`:
```json
{
  "skip_image_validation": true
}
```

**Error Handling**:
- If API call fails (network error, timeout): **WARN** but allow deployment — report the warning to user
- If API returns 500/error: **WARN** but allow deployment — report the warning to user
- Only **BLOCK** if API successfully returns invalid results

### ⛔ Pre-Deployment Gate (MANDATORY CHECKPOINT)

**DO NOT proceed to Phase 2 (Deployment) until you have confirmed ALL of the following:**

- [ ] Template rendering passed (step 6)
- [ ] Image validation completed (step 7) — results reported to user
- [ ] No FAIL results from Harbor API, OR Harbor API was unreachable (warned user)

**Report to user before proceeding:**
```
## Pre-Deployment Gate
✅ Templates rendered successfully
✅ Image validation: X/Y images passed (or ⚠️ Harbor API unreachable — proceeding with warning)
→ Proceeding to deployment...
```

**If image validation was skipped without `skip_image_validation: true` in config.json, GO BACK and complete step 7 now.**

---

### Phase 2: Deployment

#### 8. Clean Previous Deployment (Check First, Then Delete)

**IMPORTANT**: By default, delete the previous deployment before syncing — but **check if the release exists first** to avoid wasting time on an unnecessary delete command.

```bash
# Check if release exists before attempting delete (saves 5-15s on first deploys)
# Use anchored grep to avoid partial name matches (e.g. "api" matching "api-worker")
helm --kube-context dev-serve list -n <namespace> 2>/dev/null | grep -q "^<release-name>[[:space:]]" \
  && RELEASE_EXISTS=true || RELEASE_EXISTS=false

# If exists → run delete. If not → skip delete entirely.
```

**CRITICAL - Deploy Without Selector Flag**:
- Deploy using `helmfile --kube-context dev-serve -f helmfile.yaml` WITHOUT the `-l` selector flag
- This ensures all uncommented services in helmfile.yaml are deployed
- Only uncommented services will be deployed

```bash
# Only run if release exists:
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
```

**Why delete first?**
- ✅ **Clean slate**: Ensures no stale resources from previous deployments
- ✅ **Fresh configuration**: All configs applied from scratch
- ✅ **Prevents conflicts**: Avoids issues with changed resource types
- ✅ **Hook re-execution**: DB/SQS/SNS configurators run fresh
- ✅ **Secret updates**: Secrets regenerated with latest values

**Skip delete only if**:
- Release does not exist (first-time deploy) — skip automatically
- User explicitly says "deploy without deleting" or "keep existing deployment"
- Configuration has `delete_before_sync: false` in config.json

**To disable delete by default**:
Edit `agent-skills/infrastructure/skills/devstack/config.json`:
```json
{
  "delete_before_sync": false
}
```

**Note**: Keeping `delete_before_sync: true` (default) is recommended for clean deployments

#### 9. Execute Deployment

**CRITICAL - No Selector Flag**:
Deploy all uncommented services without using selector flags:

```bash
# Sync all uncommented services
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

**Complete deployment command** (executed together):
```bash
helmfile --kube-context dev-serve -f helmfile.yaml delete || true; \
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

**Monitor Output For**:
- Delete completion (if release existed)
- Helm release creation status
- Any error messages
- Resource creation confirmations

**If Deployment Fails**:
- Capture error output
- Proceed to debugging phase
- Provide detailed error analysis

**Success Indicators**:
```
# Delete phase (may not exist)
release "<service>-<label>" uninstalled

# Sync phase
Installing release=<service>-<label>, chart=./charts/<service>
Release "<service>-<label>" has been installed. Happy Helming!
```

### Phase 3: Initial Health Check

#### 10. Verify Deployment Created Resources

Wait 10 seconds after deployment, then check:

```bash
# Check if pods were created
kubectl --context dev-serve get pods -n <namespace> -l devstack_label=<label>

# Check if services were created
kubectl --context dev-serve get svc -n <namespace> | grep <label>
```

**Expected State After Deployment**:
- Pods: ContainerCreating or Running (may take 30-60s)
- Services: Created with ClusterIP assigned
- No immediate errors in events

#### 11. Fetch Release Notes and Build Access Summary (Parallel)

After helmfile sync, retrieve notes for ALL successfully deployed releases **simultaneously** — issue all `helm get notes` commands in a single parallel batch, not one by one.

```bash
# Issue ALL of these in parallel (single message with multiple tool calls):
helm --kube-context dev-serve get notes <release-1> -n <namespace-1>
helm --kube-context dev-serve get notes <release-2> -n <namespace-2>
# ... one per deployed release
```

- Release name = `<service>-<devstack_label>` (e.g., `pg-router-parag`)
- Namespace = the service's namespace from helmfile.yaml
- **Skip** any release that appears in `FAILED RELEASES`

**Parse the NOTES output** — extract URLs and headers from it to build a clean access summary. Do not guess or derive URLs from service name + label patterns; the NOTES block is the authoritative source. Some services have non-standard URL patterns (e.g., `cms-live-parag`, `terminals-live-parag` and `terminals-test-parag`) that would be wrong if derived blindly.

Extract lines that contain:
- `URL :` or `https://` — these are access URLs
- `Header :` — the request header for shared URL routing

Present them in a structured table, not as raw NOTES text. Omit boilerplate lines (greetings, reminders to run helmfile delete, etc.).

#### 12. Transition to Monitoring

After initial deployment:
- Wait 30-60 seconds for pods to initialize
- Proceed to [Monitoring Subskill](monitoring.md) for health verification
- If issues detected, proceed to [Debugging Subskill](debugging.md)

## Service Processing Order

**When deploying multiple services, always process them in the order they appear in helmfile.yaml** (top-to-bottom in the releases block). Do not sort alphabetically or by user mention order. This ensures any implicit ordering (e.g. service A must deploy before service B) is respected.

## Multi-Service Deployment Patterns

### Deploy All Uncommented Services (No Service Specified)

**User Request**: "/devstack deploy" OR "deploy all services"

**Actions**:
1. Search helmfile.yaml for all uncommented service entries (lines not starting with #)
2. Parse each service name and namespace
3. Present complete list to user with AskUserQuestion tool:
   ```
   Found X uncommented services in helmfile.yaml:
   - service1 (namespace: ns1) - line 123
   - service2 (namespace: ns2) - line 456
   - service3 (namespace: ns3) - line 789

   Deploy all these services?
   ```
4. If user confirms:
   - Deploy all services in a single helmfile command WITHOUT selector flags
   - Monitor all deployments
5. If user declines:
   - Ask which specific services to deploy

**Commands**:
```bash
# Deploy all uncommented services together (no selector flag)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

**Example**:
```bash
# User runs: /devstack deploy
# Assistant finds: pg-router, account-service, api (all uncommented)
# After confirmation, deploys all at once without selector
helmfile --kube-context dev-serve -f helmfile.yaml delete || true && helmfile --kube-context dev-serve -f helmfile.yaml sync
```

### Deploy Multiple Specific Services

**User Request**: "deploy pg-router with abc123, asv with existing, api with def456"

**IMPORTANT**: Deploy all services in a SINGLE helmfile command WITHOUT selector flags!

**Actions**:
1. Parse the request to extract:
   - Service names (pg-router, asv, api)
   - Image requirements (abc123, existing, def456)
2. For each service:
   - Locate in helmfile.yaml
   - **CRITICAL**: If ANY service is commented, AUTOMATICALLY uncomment it
   - **CRITICAL**: Read current `image:` value
   - **UPDATE image field** ONLY if new commit hash provided (abc123, def456)
   - **KEEP existing image** if "existing", "current", or "keep" specified (asv)
   - Report what image will be used for each service
3. **CRITICAL**: Ensure ALL other services in helmfile.yaml are COMMENTED OUT
4. After ALL specified services are uncommented and images updated:
   ```bash
   # CORRECT: Deploy all uncommented services (no selector flag)
   helmfile --kube-context dev-serve -f helmfile.yaml delete || true
   helmfile --kube-context dev-serve -f helmfile.yaml sync
   ```

**Commands**:
```bash
# Multi-service deployment without selector flag
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

**Example**:
```yaml
# Before update
- name: pg-router-{{ .Values.devstack_label }}
  values:
    - image: old123

- name: account-service-{{ .Values.devstack_label }}
  values:
    - image: old456  # Keep this (user said "existing")

- name: api-{{ .Values.devstack_label }}
  values:
    - image: old789

# After update (only pg-router and api changed)
- name: pg-router-{{ .Values.devstack_label }}
  values:
    - image: abc123  # UPDATED

- name: account-service-{{ .Values.devstack_label }}
  values:
    - image: old456  # UNCHANGED (existing)

- name: api-{{ .Values.devstack_label }}
  values:
    - image: def456  # UPDATED
```

**Deploy Command**:
```bash
# Deploy all uncommented services (pg-router, account-service, api)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

### Deploy Multiple Services with Same Image

**User Request**: "deploy pg-router, api, asv all with image abc123"

**Actions**:
1. Locate all three services in helmfile.yaml
2. **CRITICAL**: Uncomment pg-router, api, and asv if they are commented
3. **CRITICAL**: Comment out all other services
4. Update all three images to abc123
5. Deploy all together

**Commands**:
```bash
# Deploy all uncommented services (no selector flag)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

## Common Deployment Patterns

### Deploy Service with Specific Image Tag

**User Request**: "Deploy payment-service with label john using image abc123"

**Actions**:
1. Find service in helmfile.yaml
2. **CRITICAL**: If commented, AUTOMATICALLY uncomment it
3. **CRITICAL**: Ensure all other services are commented out
4. Update image value to abc123
5. Validate configuration
6. Delete existing deployment (ignore failures)
7. Run helmfile sync (deploys only uncommented service)
8. Monitor deployment

**Commands**:
```bash
# Deploy only uncommented services (payment-service in this case)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

### Deploy Previously Commented Service

**User Request**: "Deploy api-gateway with label alice"

**Actions**:
1. Find commented service in helmfile.yaml
2. **CRITICAL**: AUTOMATICALLY uncomment entire release block
3. **CRITICAL**: Ensure all other services are commented out
4. Add missing configurations (if any)
5. Validate configuration
6. Delete existing deployment (ignore failures)
7. Run helmfile sync (deploys only api-gateway)
8. Report uncommented service to user

**Commands**:
```bash
# Deploy only uncommented service (api-gateway)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

### Redeploy After Configuration Changes

**User Request**: "Redeploy merchant-service with updated memory limits"

**Actions**:
1. Verify new configuration in values.yaml
2. **CRITICAL**: Ensure merchant-service is uncommented
3. **CRITICAL**: Ensure all other services are commented out
4. Run template validation
5. Delete existing deployment (clean slate)
6. Execute helmfile sync (deploys only merchant-service)
7. Monitor deployment
8. Verify new pods have updated resources

**Commands**:
```bash
# Deploy only uncommented service (merchant-service)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync
```

**Note**: Delete ensures configurator hooks (DB, SQS, SNS) run with fresh config

## Deployment Flags and Options

### Helmfile Deployment Options

**CRITICAL**: All deployments use NO selector flags. Service selection is done by uncommenting in helmfile.yaml.

```bash
# Standard deployment (DEFAULT - with delete first, no selector)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync

# One-liner (preferred)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true; helmfile --kube-context dev-serve -f helmfile.yaml sync

# Update existing (skip delete) - only if user explicitly requests
helmfile --kube-context dev-serve -f helmfile.yaml sync

# Force recreate (if pods stuck in bad state)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync --force

# Skip validation (not recommended)
helmfile --kube-context dev-serve -f helmfile.yaml delete || true
helmfile --kube-context dev-serve -f helmfile.yaml sync --skip-deps
```

### Service Selection via Uncommenting

**CRITICAL WORKFLOW**:
1. **For deploying specific service(s)**: Uncomment ONLY the target service(s), comment out all others
2. **For deploying all services**: Ensure all desired services are uncommented
3. **Run helmfile without selector flags**: This deploys ALL uncommented services

## Error Scenarios

### Helmfile Sync Fails

**Error**: `Error: release <service> failed`

**Actions**:
1. Check helmfile output for specific error
2. Common causes:
   - Invalid YAML syntax → Fix and retry
   - Missing chart directory → Verify path
   - Invalid values → Check values.yaml
3. Apply fix and retry deployment

### Image Pull Errors During Deployment

**Error**: `Failed to pull image "c.rzp.io/razorpay/<service>:<tag>"`

**Actions**:
1. Verify image tag exists in registry
2. Check image name format
3. Suggest valid image tag
4. Update helmfile.yaml and redeploy

### Namespace Not Found

**Error**: `Error: namespace "<namespace>" not found`

**Actions**:
1. Verify namespace in helmfile.yaml matches cluster
2. Create namespace if missing:
   ```bash
   kubectl --context dev-serve create namespace <namespace>
   ```
3. Retry deployment

## Auto-Fix Examples

### Example 1: Missing Resource Limits

**Before** (values.yaml):
```yaml
web_requests_cpu: 50m
web_requests_memory: 50Mi
# Missing limits!
```

**Auto-Fix Applied**:
```yaml
web_requests_cpu: 50m
web_requests_memory: 50Mi
web_limits_memory: 100Mi     # ADDED
# NOTE: CPU limits intentionally NOT added to prevent throttling
```

### Example 2: Commented Service

**Before** (helmfile.yaml):
```yaml
# - name: payment-service-{{ .Values.devstack_label }}
#   namespace: payment-service
#   chart: ./charts/payment-service
```

**Auto-Fix Applied**:
```yaml
- name: payment-service-{{ .Values.devstack_label }}
  namespace: payment-service
  chart: ./charts/payment-service
```

### Example 3: Missing TTL Annotation

**Before** (deployment.yaml):
```yaml
metadata:
  annotations:
    app: payment-service
  # Missing TTL!
```

**Auto-Fix Applied**:
```yaml
metadata:
  annotations:
    app: payment-service
    janitor/ttl: "{{ .Values.ttl }}"    # ADDED
```

## Best Practices

### Always Validate Before Deploy
- Run template validation
- Check configuration against checklist
- Verify image tag exists

### Use Appropriate TTL Values
- `1h` - Short-lived testing
- `8h` - Daily development work
- `forever` - Long-running environments (use sparingly)

### Set Proper Resource Limits
- Start conservative (50m CPU, 100Mi memory)
- Monitor actual usage
- Adjust based on metrics

### Label Consistently
- Use meaningful devstack labels (your username)
- Include labels in all resources
- Use labels for cleanup

## Output Examples

### Single Service — Successful Deployment

```
## ✅ Deployment Complete

### What I Did
1. ✅ Found pg-router in helmfile.yaml:1066
2. ✅ devstack_label: parag, ttl: 2h
3. ✅ Image: 38a86d11 (from base deployment)
4. ✅ Commented out 5 other services
5. ✅ All 14 images validated via Harbor
6. ✅ helmfile delete + sync completed in 45s

### pg-router-parag (pg-router) ✅ — 45s

Parsed from `helm get notes pg-router-parag -n pg-router`:

```
Shared:  https://pg-router.dev.razorpay.in  (header: rzpctx-dev-serve-user: parag)
Direct:  https://pg-router-parag.dev.razorpay.in
```

### Pod Health
pg-router-parag: 1/1 Running

### Next Steps
Waiting 30 seconds for pods to initialize, then monitoring health...
```

### Batch Deployment — Mixed Success/Failure

```
## ⚠️ Deployment Partially Complete (5/6 succeeded)

### What I Did
1. ✅ 6 services located and configured (label: parag-rzp, ttl: 1h)
2. ✅ All 22 images validated via Harbor
3. ✅ helmfile delete + sync completed

### Access (parsed from helm get notes per release)

| Service | Access |
|---|---|
| api ✅ | Shared: https://api.dev.razorpay.in (header: rzpctx-dev-serve-user: parag-rzp)<br>Direct: https://api-parag-rzp.dev.razorpay.in |
| pg-router ✅ | Shared: https://pg-router.dev.razorpay.in (header: rzpctx-dev-serve-user: parag-rzp)<br>Direct: https://pg-router-parag-rzp.dev.razorpay.in |
| reporting ❌ | — |

### Failures

**reporting-parag-rzp** ❌
```
Error: ServiceAccount "reporting-service-account" exists and cannot be imported
into the current release: invalid ownership metadata
```
Fix: Add `- create_sa: true` to reporting values in helmfile.yaml and redeploy.

### Pod Health
- api: 1/1 Running
- pg-router: 1/1 Running
- reporting: — (not deployed)

### Next Steps
- Re-deploy reporting with `create_sa: true` fix applied
- Monitoring api and pg-router pod health...
```

### Deployment with Auto-Fixes

```
## ✅ Deployment Complete (with auto-fixes)

### What I Did
1. ✅ Found api-gateway in helmfile.yaml:445
2. ⚠️ Service was commented — AUTOMATICALLY uncommented
3. ⚠️ Missing web_limits_memory — AUTOMATICALLY added: 200Mi
4. ✅ All images validated
5. ✅ helmfile delete + sync completed

### Auto-Fixes Applied
- Uncommented service at helmfile.yaml:445-450
- Added web_limits_memory: 200Mi to values.yaml:23

### api-gateway-alice (api-gateway) ✅ — 38s

Parsed from `helm get notes api-gateway-alice -n api-gateway`:

```
Shared:  https://api-gateway.dev.razorpay.in  (header: rzpctx-dev-serve-user: alice)
Direct:  https://api-gateway-alice.dev.razorpay.in
```

### Pod Health
api-gateway-alice: 1/1 Running
```

## Related Subskills

- [Validation](validation.md) - Deep-dive into configuration validation
- [Monitoring](monitoring.md) - Post-deployment health checks
- [Debugging](debugging.md) - Troubleshooting failed deployments

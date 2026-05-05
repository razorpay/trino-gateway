---
name: agent-readiness-score
description: Scores a Razorpay service repo against the Agentic SDLC Scorecard across three pillars — Context (C1–C4), Testing (T1–T4), CI/CD (D1–D4) — and outputs a band per pillar plus an aggregate score. Use when assessing how agent-ready a service is or tracking progress across teams.
allowed-tools: Read, Grep, Glob, Task, Bash, mcp__coralogix-nonprod-server__metrics__range_query_v1, mcp__coralogix-nonprod-server__get_logs_v1
argument-hint: "<repo-name>  e.g. order-service or razorpay/order-service"
---

Three independent pillars, each with its own band. Aggregate = (C+T+D)/56×100.
Band scale: <30 Not Ready | 30–54 Needs Work | 55–79 Assisted | 80+ Ready

Scoring rubrics: [references/scoring-rubrics.md](references/scoring-rubrics.md)  
Full scorecard reference: [references/scorecard.md](references/scorecard.md)

C4 (Agent Evals) carries 0 weight — tracked but not scored.

---

## Dependencies

| Dependency | Purpose | Setup |
|-----------|---------|-------|
| `gh` CLI | Clone repos | `gh auth login` |
| `SONAR_TOKEN` env var | Fetch coverage from `sonar.razorpay.com` | `export SONAR_TOKEN=<token>` |
| `razorpay/kube-manifests` | Devstack v2 migration status | gh read access |
| `razorpay/spinacode` | Deployment pipeline validation | gh read access |
| `coralogix-nonprod-server` MCP | Devstack uptime + E2E success rate | Configured in MCP servers |

Clone dirs (workspace is fresh each run, no cleanup needed):
- `/tmp/agent-readiness-score-<repo>` — service repo (full clone for git log)
- `/tmp/agent-readiness-score-kube-manifests` — shallow
- `/tmp/agent-readiness-score-spinacode` — shallow
- `/tmp/agent-readiness-score-vajra-iac` — shallow
- `/tmp/agent-readiness-score-alert-rules` — shallow

---

## Examples

```
/agent-readiness-score swe-agent
/agent-readiness-score razorpay/orders-service
/agent-readiness-score https://github.com/razorpay/auth-service
```

Output: score report pushed to `razorpay/<APP_NAME>.wiki/agent-readiness-score`.

---

## STEP 0 — Load exceptions & aliases

Parse `$ARGUMENTS` to extract `APP_NAME` (same logic as STEP 1), then read `exceptions.yaml`:
1. `Specs/agentic_sdlc_scoring/exceptions.yaml` (team-shared canonical)
2. Same directory as this SKILL.md (local fallback)

Two formats are supported per repo:

```yaml
# Format 1 — bare list (exceptions only, backward-compatible)
swe-agent:
  - signal: T1-E2E
    reason: ...

# Format 2 — dict with optional keys (needed for aliases or both)
emv-card-auth-server:
  exceptions:
    - signal: T1-E2E
      reason: ...
  spinacode_aliases:
    - cas-acs
```

Parse: if the value is a **list**, treat it as exceptions. If it's a **dict**, read `.exceptions` and `.spinacode_aliases`.

Extract excepted signal IDs into `EXCEPTIONS` set. Empty if no entry or no exceptions.

Extract `spinacode_aliases` list from `APP_NAME.spinacode_aliases` if present. This handles **monorepos** where the Spinnaker app names differ from the repo name (e.g. `emv-card-auth-server` deploys as `cas-acs`, `cas-acs-common`, etc.). The alias list is used in D2 lookup before falling back to the standard three-pass search.

> Signal IDs: `C1` `C2` `C3` | `T1-UT` `T1-SLIT` `T1-E2E` `T2` `T3` `T4-Uptime` `T4-E2E` | `D1` `D2` `D3` `D4` `D5`

---

## STEP 1 — Parse arguments and clone

Parse `$ARGUMENTS`: strip `https://github.com/` prefix, prepend `razorpay/` if no `/`, extract `APP_NAME` as last path segment (strip `.git`). Set `CLONE_DIR=/tmp/agent-readiness-score-<APP_NAME>`.

```bash
gh repo clone razorpay/$APP_NAME $CLONE_DIR -- --quiet

KUBE_DIR=/tmp/agent-readiness-score-kube-manifests
SPINA_DIR=/tmp/agent-readiness-score-spinacode
VAJRA_DIR=/tmp/agent-readiness-score-vajra-iac
ALERT_RULES_DIR=/tmp/agent-readiness-score-alert-rules
gh repo clone razorpay/kube-manifests $KUBE_DIR -- --quiet --depth=1 &
gh repo clone razorpay/spinacode $SPINA_DIR -- --quiet --depth=1 &
gh repo clone razorpay/vajra-iac $VAJRA_DIR -- --quiet --depth=1 &
gh repo clone razorpay/alert-rules $ALERT_RULES_DIR -- --quiet --depth=1 &
wait
```

List top-level contents of `CLONE_DIR` before proceeding.

---

## STEP 2 — Score Context (C, max 14)

Copy `scripts/validate-context.sh` from this skill's directory to `/tmp/agent-readiness-score-validate-context.sh`, then run:
```bash
bash /tmp/agent-readiness-score-validate-context.sh $CLONE_DIR
```

- **C1** (max 6): map fail count from script output → rubric
- **C3** (max 4): read `Freshness Score: N/4` from script output. Score 0 if no `sources`/`extracted_at` frontmatter in repo-skill docs.
- **C2** (max 4): requires agent judgment — do not use script's coverage % directly.

The script emits a warning for each uncovered dir: `"$relpath/ ($N src files) — no AGENTS.md or CLAUDE.md"`. For each:
1. Read 3–5 files (core logic, not structs/constants)
2. Judge: meaningful domain logic or subsystem boundary an agent needs explained? **Yes** → gap. **No** (thin wrappers, generated code, constants) → exclude.
3. `ADJUSTED_COVERAGE = dirs_with_signal / (dirs_with_signal + genuine_gaps) × 100` → rubric

**C = C1 + C2 + C3** (max 14, C4 skipped — 0 pts until framework defined)

---

## STEP 3 — Score Testing (T, max 30)

### T1 — Test Coverage (max 14)

MCC (Minimum Code Coverage) only runs on SLIT and E2E — never UT. SonarQube only reliably has UT. Use each source for what it actually covers:

- **UT → SonarQube**
- **SLIT + E2E → Coralogix MCC logs**

---

**UT coverage — SonarQube:**

```bash
# SONAR_TOKEN is base64-encoded in task-worker
_SONAR_TOKEN_RESOLVED=$(echo "${SONAR_TOKEN}" | base64 -d 2>/dev/null)
echo "${_SONAR_TOKEN_RESOLVED}" | grep -q "^squ_" || _SONAR_TOKEN_RESOLVED="${SONAR_TOKEN}"

# Discover UT Sonar project key
grep -rh "sonar.projectKey" $CLONE_DIR/.github/workflows/ $CLONE_DIR/sonar-project.properties 2>/dev/null \
  | grep -i "UT\|Unit" | sort -u

curl -s --user "${_SONAR_TOKEN_RESOLVED}:" \
  "https://sonar.razorpay.com/api/measures/component?metricKeys=coverage&component=<UT_PROJECT_KEY>" \
  | jq '.component.measures[].value'
```
- Numeric value returned → UT coverage %.
- No Sonar UT project or auth error → UT coverage = 0%.

---

**SLIT + E2E coverage — Coralogix MCC logs:**

Argo workflow names follow two naming patterns:
- Forward: `{type}-{service}-{random}` (e.g. `slit-order-service-abc12`)
- Reverse: `{service}-{type}-{random}` (e.g. `payments-nb-wallet-slit-xyz45`)

Use `mcp__coralogix-nonprod-server__get_logs_v1` with a 30-day window. Run **two queries per type** — one for each pattern — and take the most recent result:

**Query A (forward pattern):**
```
source logs
| filter $l.applicationname == 'mcc'
| filter $d.log ~ 'latest master in DB'
| filter $d.kubernetes.labels['workflow-name'] ~ 'slit-<APP_NAME>-'
| orderby $m.timestamp desc
| limit 1
```

**Query B (reverse pattern):**
```
source logs
| filter $l.applicationname == 'mcc'
| filter $d.log ~ 'latest master in DB'
| filter $d.kubernetes.labels['workflow-name'] ~ '<APP_NAME>-slit-'
| orderby $m.timestamp desc
| limit 1
```

Use whichever query returns a result (if both return results, take the more recent timestamp). Repeat both queries with `e2e` in place of `slit` for E2E coverage.

Extract coverage % from the log:
```
"Fetched Service Coverage from latest master in DB: 73.9%, Setting threshold as: 73.8%"
```
→ coverage = `73.9`

- Log found (either pattern) → use extracted % for scoring.
- Neither query returns a log → coverage = 0% (service not onboarded to MCC for that test type).

---

Map UT/SLIT/E2E coverages to scores via rubric.

**T1 = UT (max 3) + SLIT (max 6) + E2E (max 5)**

### T2 — API & gRPC Contract Documentation (max 5)

Run gRPC and REST checks; take the higher score. See [references/scoring-rubrics.md](references/scoring-rubrics.md) for full signal-to-score mapping.

**gRPC checks:**
```bash
grep "razorpay/rpc\|razorpay/proto" $CLONE_DIR/go.mod 2>/dev/null           # score 2
cat $CLONE_DIR/buf.yaml 2>/dev/null | grep "^name:"                          # score 3
grep -r "buf lint\|buf breaking" $CLONE_DIR/.github/workflows/ 2>/dev/null   # score 4
grep -r "buf push" $CLONE_DIR/.github/workflows/ 2>/dev/null                 # score 5
```

**REST checks:**
```bash
find $CLONE_DIR/docs -name "swagger.yaml" -o -name "swagger.json" 2>/dev/null   # score 2
grep -r "swag init" $CLONE_DIR/.github/workflows/ $CLONE_DIR/Makefile 2>/dev/null # score 3
find $CLONE_DIR -name "openapi.yaml" -o -name "openapi.json" 2>/dev/null | grep -v vendor # score 4
grep -r "openapi-diff\|spectral" $CLONE_DIR/.github/workflows/ 2>/dev/null   # score 5
```

**T2 = max(gRPC score, REST score)**

### T3 — Devstack v2 Readiness (max 5)

Ref: [MIGRATION-APPROACHES.md](https://github.com/razorpay/kube-manifests/blob/db0d07db54a1106e9ca317e7a306aa1a7683cbad/devstack-prod-parity/MIGRATION-APPROACHES.md)

```bash
ls $KUBE_DIR/dev/$APP_NAME/ 2>/dev/null && echo "v2 confirmed" || echo "v2 not found"
find $CLONE_DIR -maxdepth 2 -name "helmfile.yaml" 2>/dev/null && echo "v1 helmfile detected"
grep -r "from-prod\|prod-template" $KUBE_DIR/dev/$APP_NAME/ 2>/dev/null
```

**Hot Reload (Go services):** If the service is a Go application and hot reload is not set up, use the [`devstack-golang-hot-reload`](https://github.com/razorpay/agent-skills/tree/master/development/skills/devstack-golang-hot-reload) skill to configure CompileDaemon-based hot reload before scoring T3/T4.

Score 4 requires devstack returning live data — fetch via `mcp__coralogix-nonprod-server__metrics__range_query_v1`:
```promql
histogram_quantile(0.99, sum(rate(traefik_service_request_duration_seconds_bucket{service=~".*APP_NAME.*"}[5m])) by (le))
```
No data → score T3 on migration status only.

| Score | Signal |
|-------|--------|
| 0 | No devstack support at all |
| 1 | v1 only — `helmfile.yaml` in service repo; no `dev/<app>/` in kube-manifests |
| 2 | v2 started — `dev/<app>/` present in kube-manifests; provisioning documented in `CLAUDE.md` |
| 3 | v2 + values derived from prod (`from-prod`, `prod-template` markers in `dev/<app>/values.yaml`) |
| 4 | Above + devstack returning live p99/p90 data (Coralogix nonprod) + hot-reload working |
| 5 | Above + E2E suite runs against devstack in CI; single documented provision/teardown command |

**Migration workflow to reach score 2+:**

1. **Check draft PR:** [kube-manifests#25843](https://github.com/razorpay/kube-manifests/pull/25843) has pre-generated `values.yaml` for many services — checkout and merge if present.
2. **Generate if missing:** Use [devstack-prod-parity SKILL.md](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/SKILL.md) with prompt `Run PHASE1 to generate the values.yaml`.
3. **Test after merge:**
   - E2E present → [PHASE3-E2E-TESTING_SKILL.md](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/PHASE3-E2E-TESTING_SKILL.md)
   - No E2E → [PHASE3-Spinnaker-Deploy_SKILL.md](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/PHASE3-Spinnaker-Deploy_SKILL.md) + sanity check (hit API, check logs)
4. **Rollback:** If issues, revert values.yaml and redeploy old helmfile chart via base pod pipeline.

### T4 — Devstack Runtime Health (max 6)

Both signals via `mcp__coralogix-nonprod-server__metrics__range_query_v1`, 7-day window. See [references/t4-queries.md](references/t4-queries.md) for full PromQL.

**Signal A — Devstack Uptime SR (max 3):** kube deployment readiness ratio for `*-base` deployments in `dev-serve` cluster, filtered to `APP_NAME` namespace. No data → score 0.

**Signal B — E2E Success Rate (max 3):** `argo_workflows_e2e_status_total` Succeeded/Total ratio across pr_validation + master_validation + dry_run workflow classes for this service. No data (no runs in 7d) → N/A, excluded from denominator. Skip entirely if `T1-E2E` in `EXCEPTIONS`.

**T = T1 + T2 + T3 + T4** (max 30)

---

## STEP 4 — Score CI/CD (D, max 12)

### D1 — Agent Skill Coverage in CI (max 4)

```bash
ls $CLONE_DIR/.agents/skills/ 2>/dev/null
grep -r "code-review\|pre-mortem\|risk-assessment\|log-volume-optimizer\|baseline-observability" $CLONE_DIR/.github/workflows/ 2>/dev/null
```
Five skills checked: `code-review`, `pre-mortem`, `risk-assessment`, `log-volume-optimizer`, `baseline-observability`.

### D2 — Deployment Automation (max 4)

**Pass 0 — spinacode aliases** (for monorepos where Spinnaker app names ≠ repo name):

If `spinacode_aliases` is set for `APP_NAME` in `exceptions.yaml`, iterate through each alias and look for `$SPINA_DIR/v3/<alias>/`. Collect **all** matching folders — monorepos may have multiple pipelines. Use the **best-scoring** folder for the D2 score.

```bash
# Example: emv-card-auth-server has aliases [cas-acs, cas-acs-common, cas-acs-sbic, cas-acs-dark]
for ALIAS in $SPINACODE_ALIASES; do
  ls $SPINA_DIR/v3/$ALIAS/ 2>/dev/null && SPINA_FOLDERS+=("$SPINA_DIR/v3/$ALIAS")
  ls $SPINA_DIR/$ALIAS/ 2>/dev/null && SPINA_FOLDERS+=("$SPINA_DIR/$ALIAS")
done
```

If aliases matched, skip Passes 1–3 and score using the best-scoring folder.

---

If no aliases configured, fall back to the standard three-pass lookup (stop at first match):

**Pass 1 — exact folder:**
```bash
ls $SPINA_DIR/v3/$APP_NAME/ 2>/dev/null || ls $SPINA_DIR/$APP_NAME/ 2>/dev/null
```

**Pass 2 — app.json name suffix** (handles repos where folder name ≠ repo name, e.g. `swe-agent` → `v3/slash/`):
```bash
grep -r "\"$APP_NAME\"" $SPINA_DIR/v3/*/prod/*/app.json 2>/dev/null | grep '"name"' | head -5
```
Matching file's grandparent dir is the spinacode folder.

**Pass 3 — generic grep fallback:**
```bash
grep -r "$APP_NAME" $SPINA_DIR --include="*.json" --include="*.jsonnet" -l 2>/dev/null | head -10
```

Once `SPINA_FOLDER` found, check:
```bash
grep -r '"type": "templatedPipeline"' $SPINA_FOLDER/ 2>/dev/null | wc -l
ls $SPINA_FOLDER/dev-serve/ 2>/dev/null
find $SPINA_FOLDER -name "canary-config.json" -o -name "progressive-canary*.json" 2>/dev/null
```

Score: not found→0 | v1 jsonnet or non-templated v3→1 | v3+templatedPipeline+prod→2 | +canary→3 | +workflow_dispatch documented in CLAUDE.md→4

### D3 — CI Speed & Build Hygiene (max 3)

Fetch check-run durations for last 5 merged PRs via `gh api`. Extract repo-owned job IDs:
```bash
grep -h "^  [a-zA-Z][a-zA-Z0-9_-]*:$" $CLONE_DIR/.github/workflows/*.yml 2>/dev/null | tr -d ' :' | sort -u
```
Keep only check-runs whose name matches a repo-defined job ID (auto-excludes platform jobs). `CI_P50_SECONDS` = median of per-PR wall-clock (max parallel job duration). Also check:
```bash
grep -r "razorpay/actions/docker-image-build-push" $CLONE_DIR/.github/workflows/ 2>/dev/null | wc -l
grep -r "use-default-caching.*true" $CLONE_DIR/.github/workflows/ 2>/dev/null | wc -l
grep -r "paths:\|paths-ignore:" $CLONE_DIR/.github/workflows/ 2>/dev/null | wc -l
```

**Recommended skills for improving D3 score:**

When reporting a D3 score below 3, surface the following skills to the developer based on the repo's tech stack:

_Go services:_
- `/go-docker-build-optimize` — implements Go module caching via a separate `Dockerfile.gomod` with content-addressable cache tags; significantly reduces Docker build times for Go repos.
- `/go-docker-build-audit` — audits PRs produced by `go-docker-build-optimize` across 6 sequential checks (branch sync, file scope, Dockerfile correctness, CI workflow, build status). Run this after applying the optimize skill.

_All services:_
- `/docker-action-migrate` — migrates legacy `docker build` / `docker push` steps to `razorpay/actions/docker-image-build-push@master` with harbor-cache enabled; addresses the "no central action" gap checked above.
- `/docker-optimisations` — reorders Dockerfile layers for better cache hit rates and checks proto commit pinning; complements the action migration.
- `/docker-pr-audit` — audits and fixes PRs produced by `/docker-action-migrate` and `/docker-optimisations` (branch sync, file scope, Dockerfile correctness, proto pinning, CI status). Run this after applying either skill to non-Go repos.

### D4 — Take-it-Live Config (max 1)

Use the deterministic script/checks for D4 only on Go repos. The deterministic logic here is built only for Go regex patterns.

First determine repo language/runtime from top-level signals (for example: `go.mod` => Go, `package.json` => Node, `pom.xml`/`build.gradle` => Java, `pyproject.toml`/`requirements.txt` => Python).

- **If Go repo:** run the deterministic D4 check below.
- **If non-Go repo:** **skip the deterministic script/check entirely** and use direct agent judgment only. Do not try to adapt the Go regex-based check to other languages. Evaluate D4 manually from repo contents instead.

```bash
find $CLONE_DIR -maxdepth 2 -name "take-it-live.yaml" 2>/dev/null
```
Score 1 if present with service metadata + phases; 0 if absent.

### D5 — Baseline-Monitoring (max 2)

Copy `scripts/d5-baseline-metrics-check.py` from this skill's directory to `/tmp/agent-readiness-score-d5-baseline-metrics-check.py`, then run:
```bash
python3 /tmp/agent-readiness-score-d5-baseline-metrics-check.py \
  --repo $CLONE_DIR \
  --output /tmp/agent-readiness-score-d5.json
```

Use the JSON output as the first-pass baseline-metrics signal, then perform three checks:

1. **Baseline metrics present?**
   Have an agent verify whether the scan shows real baseline instrumentation in production code.
   - `NO` if no relevant surfaces are detected
   - `NO` if evidence is only docs/tests/constants
   - `YES` if at least one real owned surface has required baseline metrics instrumented, or is genuinely satisfied by an approved shared library integration

2. **Baseline dashboard present?**
   Search `$VAJRA_DIR/dashboards/prod/baseline/` using fuzzy matching around `APP_NAME`.

   Allowed candidate forms should stay close to the repo/app name:
   - `$APP_NAME`
   - `${APP_NAME//_/-}`
   - `${APP_NAME//-/_}`
   - `${APP_NAME//-/}`
   - `${APP_NAME//_/}`

   Matching strategy:
   - exact filename match first
   - normalized match next (lowercase, remove `-` / `_`)
   - loose short-form / token-overlap fallback only if there is exactly one unambiguous hit
   - if multiple plausible matches exist, treat as ambiguous and do not auto-mark present

3. **Baseline alert present?**
   Search `$ALERT_RULES_DIR/baseline-monitoring-inputs/` using the same fuzzy strategy against `*.input.yaml` basenames.

Score D5 as:
- **0** → baseline metrics absent
- **1** → baseline metrics present, but dashboard or alert onboarding missing
- **2** → baseline metrics present, baseline dashboard present, and baseline alert present

Dashboard/alert presence never gives credit if baseline metrics are absent.

Expected machine-readable verification block:

```text
BASELINE_METRICS_PRESENT: YES|NO
BASELINE_DASHBOARD_PRESENT: YES|NO|AMBIGUOUS
BASELINE_ALERT_PRESENT: YES|NO|AMBIGUOUS
D5_SCORE: <0-2>
D5_REASON: <one sentence>
```

**D = D1 + D2 + D3 + D4 + D5** (max 15)

---

## STEP 5 — Report

**Run this bash block exactly to compute the denominator — do not calculate it mentally.**

Signal max points: C1=6, C2=4, C3=4 | T1-UT=3, T1-SLIT=6, T1-E2E=5, T2=5, T3=6, T4-Uptime=3, T4-E2E=3 | D1=4, D2=4, D3=3, D4=1, D5=2

Replace the `EXCEPTIONS=` line with the actual exceptions from STEP 0 (space-separated signal IDs, or leave empty string if none):


```bash
# ← Claude fills this in from STEP 0: space-separated signal IDs, e.g. "T4-E2E T1-SLIT"
EXCEPTIONS=""

C_MAX=14; T_MAX=30; D_MAX=12
for signal in $EXCEPTIONS; do
  case "$signal" in
    C1) C_MAX=$((C_MAX-6)) ;; C2) C_MAX=$((C_MAX-4)) ;; C3) C_MAX=$((C_MAX-4)) ;;
    T1-UT)    T_MAX=$((T_MAX-3)) ;; T1-SLIT)   T_MAX=$((T_MAX-6)) ;;
    T1-E2E)   T_MAX=$((T_MAX-5)) ;; T2)         T_MAX=$((T_MAX-5)) ;;
    T3)       T_MAX=$((T_MAX-5)) ;; T4-Uptime)  T_MAX=$((T_MAX-3)) ;;
    T4-E2E)   T_MAX=$((T_MAX-3)) ;;
    D1) D_MAX=$((D_MAX-4)) ;; D2) D_MAX=$((D_MAX-4)) ;;
    D3) D_MAX=$((D_MAX-3)) ;; D4) D_MAX=$((D_MAX-1)) ;;
  esac
done
TOTAL_MAX=$((C_MAX + T_MAX + D_MAX))

# Sanity check
if [ "$TOTAL_MAX" -lt 30 ] || [ "$TOTAL_MAX" -gt 56 ]; then
  echo "DENOMINATOR_ERROR: TOTAL_MAX=$TOTAL_MAX is outside [30,56] — STOP, do not write the report, report this error instead."
else
  echo "OK: C_MAX=$C_MAX T_MAX=$T_MAX D_MAX=$D_MAX TOTAL_MAX=$TOTAL_MAX"
fi
```

**If the output contains `DENOMINATOR_ERROR`, halt — do not write a score report. Report the error to the user instead.**

Use the printed `TOTAL_MAX` value verbatim in the report — **never override it manually**.

Assign bands via [references/scoring-rubrics.md](references/scoring-rubrics.md).

**The report MUST follow this exact structure. Do NOT add, remove, or rename any section. Use the bash-computed values from above.**

---

# Agent Readiness Score — razorpay/<APP_NAME>

| Pillar | Score | /Max | % | Band |
|--------|-------|------|---|------|
| Context | <C_SCORE> | /<C_MAX> | <C_PCT>% | <band> |
| Testing | <T_SCORE> | /<T_MAX> | <T_PCT>% | <band> |
| CI/CD | <D_SCORE> | /<D_MAX> | <D_PCT>% | <band> |
| **Aggregate** | **<TOTAL>** | **/<TOTAL_MAX>** | **<AGGREGATE_PCT>%** | **<band>** |

> Exceptions applied: `<signal> (N/A — <reason>)` — omit this line entirely if no exceptions.

## Sub-criterion Breakdown

| # | Sub-criterion | Score | Max | Key Evidence |
|---|--------------|-------|-----|--------------|
| C1 | Agent-ready plugin outputs | | 6 | |
| C2 | Codebase navigability | | 4 | |
| C3 | Context freshness | | 4 | |
| T1 (UT) | Unit test coverage | | 3 | UT: __% |
| T1 (SLIT) | SLIT / integration coverage | | 6 | SLIT: __% |
| T1 (E2E) | E2E coverage | | 5 | E2E: __% |
| T2 | API & gRPC contract docs | | 5 | |
| T3 | Devstack v2 readiness | | 5 | |
| T4 (Uptime) | Devstack deployment SR | | 3 | Uptime: __% (7d) |
| T4 (E2E SR) | E2E test success rate | | 3 | E2E SR: __% (7d) |
| D1 | Agent skill coverage in CI | | 4 | |
| D2 | Deployment automation | | 4 | spinacode: found/not found |
| D3 | CI speed & build hygiene | | 3 | p50: __s (last 5 PRs) |
| D4 | Take-it-live config | | 1 | present/absent |
| D5 | Baseline-monitoring | | 2 | metrics: yes/no, dashboard: yes/no, alert: yes/no |

For excepted signals: Score = `N/A`, Max = `—`.

For D5, if fuzzy dashboard/alert lookup is ambiguous, include the ambiguous candidates in evidence and score conservatively.
**Do NOT add any other tables, sections, or content between the Sub-criterion Breakdown and the Summary.**

## Summary

3–5 sentences: what works well for agents today, weakest pillar and why, single highest-leverage fix.

---

*Cloned to `$CLONE_DIR` — re-run after changes are pushed.*

---

## STEP 6 — Publish to repo wiki

Push the score to the repo wiki. The target wiki page depends on whether the scored commit is on the default branch:

- **Default branch (master/main):** push to `agent-readiness-score` — the canonical page used by the leaderboard.
- **Non-default branch:** push to `test-agent-readiness-score` — a sandbox page that does not affect leaderboard rankings.

Wiki publishing is **mandatory** — do not skip. **You MUST execute every bash command in this step literally. Do not summarise, simulate, or describe what you would do — run the commands and show the actual output. If a command fails, show the error and retry.**

```bash
WIKI_DIR=/tmp/agent-readiness-score-${APP_NAME}-wiki

# Check wiki is enabled
gh api repos/razorpay/${APP_NAME} --jq '.has_wiki' | grep -q true || {
  echo "Wiki not enabled on razorpay/${APP_NAME} — skipping publish"
  exit 0
}

# Determine target wiki page: use test page for non-default branches
DEFAULT_BRANCH=$(gh api repos/razorpay/${APP_NAME} --jq '.default_branch')
CURRENT_BRANCH=$(git -C $CLONE_DIR rev-parse --abbrev-ref HEAD 2>/dev/null || echo "$DEFAULT_BRANCH")
if [ "$CURRENT_BRANCH" = "$DEFAULT_BRANCH" ]; then
  WIKI_PAGE="agent-readiness-score"
else
  WIKI_PAGE="test-agent-readiness-score"
  echo "Non-default branch detected ($CURRENT_BRANCH vs default $DEFAULT_BRANCH) — publishing to wiki page: $WIKI_PAGE"
fi
```

Clone using `gh repo clone` (avoids embedded-token URLs which are blocked by the task-worker sandbox):

```bash
rm -rf "$WIKI_DIR"
gh repo clone razorpay/${APP_NAME}.wiki "$WIKI_DIR" 2>/dev/null || {
  echo "Wiki not yet initialized for razorpay/${APP_NAME}."
  echo "Go to https://github.com/razorpay/${APP_NAME}/wiki and create the first page, then re-run."
  exit 0
}
```

Write the report, then commit and push:

```bash
cat > $WIKI_DIR/${WIKI_PAGE}.md << 'REPORT'
# Agent Readiness Score — razorpay/<APP_NAME>

> Scored by `/agent-readiness-score`. Band scale: <30 Not Ready | 30–54 Needs Work | 55–79 Assisted | 80+ Ready.

_Last scored: <date>_

<full score report>
REPORT

git -C $WIKI_DIR add ${WIKI_PAGE}.md
git -C $WIKI_DIR -c user.email="agent@razorpay.com" -c user.name="Agent" \
  commit -m "score: update agent readiness score $(date +%Y-%m-%d)" --quiet
git -C $WIKI_DIR push origin HEAD:master \
  || (git -C $WIKI_DIR pull --rebase origin master --quiet && \
      git -C $WIKI_DIR push origin HEAD:master)
```

If the push fails after the rebase retry, do not save to any local path — instead return the full score report as your response text so the user has the content directly.

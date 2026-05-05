# AI Week — Agentic SDLC Quick Scorecard

**Version**: 1.5 | **Date**: May 2026 | **Owner**: DevEx Team  
**Run `/agent-readiness-score <repo>` to generate a score automatically.**

Each pillar is scored independently. Aggregate = (C + T + D) / 58 × 100 for directional tracking.  
Band scale per pillar: **<30 Not Ready** | **30–54 Needs Work** | **55–79 Assisted** | **80+ Ready**

---

## Pillar Summary

| Pillar | Max | Criteria |
|--------|-----|----------|
| Context | 24 | C1 Agent-ready plugin outputs, C2 Codebase navigability, C3 Context freshness, C4 Agent evals (0 pts, TBD) |
| Testing | 31 | T1 Test coverage (UT/3 + SLIT/6 + E2E/5), T2 API & contract docs, T3 Devstack v2 readiness, T4 Devstack runtime health (Uptime SR/3 + E2E SR/3) |
| CI/CD | 15 | D1 Agent skill coverage in CI, D2 Deployment automation, D3 CI speed, D4 Take-it-live config, D5 baseline-monitoring |

---

## Context (max 14 pts)

### C1 — Agent-Ready Plugin Outputs (max 6)

**What we validate:** The repo has the full agent-ready plugin structure installed — directory layout, skills directory, mandatory upstream skills, and extracted knowledge files.

**How we measure:** Run `scripts/validate-context.sh` against the cloned repo. Counts hard failures and warnings.

**What the script checks:**
- `.agents/skills/repo-skill/` directory tree (`core/`, `modules/domain/`, `modules/technical/`, `modules/integration/`)
- `.claude/skills` present (symlink to `.agents/skills` or a real directory — both are valid)
- `.agents/polyfills/agentsmd/` hook scripts present
- Extracted knowledge files: `boundaries.md`, `quick-ref.md`, domain entity files, `technical-patterns.md`, `service-contracts.md`, `external-deps.md`
- `AGENTS.md` present
- `repo-skill/SKILL.md` present
- `CLAUDE.md` (uppercase) present

**Scoring:**

| Failures | Score |
|----------|-------|
| 0 failures, 0 warnings | 6 |
| 0 failures, 1–3 warnings | 5 |
| 1–3 failures | 4 |
| 4–7 failures | 3 |
| 8–12 failures | 2 |
| >12 or AGENTS.md + .agents/ both missing | 0 |

**To improve:** Run the [agent-ready plugin](https://github.com/razorpay/claude-plugins/tree/master/plugins/agent-ready) on the repo.

---

### C2 — Codebase Navigability (max 4)

**What we validate:** Large subdirectories (>20 source files at depth 2–4) have an `AGENTS.md` or `CLAUDE.md` file so agents get scoped context when they navigate into that area.

**How we measure — two steps:**

1. **Script** (`validate-context.sh`) finds qualifying subdirs (depth 2–7, ≥15 source files, source files >50% of total) and checks each for `AGENTS.md` or `CLAUDE.md`. Automatically excludes: `mock`, `gen`, `generated`, `proto`, `pb`, `testdata`, `test/dummy`, `migrations`, `.github`.

2. **Agent judgment** on uncovered dirs — for each dir flagged by the script as missing context, the agent reads 3–5 representative files and decides: does this directory contain meaningful domain logic, complex business rules, or a subsystem boundary an agent would need explained? Thin wrappers, request/response structs, simple constants, and auto-generated helpers are excluded from the gap count. Only genuine domain logic gaps count against coverage.

**Coverage % = dirs with AGENTS.md or CLAUDE.md / total large dirs × 100**

**Scoring:**

| Coverage | Score |
|----------|-------|
| 100% or no large subdirs | 4 |
| 75–99% | 3 |
| 40–74% | 2 |
| 1–39% | 1 |
| 0% | 0 |

**To improve:** Add `AGENTS.md` to each large subdir. Keep it concise (under 100 lines) — focus on what the agent would get wrong without it: local conventions, which files to touch, what to avoid, key entry points.

**Industry best practices (not currently scored — require human review):**
- **Root AGENTS.md includes a module/directory map** — a short index pointing agents to key areas (e.g. `## Structure: auth/ → JWT handling, payment/ → Razorpay gateway`). Research consistently shows this is the highest-leverage single addition for large repos.
- **Clear, consistent directory naming** — agents navigate by reading file trees. `internal/payment/handler.go` is more navigable than `internal/svc2/h1.go`. No automated measure exists but it matters.
- **Architecture overview doc** — a `docs/architecture.md` or equivalent that maps system boundaries, data flows, and key dependencies. Aider's repo-map pattern and Cursor's architecture-in-rules pattern both point here.
- **Avoid stale context files** — an inaccurate AGENTS.md is worse than none. Keep subdirectory files updated as the code evolves.

---

### C3 — Context Freshness (max 4)

**What we validate:** Every knowledge doc in `.agents/skills/repo-skill/` declares which source files it was extracted from, and none of those source files have had commits since the doc was last extracted.

**How it works:** The agent-ready plugin writes a `sources` frontmatter block into each doc at extraction time listing exact source file paths and an `extracted_at` date. C3 validation is then a deterministic `git log` check — no guessing, no dirname matching, language-agnostic, runs in CI.

```yaml
---
sources:
  - internal/payment/handler.go
  - internal/payment/service.go
extracted_at: 2026-04-01
---
```

**How we measure:** `validate-context.sh` walks every `.md` in `repo-skill/`, reads the `sources` + `extracted_at` frontmatter, and runs:
```bash
git log --since="<extracted_at>" -- <source_files>
```
Any commits found = doc is stale.

| State | Score |
|-------|-------|
| All docs fresh — no source files changed since extracted_at | 4 |
| 1–2 docs stale | 2 |
| 3+ docs stale | 0 |
| No docs have frontmatter (plugin not updated yet) | 0 |

**To improve:** Re-run the agent-ready plugin after significant source changes. The plugin writes updated frontmatter automatically. See [`AGENT_READY_FRESHNESS_SPEC.md`](AGENT_READY_FRESHNESS_SPEC.md) for the plugin contract.

**CI integration:** This check is fully deterministic — add `bash validate-context.sh` to your PR workflow to get automatic staleness alerts without any manual review.

---

### C4 — Agent Evals (0 pts — framework TBD)

No scoring framework defined yet. Placeholder for tracking eval suite presence once methodology is established.

---

## Testing (max 30 pts)

### T1 — Test Coverage (max 14) = UT + SLIT + E2E

**What we validate:** Unit, integration (SLIT), and E2E coverage numbers. Each layer scored independently against coverage thresholds.

**How we measure — two data sources:**

- **UT coverage → SonarQube** (the only reliable source for unit test coverage)
- **SLIT + E2E coverage → Coralogix MCC logs** (MCC is the system that runs and records integration/E2E coverage at Razorpay)

---

#### UT Coverage — SonarQube

Discover the UT Sonar project key from CI workflow files:
```bash
grep -rh "sonar.projectKey" .github/workflows/ sonar-project.properties 2>/dev/null \
  | grep -i "UT\|Unit" | sort -u
```

Fetch coverage (requires `SONAR_TOKEN`):
```
GET https://sonar.razorpay.com/api/measures/component
  ?metricKeys=coverage
  &component=<UT_PROJECT_KEY>
```
If no Sonar UT project exists or auth fails, UT coverage = 0%.

---

#### SLIT + E2E Coverage — Coralogix MCC Logs

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

Extract coverage % from the log line:
```
"Fetched Service Coverage from latest master in DB: 73.9%, Setting threshold as: 73.8%"
```
→ coverage = `73.9`

- Log found (either pattern) → use extracted % for scoring.
- Neither query returns a log → coverage = 0% (service not onboarded to MCC for that test type).

---

#### Unit Tests (max 3)

| Score | UT_COVERAGE |
|-------|-------------|
| 0 | < 10% (or no Sonar UT project) |
| 1 | 10–40% |
| 2 | 40–70% |
| 3 | 70%+ |

---

#### SLIT / Integration Tests (max 6)

| Score | SLIT_COVERAGE |
|-------|---------------|
| 0 | 0% (or no MCC log found) |
| 1 | 0–20% |
| 2 | 20–40% |
| 3 | 40–60% |
| 4 | 60–70% |
| 5 | 70–80% |
| 6 | 80%+ |

---

#### E2E Tests (max 5)

| Score | E2E_COVERAGE |
|-------|--------------|
| 0 | 0% (or no MCC log found) |
| 1 | 0–10% |
| 2 | 10–20% |
| 3 | 20–30% |
| 4 | 30–50% |
| 5 | 50%+ |

---

**T1 = UT + SLIT + E2E (max 14)**

---

### T2 — API & gRPC Contract Documentation (max 5)

**What we validate:** External API shapes are machine-readable so agents don't have to infer them from source. Scored separately for gRPC and REST — take the higher of the two if both apply.

---

#### gRPC services

Razorpay centralises proto definitions in [`razorpay/rpc`](https://github.com/razorpay/rpc) and [`razorpay/proto`](https://github.com/razorpay/proto), both published to `buf.build/razorpay/*`. Buf lint and breaking-change detection run at the central repo level — service repos consume generated Go modules via `go.mod`.

**How we measure:**
```bash
# Consumes central proto — contract defined upstream
grep "razorpay/rpc\|razorpay/proto" $CLONE_DIR/go.mod

# Service publishes its own protos (buf.yaml with service name)
cat $CLONE_DIR/buf.yaml 2>/dev/null | grep "name:"

# buf lint / breaking runs in service CI (own proto validation)
grep -r "buf lint\|buf breaking" $CLONE_DIR/.github/workflows/ 2>/dev/null
```

| Score | Signal |
|-------|--------|
| 0 | gRPC service with no `razorpay/rpc` or `razorpay/proto` dependency and no local protos |
| 2 | Consumes `razorpay/rpc` or `razorpay/proto` — contract defined and versioned centrally |
| 3 | Above + service publishes own protos (`buf.yaml` with `name: buf.build/razorpay/<service>` + `proto/` dir) |
| 4 | Above + `buf lint` and `buf breaking` run in service repo CI on every PR |
| 5 | Above + generated clients/SDKs published and consumed by downstream services |

---

#### REST services

**How we measure:**
```bash
# Auto-generated swagger (swaggo/swag) — committed spec
find $CLONE_DIR/docs -name "swagger.yaml" -o -name "swagger.json" 2>/dev/null
grep -r "swaggo/swag\|swag init" $CLONE_DIR/go.mod $CLONE_DIR/Makefile $CLONE_DIR/.github/workflows/ 2>/dev/null

# Spec-first (oapi-codegen / hand-written OpenAPI 3.x)
find $CLONE_DIR -name "openapi.yaml" -o -name "openapi.json" 2>/dev/null | grep -v vendor

# CI keeps spec fresh
grep -r "swag init\|oapi-codegen\|openapi-diff\|spectral" $CLONE_DIR/.github/workflows/ 2>/dev/null
```

| Score | Signal |
|-------|--------|
| 0 | No spec — agent must read handler code to infer API shapes |
| 1 | Partial spec — some endpoints documented, others missing |
| 2 | Full committed spec via `swaggo/swag` annotations covering all external endpoints |
| 3 | Above + spec regenerated in CI on every PR (`swag init` in workflow) keeping it fresh |
| 4 | Spec-first with OpenAPI 3.x (`oapi-codegen` or hand-written) — spec is source of truth, handlers generated from it |
| 5 | Above + breaking change detection in CI (`openapi-diff` or `spectral`) gates merges |

**Go-specific note:** `swaggo/swag` (Swagger 2.0, annotation-based) is the pragmatic choice for existing repos. `oapi-codegen` (OpenAPI 3.x, spec-first) is the industry direction for new services. Both score better than nothing — the key requirement is that the spec is committed and stays current.

---

### T3 — Devstack v2 Readiness (max 5)

**What we validate:** The service runs reliably in devstack (prod-parity) so agents can test changes without touching prod.

**Migration reference:** [`kube-manifests/devstack-prod-parity/MIGRATION-APPROACHES.md`](https://github.com/razorpay/kube-manifests/blob/db0d07db54a1106e9ca317e7a306aa1a7683cbad/devstack-prod-parity/MIGRATION-APPROACHES.md)

**Background:** Devstack v1 uses dev charts under `helmfile/charts/<service>/` in the service repo — ephemeral DB/cache/queue, dev-specific naming, all separate from prod. The prod-parity migration (v2) moves services to run from prod charts, tracked under `dev/<service>/` in kube-manifests. Two migration approaches are in-flight (prod chart with dev conditionals vs. DevPod CRD overlay) — presence of `dev/<app>/` in kube-manifests confirms the service has started this migration.

**How we measure — three data sources:**

1. **v2 migration status** via `razorpay/kube-manifests` (cloned to `/tmp/agent-readiness-score-kube-manifests`):
   ```bash
   ls $KUBE_DIR/dev/$APP_NAME/   # present = prod-parity migration started (v2)
   find $CLONE_DIR -name "helmfile.yaml"  # present in service repo = still on v1 dev charts
   ```
   A `dev/<app>/values.yaml` in kube-manifests is the canonical v2 signal. A `helmfile.yaml` in the service repo means v1 dev charts are still in use.

2. **Devstack live availability** via `coralogix-nonprod-server` MCP — two parallel PromQL queries using `mcp__coralogix-nonprod-server__metrics__range_query_v1`:
   - p99 latency: `histogram_quantile(0.99, sum(rate(traefik_service_request_duration_seconds_bucket{service=~".*<APP_NAME>.*"}[5m])) by (le))`
   - p90 latency: `histogram_quantile(0.90, sum(rate(traefik_service_request_duration_seconds_bucket{service=~".*<APP_NAME>.*"}[5m])) by (le))`
   - Data present = devstack is live and responding with measurable traffic

3. **In-repo signals:** hot-reload config, E2E-against-devstack CI step, provision/teardown docs in `CLAUDE.md`

| Score | Signal |
|-------|--------|
| 0 | No devstack support at all |
| 1 | v1 only — `helmfile.yaml` in service repo; no `dev/<app>/` in kube-manifests |
| 2 | v2 started — `dev/<app>/` present in kube-manifests; provisioning documented in `CLAUDE.md` |
| 3 | v2 + values derived from prod (`from-prod`, `prod-template` markers in `dev/<app>/values.yaml`) |
| 4 | Above + devstack returning live p99/p90 data (Coralogix nonprod) + hot-reload working |
| 5 | Above + E2E suite runs against devstack in CI; single documented provision/teardown command |

**To improve — step-by-step migration workflow:**

1. **Check if values.yaml already exists:** A draft PR ([kube-manifests#25843](https://github.com/razorpay/kube-manifests/pull/25843)) has pre-generated `values.yaml` for many services. If your service is there, checkout the draft branch and merge to master.
2. **Generate if missing:** If no `values.yaml` exists for your service, use the [devstack-prod-parity skill](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/SKILL.md) with prompt: `Run PHASE1 to generate the values.yaml`.
3. **Test the deployment** after values are merged to master:
   - **If E2E tests exist for the service:** Use the [PHASE3 E2E testing skill](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/PHASE3-E2E-TESTING_SKILL.md) to validate the prod chart with dev values.
   - **If no E2E tests:** Deploy the prod chart over base via Spinnaker using the [PHASE3 Spinnaker deploy skill](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/PHASE3-Spinnaker-Deploy_SKILL.md), then run a basic sanity check (hit an API endpoint, check logs for errors).
4. **Rollback if needed:** If the new deployment shows issues, revert the values.yaml commit and redeploy the old helmfile chart via the base pod pipeline.

---

### T4 — Devstack Runtime Health (max 6)

**What we validate:** Actual devstack stability and E2E test reliability over the last 7 days — two independent runtime signals from real telemetry.

Reference dashboard: [DevProductivity FY26 Q1 OKRs](https://grafana.np.razorpay.in/d/f6c01c72-3165-4008-b570-c7dea5eb177e/devproductivity-fy26-q1-okrs?orgId=1&from=now-7d&to=now)

Both fetched via `mcp__coralogix-nonprod-server__metrics__range_query_v1` with a 7-day range.

---

#### Devstack Deployment Uptime SR (max 3)

**What it measures:** % of time the service's `*-base` deployment on `dev-serve` has ready replicas over the last 7 days.

```promql
(count(kube_deployment_status_replicas_ready{cluster='dev-serve', deployment=~'.*-base', namespace=~'APP_NAME|APP_NAME-.*', namespace!~"capital-loc|capital-lender|doc-vault|iso-connector-base24|frontend-universe-node-demo-app-01|edge-cp|backstage|capital-bnpl|capital-bnpl-ext|capital-collections|capital-es|capital-los|capital-scorecard|litellm"}[7d] > 0) by (namespace) / count(kube_deployment_spec_replicas{cluster='dev-serve', deployment=~'.*-base', namespace=~'APP_NAME|APP_NAME-.*', namespace!~"capital-loc|capital-lender|doc-vault|iso-connector-base24|frontend-universe-node-demo-app-01|edge-cp|backstage|capital-bnpl|capital-bnpl-ext|capital-collections|capital-es|capital-los|capital-scorecard|litellm"}[7d] > 0) by (namespace)) * 100
```

No data = service not on devstack → score 0.

| Score | DEVSTACK_UPTIME_PCT (7d) |
|-------|--------------------------|
| 0 | 0% or no data |
| 1 | 1–70% |
| 2 | 70–90% |
| 3 | 90%+ |

---

#### E2E Test Success Rate (max 3)

**What it measures:** % of E2E workflow runs that succeeded over the last 7 days (PR validation + master validation + dry-run).

```promql
sum(increase(argo_workflows_e2e_status_total{status="Succeeded", namespace="argo-workflows", workflow_class=~"end_to_end_tests_service_pr_validation|end_to_end_tests_service_master_validation|end_to_end_tests_dry_run", service="APP_NAME"}[7d])) / sum(increase(argo_workflows_e2e_status_total{namespace="argo-workflows", workflow_class=~"end_to_end_tests_service_pr_validation|end_to_end_tests_service_master_validation|end_to_end_tests_dry_run", service="APP_NAME"}[7d]))
```

No data (no E2E runs in last 7d) → N/A, excluded from denominator.  
If `T1-E2E` is in exceptions, skip this signal too.

| Score | E2E_SUCCESS_RATE_PCT (7d) |
|-------|---------------------------|
| 0 | < 50% |
| 1 | 50–70% |
| 2 | 70–90% |
| 3 | 90%+ |

---

## CI/CD (max 15 pts)

### D1 — Agent Skill Coverage in CI (max 4)

**What we validate:** Five core agent skills installed — three for PR safety (`code-review`, `pre-mortem`, `risk-assessment`) and two for observability (`log-volume-optimizer`, `baseline-observability`).

**How we measure:**
```bash
ls $CLONE_DIR/.agents/skills/
grep -r "code-review\|pre-mortem\|risk-assessment\|log-volume-optimizer\|baseline-observability" .github/workflows/
```

| Score | Signal |
|-------|--------|
| 0 | None of the five skills present in `.agents/skills/` |
| 1 | 1–2 skills present; none wired to CI |
| 2 | 3+ skills present; at least 1 auto-runs on PRs |
| 3 | All 5 present; code-review + observability skills wired to CI |
| 4 | Above + CI parses outputs as structured pass/fail gate |

---

### D2 — Deployment Automation (max 4)

**What we validate:** The service is in `razorpay/spinacode` using v3 template-based pipelines (`type: templatedPipeline`), with progressive rollout. v1 jsonnet pipelines or non-template v3 configs score low — the standard is spinacode templates.

**How we measure** via `razorpay/spinacode` (cloned to `/tmp/agent-readiness-score-spinacode`).

**Pass 0 — spinacode aliases** (for monorepos where Spinnaker app names ≠ repo name):

If `spinacode_aliases` is set for `APP_NAME` in `exceptions.yaml`, iterate each alias:
```bash
# Example: emv-card-auth-server has aliases [cas-acs, cas-acs-common, cas-acs-sbic, cas-acs-dark]
for ALIAS in $SPINACODE_ALIASES; do
  ls $SPINA_DIR/v3/$ALIAS/ 2>/dev/null && SPINA_FOLDERS+=("$SPINA_DIR/v3/$ALIAS")
  ls $SPINA_DIR/$ALIAS/ 2>/dev/null && SPINA_FOLDERS+=("$SPINA_DIR/$ALIAS")
done
```
If aliases matched, skip Passes 1–3. Score using the **best-scoring** folder (highest D2 score among all alias matches).

---

If no aliases configured, fall back to three-pass lookup — stop at the first pass that finds the service:

**Pass 1 — exact folder name:**
```bash
ls $SPINA_DIR/v3/$APP_NAME/ 2>/dev/null    # v3 template-based (preferred)
ls $SPINA_DIR/$APP_NAME/ 2>/dev/null        # top-level v1 jsonnet (legacy)
```

**Pass 2 — app.json name-suffix match** (for repos where the spinacode folder name differs, e.g. `swe-agent` lives under `v3/slash/`):
```bash
grep -r "\"$APP_NAME\"" $SPINA_DIR/v3/*/prod/*/app.json 2>/dev/null | grep '"name"'
grep -r "\"$APP_NAME\"" $SPINA_DIR/v3/*/dev-serve/*/app.json 2>/dev/null | grep '"name"'
# The matching file's grandparent dir (e.g. v3/slash/) is the spinacode folder
```

**Pass 3 — generic grep fallback:**
```bash
grep -r "$APP_NAME" $SPINA_DIR --include="*.json" --include="*.jsonnet" -l 2>/dev/null | head -10
```

Once found (`SPINA_FOLDER`), check template usage, staging, and canary:
```bash
grep -r '"type": "templatedPipeline"' $SPINA_FOLDER/ 2>/dev/null | wc -l
ls $SPINA_FOLDER/dev-serve/ 2>/dev/null     # staging automated
ls $SPINA_FOLDER/prod/ 2>/dev/null          # prod present
find $SPINA_FOLDER -name "canary-config.json" \
  -o -name "progressive-canary*.json" 2>/dev/null
```

| Score | Signal |
|-------|--------|
| 0 | Not found in spinacode at all |
| 1 | Found only in top-level (v1 jsonnet) or v3 exists but pipelines are not `type: templatedPipeline` |
| 2 | v3 + `templatedPipeline` + prod present; staging (`dev-serve`) automated; manual prod gate |
| 3 | Above + `canary-config.json` or `progressive-canary*.json` present |
| 4 | Above + `workflow_dispatch` in service CI with deploy step documented in CLAUDE.md |

---

### D3 — CI Speed & Build Hygiene (max 3)

**What we validate:** The repo's own build/test CI completes fast enough for agents to iterate. Platform-mandated checks that every repo runs (security, code review, AI review) are excluded — they're not in the team's control.

**How we measure** — real wall-clock timing from GitHub API across last 5 merged PRs:
```bash
gh pr list --repo razorpay/$APP_NAME --state merged --limit 5 --json number,mergeCommit \
  --jq '.[] | {number, sha: .mergeCommit.oid}'

gh api "repos/razorpay/$APP_NAME/commits/<SHA>/check-runs" \
  --jq '.check_runs[] | select(.conclusion != null) | {name, duration_s: ((.completed_at | fromdateiso8601) - (.started_at | fromdateiso8601))}'
```

**Filter to repo-owned jobs only** — extract job IDs defined in the repo's own workflow files, then keep only check-runs whose name contains one of those IDs. Platform-injected workflows (security scans, code review suites, AI review) are automatically excluded because they don't match any repo-defined job ID. Also drop any job with `duration_s <= 0`.

```bash
# Extract repo-defined job IDs
grep -h "^  [a-zA-Z][a-zA-Z0-9_-]*:$" .github/workflows/*.yml 2>/dev/null \
  | tr -d ' :' | sort -u
```

`CI_P50_SECONDS` = median of per-PR wall-clock (= max duration among surviving parallel jobs per PR).

**Check central action and caching** — the Razorpay standard for Docker builds is `razorpay/actions/docker-image-build-push` (BuildKit + multiarch + harbor-cache + build metrics). Also check path filters:
```bash
grep -r "razorpay/actions/docker-image-build-push" .github/workflows/ | wc -l
grep -r "use-default-caching.*true" .github/workflows/ | wc -l
grep -r "paths:\|paths-ignore:" .github/workflows/ | wc -l
```

| Score | CI p50 | Additional signals |
|-------|--------|--------------------|
| 0 | >30 min | No `razorpay/actions/docker-image-build-push`; raw `docker build` or no caching |
| 1 | 15–30 min | Uses central action but `use-default-caching` not enabled; no path filters |
| 2 | 5–15 min | Central action + harbor-cache enabled (`use-default-caching: true`); parallel jobs |
| 3 | <5 min | Above + path filters on workflows; jobs split by type (build/test/lint separate) |

---

### D4 — Take-it-Live Config (max 1)

**What we validate:** A `take-it-live.yaml` is committed to the repo root — the structured deployment lifecycle/golden path config covering phases, infra setup, deployment sequence, and GnG readiness gates.

**How we measure:**
```bash
find $CLONE_DIR -maxdepth 2 -name "take-it-live.yaml" 2>/dev/null
```

| Score | Signal |
|-------|--------|
| 0 | No `take-it-live.yaml` present |
| 1 | `take-it-live.yaml` committed with service metadata, phases, and deployment sequence |

---

### D5 — Baseline-Monitoring (max 2)

**What we validate:** Whether the service has crossed the basic baseline-monitoring maturity bar:
1. baseline metrics exist in the service repo
2. a baseline dashboard exists in `razorpay/vajra-iac`
3. a baseline alerting input exists in `razorpay/alert-rules`

**How we measure:**

1. **Baseline metrics gate** using `scripts/d5-baseline-metrics-check.py` plus agent verification
   - Detect likely service surfaces from source code: `http`, `grpc`, `worker`, `egress`, `outbox`
   - Check whether the repo contains real production baseline instrumentation for at least one owned surface
   - Approved shared-library integrations count only if the agent confirms they are genuinely used in production code

2. **Baseline dashboard lookup** in `razorpay/vajra-iac`
   - Search `dashboards/prod/baseline/*.jsonnet`
   - Use fuzzy matching around the app/repo name
   - Exact match first, normalized match next, then unambiguous short-form/token-overlap fallback
   - Ambiguous matches should not be auto-counted as present

3. **Baseline alerting lookup** in `razorpay/alert-rules`
   - Search `baseline-monitoring-inputs/*.input.yaml`
   - Use the same fuzzy matching strategy
   - Ambiguous matches should not be auto-counted as present

| Score | Signal |
|-------|--------|
| 0 | Baseline metrics absent in the service repo |
| 1 | Baseline metrics present, but dashboard or alert onboarding missing |
| 2 | Baseline metrics present, baseline dashboard present, and baseline alert onboarding present |

Dashboard/alert presence never gives credit if baseline metrics are absent.

---

## Signal Exceptions

Some repos are structurally exempt from certain signals — e.g. a tooling or infra repo with no user-facing flows doesn't need E2E tests; a library has no devstack environment.

Exceptions are declared in `exceptions.yaml` in the skill directory:

```
~/.claude/skills/agent-readiness-score/exceptions.yaml   (local)
# or the team-shared copy at:
Specs/agentic_sdlc_scoring/exceptions.yaml
```

### How exceptions work

- An excepted signal is marked **N/A** in the output table and scored **0**.
- Its max points are **subtracted from the pillar denominator** before computing percentages, so the repo isn't penalised for signals that don't apply.
- The aggregate is recalculated over the adjusted total (shown as `/N` in the report header).

### YAML schema

Two formats supported per repo. The scorer handles both:

```yaml
# Format 1 — bare list (exceptions only, backward-compatible)
swe-agent:
  - signal: T1-E2E                     # signal ID — see list below
    reason: No user-facing flows; infrastructure tooling only
    owner: your-github-handle
    added: YYYY-MM-DD

# Format 2 — dict with optional keys (needed for aliases, or both)
emv-card-auth-server:
  exceptions:                          # same list as Format 1
    - signal: T1-E2E
      reason: ...
  spinacode_aliases:                   # Spinnaker app names for monorepos
    - cas-acs
    - cas-acs-common
```

If the value is a **list** → treat as exceptions. If a **dict** → read `.exceptions` and `.spinacode_aliases`.

### How to add an entry

Open a PR against this repo, add your entry under the repo name, get one peer approval.

### Spinacode aliases (for monorepos)

For monorepos where the Spinnaker app names differ from the repo name, add `spinacode_aliases` so D2 lookup finds the correct folders:

```yaml
emv-card-auth-server:
  spinacode_aliases:
    - cas-acs
    - cas-acs-common
    - cas-acs-sbic
    - cas-acs-dark
```

When aliases are present, D2 checks each alias folder in spinacode and scores using the best match. This avoids a false 0 for repos whose directory name can't match the Spinnaker app name.

**Signal IDs:** `C1`, `C2`, `C3` | `T1-UT`, `T1-SLIT`, `T1-E2E`, `T2`, `T3`, `T4-Uptime`, `T4-E2E` | `D1`, `D2`, `D3`, `D4`, `D5`

One peer approval is required. Exceptions should be the exception — if a signal genuinely doesn't apply, add it; don't use exceptions to hide a gap.

---

## Security (future pillar — not scored yet)

Signals and measurement methodology TBD. Reference spec: https://docs.google.com/document/d/1qzVXbohOIIIQlZy5ub9KpeMUcTeuEQkRCOTjb2MdCJ8/edit?tab=t.0

---

## Scoring Output

Scores are written to `Specs/agentic_sdlc_scoring/scores/<repo>.md` and published to the repo's own wiki at `https://github.com/razorpay/<repo>/wiki/agent-readiness-score`.

*Full framework: `AGENTIC_SDLC_SCORING_FRAMEWORK.md` | Automated scoring: `/agent-readiness-score <repo>`*

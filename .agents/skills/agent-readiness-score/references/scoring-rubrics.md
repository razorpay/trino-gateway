# AI Week Score — Scoring Rubrics

## C1 — Agent-Ready Plugin Outputs (max 6)

Maps the fail count from `scripts/validate-context.sh`:

| Failures | Score |
|----------|-------|
| 0, 0 warnings | 6 |
| 0, 1–3 warnings | 5 |
| 1–3 | 4 |
| 4–7 | 3 |
| 8–12 | 2 |
| >12 or AGENTS.md + .agents/ both missing | 0 |

---

## C2 — Codebase Navigability (max 4)

Read the `Navigability Coverage: N%` line from script output.

| Coverage | Score |
|----------|-------|
| 100% or no large subdirs | 4 |
| 75–99% | 3 |
| 40–74% | 2 |
| 1–39% | 1 |
| 0% | 0 |

---

## C3 — Context Freshness (max 4)

Read the `Freshness Score: N/4` line from script output directly.

Requires `sources` + `extracted_at` frontmatter in each repo-skill doc (written by agent-ready plugin). If no docs have frontmatter, score = 0.

| State | Score |
|-------|-------|
| All docs fresh (no source files changed since extracted_at) | 4 |
| 1–2 docs stale | 2 |
| 3+ docs stale | 0 |
| No frontmatter on any doc | 0 |

---

## T1 — Test Coverage (max 14)

T1 = UT + SLIT + E2E. Each scored independently from SonarQube API.

---

### Unit Test Coverage (max 3)

| Score | UT_COVERAGE |
|-------|-------------|
| 0 | No Sonar UT project, or coverage < 10% |
| 1 | 10–40% |
| 2 | 40–70% |
| 3 | 70%+ |

---

### SLIT / Integration Test Coverage (max 6)

| Score | SLIT_COVERAGE |
|-------|---------------|
| 0 | No Sonar SLIT project, or coverage = 0% |
| 1 | 0–20% |
| 2 | 20–40% |
| 3 | 40–60% |
| 4 | 60–70% |
| 5 | 70–80% |
| 6 | 80%+ |

---

### E2E Test Coverage (max 5)

| Score | E2E_COVERAGE |
|-------|--------------|
| 0 | No Sonar E2E project, or coverage = 0% |
| 1 | 0–10% |
| 2 | 10–20% |
| 3 | 20–30% |
| 4 | 30–50% |
| 5 | 50%+ |

---

## T2 — API & gRPC Contract Documentation (max 5)

Score gRPC and REST separately, take the higher of the two.

**gRPC** (central protos in `razorpay/rpc` / `razorpay/proto`):

| Score | Signal |
|-------|--------|
| 0 | gRPC with no `razorpay/rpc` or `razorpay/proto` in `go.mod` and no local protos |
| 2 | Consumes `razorpay/rpc` or `razorpay/proto` — contract defined and versioned centrally |
| 3 | Above + service publishes own protos (`buf.yaml` with `name: buf.build/razorpay/<service>`) |
| 4 | Above + `buf lint` and `buf breaking` run in service repo CI on every PR |
| 5 | Above + generated clients published and consumed by downstream services |

**REST** (Go services):

| Score | Signal |
|-------|--------|
| 0 | No spec — agent must read handler code to infer API shapes |
| 1 | Partial spec — some endpoints documented, others missing |
| 2 | Full committed spec via `swaggo/swag` covering all external endpoints |
| 3 | Above + `swag init` runs in CI on every PR keeping spec fresh |
| 4 | Spec-first with OpenAPI 3.x (`oapi-codegen` or hand-written) |
| 5 | Above + breaking change detection in CI (`openapi-diff` or `spectral`) gates merges |

---

## T3 — Devstack v2 Readiness (max 5)

| Score | Signal |
|-------|--------|
| 0 | No devstack support |
| 1 | v1 only (helmfile in service repo; no `dev/<app>/` in kube-manifests) |
| 2 | v2 confirmed (`dev/<app>/` in kube-manifests); provisioning documented in CLAUDE.md |
| 3 | v2 + template derived from prod (`from-prod`, `prod-template` in kube-manifests values) |
| 4 | Above + devstack returning live p99/p90 latency data (coralogix-nonprod) + hot-reload working |
| 5 | Above + E2E runs against devstack in CI; single documented provision/teardown command |

**Migration path:** Check [draft PR #25843](https://github.com/razorpay/kube-manifests/pull/25843) for pre-generated values.yaml → generate via [PHASE1 skill](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/SKILL.md) if missing → test with [E2E skill](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/PHASE3-E2E-TESTING_SKILL.md) or [Spinnaker deploy skill](https://github.com/razorpay/kube-manifests/blob/master/devstack-prod-parity/PHASE3-Spinnaker-Deploy_SKILL.md) → rollback to helmfile if issues.

---

## T4 — Devstack Runtime Health (max 6)

Measured from `coralogix-nonprod-server` MCP over **last 7 days**. Two independent signals.

### Devstack Deployment Uptime SR (max 3)

`DEVSTACK_UPTIME_PCT` from kube deployment readiness query. If no data (service not on devstack), score 0.

| Score | DEVSTACK_UPTIME_PCT (7d) |
|-------|--------------------------|
| 0 | 0% or no data |
| 1 | 1–70% |
| 2 | 70–90% |
| 3 | 90%+ |

### E2E Test Success Rate (max 3)

`E2E_SUCCESS_RATE_PCT` from argo-workflows e2e status query. If no E2E runs in last 7d (no data), treat as N/A — exclude from denominator.

| Score | E2E_SUCCESS_RATE_PCT (7d) |
|-------|---------------------------|
| 0 | < 50% |
| 1 | 50–70% |
| 2 | 70–90% |
| 3 | 90%+ |

---

## D1 — Agent Skill Coverage in CI (max 4)

Five skills checked: `code-review`, `pre-mortem`, `risk-assessment`, `log-volume-optimizer`, `baseline-monitoring`, `observability-log-optimization`.

| Score | Signal |
|-------|--------|
| 0 | None of the six skills present in `.agents/skills/` |
| 1 | 1–2 skills present; none wired to CI |
| 2 | 3+ skills present; at least 1 auto-runs on PRs |
| 3 | All 6 present; code-review + observability skills wired to CI |
| 4 | Above + CI parses outputs as structured pass/fail gate |

---

## D2 — Deployment Automation (max 4)

Lookup order: (0) `spinacode_aliases` from `exceptions.yaml` — for monorepos with different Spinnaker app names, (1) exact folder name in `v3/<name>/` or `<name>/`, (2) `app.json` `"name"` suffix match across all `v3/*/prod/*/app.json`, (3) generic grep for repo name across all spinacode JSON/jsonnet files. When aliases match multiple folders, the best-scoring folder is used.

| Score | Signal |
|-------|--------|
| 0 | Not found in spinacode at all |
| 1 | Found only in top-level (v1 jsonnet) or v3 folder exists but pipelines are **not** `type: templatedPipeline` |
| 2 | v3 + `templatedPipeline` + prod present; staging (dev-serve) automated; manual prod gate |
| 3 | Above + `canary-config.json` or `progressive-canary*.json` present (canary/progressive rollout) |
| 4 | Above + `workflow_dispatch` trigger in service CI with deploy step documented in CLAUDE.md (agent-triggerable) |

---

## D3 — CI Speed & Build Hygiene (max 3)

p50 wall-clock across last 5 merged PRs (repo-owned jobs only; platform checks excluded).

| Score | CI p50 | Additional signals |
|-------|--------|--------------------|
| 0 | >30 min | No `razorpay/actions/docker-image-build-push`; raw `docker build` or no caching |
| 1 | 15–30 min | Uses central action but `use-default-caching` not enabled; no path filters |
| 2 | 5–15 min | Central action + harbor-cache enabled (`use-default-caching: true`); parallel jobs |
| 3 | <5 min | Above + path filters on workflows; jobs split by type (build/test/lint separate) |

---

## D4 — Take-it-Live Config (max 1)

| Score | Signal |
|-------|--------|
| 0 | No `take-it-live.yaml` in repo |
| 1 | `take-it-live.yaml` committed with service metadata, phases, and deployment sequence |

---

## D5 — Baseline-Monitoring (max 2)

Run `scripts/d5-baseline-metrics-check.py`, then have an agent verify whether the repo has real baseline metrics in production instrumentation code. Approved shared-library integrations (`goutils/grpcserver`, `goutils/worker`, `goutils/request/httpclient`, `goutils/outbox`) count only if the agent confirms they are genuinely used.

Then check for monitoring rollout in the platform repos:
- baseline dashboard in `razorpay/vajra-iac` under `dashboards/prod/baseline/`
- baseline alerting input in `razorpay/alert-rules` under `baseline-monitoring-inputs/`

For dashboard/alert checks, use fuzzy matching around the app/repo name:
- exact match first
- normalized match next (lowercase, remove `-` / `_`)
- short-form/token-overlap fallback only if there is a single unambiguous match
- ambiguous fuzzy matches should not be auto-counted as present

| Score | Signal |
|-------|--------|
| 0 | Baseline metrics absent in the service repo |
| 1 | Baseline metrics present, but dashboard or alert onboarding missing |
| 2 | Baseline metrics present, baseline dashboard present, and baseline alert onboarding present |

Dashboard/alert presence never gives credit if baseline metrics are absent.

---

## Band Scale

Applied per pillar and to the aggregate.

| Score % | Band |
|---------|------|
| 80–100 | Ready |
| 55–79 | Assisted |
| 30–54 | Needs Work |
| 0–29 | Not Ready |

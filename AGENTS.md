# Trino Gateway

Trino query load balancer and reverse proxy. Routes SQL queries from clients (Querybook, Airflow, ad-hoc tools) to appropriate Trino backend clusters based on configurable routing policies. Built with Go + Twirp RPC, backed by MySQL.

## Tech Stack

Go 1.22 · Twirp RPC · MySQL 8+ (GORM/Spine) · Prometheus · Viper/TOML config · Docker/K8s

## Quick Start

```bash
make setup          # Install deps + protoc toolchain
make build          # Compile gateway binary
make test-unit      # Run unit tests
make dev-docker-up  # Start local Docker stack
make dev-migration  # Run DB migrations (use `go run ./cmd/migration` if Makefile target fails)
```

Swagger UI: `http://localhost:8000/admin/swaggerui/`
Metrics: `http://localhost:8002/metrics`

## Architecture

**Owns:** Query routing, backend health monitoring, routing policy CRUD, query audit trail
**Does NOT own:** Trino query execution, data processing, cluster provisioning

Two server processes in one binary:
- **Twirp API** (port 8000) — Management APIs for backends, groups, policies, queries
- **Reverse Proxy** (gateway ports 8080/8081) — Routes actual Trino queries to backends

The proxy calls its own Twirp API via localhost for routing decisions — this is intentional architecture, not a bug.

**Layers:** Twirp Handler → Hooks (Auth/Metrics/RequestID) → Core Logic → Repository → MySQL

## Domain Entities

- **Backend** — Trino cluster instance with health/load tracking and uptime scheduling
- **Group** — Routing group mapping backends via strategies (LEAST_LOAD, ROUND_ROBIN, RANDOM)
- **Policy** — Routing rules matching client requests to groups by headers/port
- **Query** — Audit trail recording which backend handled each routed query
- **Router** — Reverse proxy classifying requests and proxying to selected backends

## Key Patterns & Gotchas

- All entities embed `spine.Model` (Razorpay ORM wrapper). Use `CreateOrUpdate` for upserts.
- Table `groups_` has underscore suffix — MySQL reserved keyword. Never rename it.
- Read endpoints (`/Get*`, `/List*`) skip auth via substring match. Write endpoints require `X-Auth-Key`.
- Validation is commented out across all entities — no runtime business-rule checks exist.
- Router auth has hardcoded exempted service accounts in `router/auth.go` — code change required to update.
- Auth delegation fails open on error — availability over security design choice.
- Query records saved async (goroutine) — audit gaps possible on failures.
- Health check HTTP status code check in `monitor/trino.go` has a tautology bug (never rejects).
- Config loads via TOML layering: `default.toml` → env-specific file. Env vars override with `TRINO-GATEWAY_*` prefix.

## Deeper Context

- **Repo skill:** `.agents/skills/repo-skill/SKILL.md` — Full domain modules with decisions, constraints, flow maps
- **Path-scoped rules:** `.claude/rules/*.md` — Auto-loaded guardrails per entity
- **Swagger spec:** `/admin/swaggerui/` or `rpc/gateway/service.proto`

## Skills Index

| Skill | Trigger |
|-------|---------|
| **repo-skill** | Any domain question — load `.agents/skills/repo-skill/SKILL.md` |
| **code-security** | Security review, auth changes, input validation |
| **tech-spec-reviewer** | Review or write tech specs, design docs, RFCs |
| **log-volume-optimizer** | Optimize logging, reduce log noise |
| **devstack** | Local dev environment setup or debugging |
| **go-code-reviewer** | Go code review, PR review |
| **go-unit-test-generator** | Generate unit tests for Go code |
| **pr-autopilot** | Autonomous PR CI fix and review handling |
| **pre-mortem** | Pre-merge reliability/security check on PRs |
| **api-flow** | Visualize API endpoint execution paths |
| **docker-optimisations** | Optimize Dockerfiles for caching |
| **grafana-mcp** | Query Prometheus metrics via Grafana |
| **grafana-dashboard** | Generate Grafana dashboard JSON |
| **alert-coverage-analyzer** | Find missing alerts for the service |
| **post-deployment-monitor** | Monitor deployments after updates |
| **go-panic-recovery** | Find unprotected goroutines, add panic recovery |
| **k8s-debugger** | Debug Kubernetes pods and deployments |
| **k8s-resource-analyzer** | Compare K8s resource usage vs allocation |
| **helm-chart-generator** | Generate Helm charts |
| **baseline-alerting** | Generate baseline Prometheus alert rules |
| **baseline-monitoring** | Standard metric families for observability |
| **databases** | Database patterns and repository guidance |
| **dataplatform-advisor** | Data system selection framework |
| **trino-analyzer** | Trino query cost analysis and optimization |
| **dp-oncall-debugger** | Debug Data Platform CDC pipeline issues |
| **integration-tester** | End-to-end integration tests |
| **pr-action-analyzer** | Diagnose CI check failures on PRs |
| **pr-qa-tester** | Full PR validation lifecycle |
| **curl-command-generator** | Generate cURL commands for API testing |
| **feature-poc-finder** | Find the right person to contact for a feature |
| **risk-assessment** | Assess PR risk level |

## Agent Config

| File | Agent | Purpose |
|------|-------|---------|
| `AGENTS.md` | All | This file — concise service map |
| `.agents/skills/` | All | Shared skills (via symlinks) |
| `.claude/rules/` | Claude Code | Path-scoped domain rules |
| `.claude/settings.json` | Claude Code | Hooks and permissions |

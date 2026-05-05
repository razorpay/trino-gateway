# Trino Gateway — Repo Skill

**Purpose:** Deep domain knowledge for the Trino Gateway service — a Go-based Trino query load balancer/proxy that routes SQL queries to appropriate backend Trino clusters based on configurable routing policies.

## Loading Rules

### Always Load (core context)
- `modules/core/boundaries.md` — Service boundaries, ownership, subsystems
- `modules/core/quick-ref.md` — Build commands, config, debugging endpoints

### On Mention (domain entities)
- `modules/domain/backend.md` — When working with backends, health monitoring, cluster load
- `modules/domain/group.md` — When working with routing groups, backend selection strategies
- `modules/domain/policy.md` — When working with routing policies, rule evaluation, auth delegation
- `modules/domain/query.md` — When working with query audit trail, FindBackendForQuery
- `modules/domain/router.md` — When working with reverse proxy, request routing, response processing

### On File Change (technical context)
- `modules/technical-patterns.md` — When modifying infrastructure code (spine ORM, boot init, config, hooks)
- `modules/integration/service-contracts.md` — When modifying API definitions or adding endpoints
- `modules/integration/external-deps.md` — When modifying external service calls (Trino, MySQL, auth provider)

## Key Entities

| Entity | Package | Description |
|--------|---------|-------------|
| Backend | `internal/gatewayserver/backendApi/` | Trino cluster instance with health/load tracking |
| Group | `internal/gatewayserver/groupApi/` | Routing group mapping to backends via strategies |
| Policy | `internal/gatewayserver/policyApi/` | Routing rules matching clients to groups |
| Query | `internal/gatewayserver/queryApi/` | Audit trail of routing decisions |
| Router | `internal/router/` | Reverse proxy subsystem for query routing |

## Critical Patterns

- All entities use `spine.Model` (Razorpay custom ORM wrapper over GORM)
- Twirp RPC with hook chain: RequestID → Auth → Metrics → Context
- Router self-calls its own Twirp API via localhost for routing decisions
- Health monitor runs on gocron schedule, calls BackendApi via Twirp
- Query records are saved asynchronously (fire-and-forget goroutine)

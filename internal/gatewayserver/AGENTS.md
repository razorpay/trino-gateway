# gatewayserver — Twirp Management API

Five Twirp RPC services managing trino-gateway's core entities. Each service follows the same pattern: `server.go` (Twirp handler) → `core.go` (business logic) → `repo/` (database).

## Services

| Package | Entity | Key Endpoints |
|---------|--------|--------------|
| `backendApi/` | Backend | CreateOrUpdate, Enable/Disable, MarkHealthy/Unhealthy, UpdateClusterLoad |
| `groupApi/` | Group | CreateOrUpdate, Enable/Disable, **EvaluateBackendForGroups** (routing) |
| `policyApi/` | Policy | CreateOrUpdate, Enable/Disable, **EvaluateGroupsForClient** (routing) |
| `queryApi/` | Query | CreateOrUpdate, **FindBackendForQuery** (routing continuity) |
| `healthApi/` | Health | Check (liveness probe, no auth) |

## Key Rules

- All entities embed `spine.Model` — use `CreateOrUpdate` for upserts, never raw INSERT.
- `groups_` table has underscore suffix (MySQL reserved keyword).
- Read endpoints (`/Get*`, `/List*`) skip auth. Write endpoints require `X-Auth-Key`.
- Validation is commented out in all `validation.go` files — no runtime checks.
- Twirp hooks execute in order: RequestID → Auth → Metrics → Context.
- The `EvaluateBackendForGroups` function implements LEAST_LOAD, ROUND_ROBIN, and RANDOM strategies.

## Proto Definition

`rpc/gateway/service.proto` — all services defined here with OpenAPI v2 annotations.

For full domain docs: `.agents/skills/repo-skill/modules/domain/`

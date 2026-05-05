---
paths:
  - "internal/gatewayserver/backendApi/**"
  - "internal/gatewayserver/models/backend.go"
  - "internal/gatewayserver/repo/backend.go"
  - "internal/monitor/**"
---

## Backend Domain Rules

- Backend active state requires BOTH `is_enabled=true` AND `is_healthy=true`. Never check one without the other.
- CreateOrUpdateBackend is an upsert — finds by ID first, then creates or updates. Do not add separate create/update paths.
- Health check uses two-phase approach: HTTP API check (`/v1/info`) then SQL query. Both must pass for healthy status.
- Load formula is `(running * 2) + (queued / 3)`. The TODOs for CPU/queue-time are unimplemented — do not assume they exist.
- Monitor excludes its own Trino user from load calculation to avoid feedback loop — preserve this behavior.
- UptimeSchedule uses 5-field cron. Parse failure marks backend unhealthy (fail-closed design).
- Stale load stats (older than `StatsValiditySecs`) are treated as load=0, not infinity.
- Enable/Disable and MarkHealthy/MarkUnhealthy are idempotent — safe to call multiple times.

For full context: `.agents/skills/repo-skill/modules/domain/backend.md`

---
paths:
  - "internal/gatewayserver/groupApi/**"
  - "internal/gatewayserver/models/group.go"
  - "internal/gatewayserver/repo/group.go"
---

## Group Domain Rules

- Table name is `groups_` (with underscore) because "groups" is a MySQL reserved keyword. Never rename it.
- GroupBackendsMapping has a composite unique key on `(group_id, backend_id)` — duplicates are rejected at DB level.
- Backend deletion cascades to mappings, but group deletion does NOT cascade (Policy FK prevents multi-level cascade).
- LEAST_LOAD strategy treats stale stats as load=0 — this can route to an unhealthy/unmonitored backend.
- LastRoutedBackend is persisted to DB for cross-replica round-robin consistency, but has a race condition under concurrency.
- Validation in `validation.go` is entirely commented out — no runtime business-rule checks exist.
- DefaultRoutingGroup from config is a hard dependency — if misconfigured, all fallback routing fails silently.

For full context: `.agents/skills/repo-skill/modules/domain/group.md`

---
sources:
  - internal/gatewayserver/models/group.go
  - internal/gatewayserver/groupApi/core.go
  - internal/gatewayserver/groupApi/server.go
  - internal/gatewayserver/repo/group.go
extracted_at: 2026-05-05
---

# Group

A Group is a routing abstraction that maps a named cluster of Trino backends to a load-balancing strategy. Policies resolve client requests to one or more Groups; the Group then selects a single Backend to serve the query. Groups are the bridge between "who is asking" (Policy) and "where the query runs" (Backend).

---

## Decisions

### D1: Table named `groups_` (trailing underscore)

**Context:** The entity's natural name is "group", but `GROUP` is a MySQL reserved keyword. Using it as a table name would require quoting everywhere and break raw SQL tooling.
**Decision:** Name the table `groups_` with a trailing underscore.
**Alternatives considered:**
- **`groups`:** Still a reserved keyword in MySQL 8 -- rejected because every raw query and migration would need backtick-quoting.
- **`routing_groups`:** Semantic rename -- rejected to keep the domain language simple and consistent with the proto definition (`Group`).
**Trade-offs:**
- Gained: Zero quoting issues in migrations, raw queries, and GORM
- Lost: Mismatch between Go type name (`Group`) and table name (`groups_`) -- requires explicit `TableName()` override
**Code:** `models/group.go:TableName()`
**Revisit if:** The service migrates away from MySQL or adopts an ORM that auto-quotes identifiers.

### D2: Three routing strategies (LEAST_LOAD, ROUND_ROBIN, RANDOM)

**Context:** Different Trino workloads need different distribution patterns. Interactive dashboards need the fastest available cluster; batch ETL needs even spread; ad-hoc queries are fine with random.
**Decision:** Support three strategies defined in protobuf as an enum: `LEAST_LOAD` (0), `ROUND_ROBIN` (1), `RANDOM` (2). RANDOM is the default/fallback when the strategy string doesn't match known values.
**Alternatives considered:**
- **Single strategy (round robin only):** Simpler -- rejected because Trino clusters have heterogeneous load and a least-loaded option is essential for latency-sensitive workloads.
- **Weighted routing:** Percentage-based traffic splits -- rejected; adds operational complexity and the three strategies cover known use cases.
**Trade-offs:**
- Gained: Flexibility to tune routing per group without code changes
- Lost: LEAST_LOAD depends on fresh `ClusterLoad` stats from the monitor; stale stats degrade to effectively random (load treated as 0)
**Code:** `groupApi/core.go:findBackend()`, `service.proto` enum `RoutingStrategy`
**Revisit if:** Need for percentage-based canary routing or priority queues across backends.

### D3: LastRoutedBackend tracking for round-robin state

**Context:** Round-robin requires knowing which backend was last selected so the next call picks the next one in sorted order.
**Decision:** Persist `LastRoutedBackend` on the Group row itself, updated after every routing decision in `findBackend()`.
**Alternatives considered:**
- **In-memory counter:** Faster -- rejected because the gateway can have multiple instances; shared DB state ensures consistency across replicas.
- **Redis counter:** Lower latency than MySQL -- rejected to avoid adding a new infrastructure dependency for a single field.
**Trade-offs:**
- Gained: Consistent round-robin across gateway replicas without additional infrastructure
- Lost: Extra DB write on every routed request; under high concurrency two replicas can read the same LastRoutedBackend before either writes, causing occasional duplicate picks (acceptable for load distribution, not for strict ordering)
**Code:** `groupApi/core.go:findBackend()` (lines updating `LastRoutedBackend` after strategy evaluation)
**Revisit if:** Gateway scales to many replicas and the duplicate-pick rate becomes a noticeable skew.

### D4: CreateOrUpdate as a single upsert-like operation

**Context:** Groups are configured declaratively (often from config scripts or CI). The caller wants to "ensure this group exists with these backends" without caring whether it's new.
**Decision:** `CreateOrUpdateGroup` does a `Find` first -- if the group exists it updates, otherwise it creates. This is not a database-level upsert; it's two operations.
**Alternatives considered:**
- **Separate Create and Update endpoints:** More RESTful -- rejected because the primary consumer is automation that wants idempotent "ensure" semantics.
- **MySQL `ON DUPLICATE KEY UPDATE`:** True DB upsert -- rejected because the GORM association replacement (`ReplaceAssociations`) must happen as a separate step anyway.
**Trade-offs:**
- Gained: Idempotent group configuration; callers don't need to know if a group already exists
- Lost: Non-atomic (Find + Create/Update), but acceptable since group configuration is a low-frequency admin operation
**Code:** `groupApi/core.go:CreateOrUpdateGroup()`
**Revisit if:** Group creation becomes a hot path or needs strict atomicity guarantees.

---

## Non-Obvious Constraints

### C1: GroupBackendsMapping composite unique key

**Rule:** The `(group_id, backend_id)` pair has a `UNIQUE KEY` constraint in MySQL.
**Why:** Prevents accidentally assigning the same backend to a group twice, which would skew round-robin distribution and cause confusing duplicate entries.
**Enforced at:** Database level (migration `20210805195304_bootstrap.go`).
**Example:** Attempting to assign backend `trino-prod-1` to group `adhoc` twice will fail at the DB level, not in application code.

### C2: Cascade behavior is asymmetric

**Rule:** The FK from `group_backends_mappings.backend_id` to `backends.id` has `ON DELETE CASCADE ON UPDATE CASCADE`. The FK from `group_backends_mappings.group_id` to `groups_.id` does NOT have cascade.
**Why:** Deleting a Backend should automatically clean up its group memberships (backend is gone, mapping is meaningless). Deleting a Group requires explicit cleanup because Policies reference Groups via FK -- cascade would need to propagate further.
**Enforced at:** Database FKs (migration `20210805195304_bootstrap.go`).
**Example:** Deleting a backend automatically removes it from all groups. Deleting a group while a policy still references it will fail with an FK violation.

### C3: LEAST_LOAD silently degrades when stats are stale

**Rule:** If `Backend.StatsUpdatedAt` is older than `Config.Monitor.StatsValiditySecs`, the backend's load is treated as 0 (not as unavailable).
**Why:** The design prefers routing to a potentially busy cluster over refusing to route at all. A cluster with stale stats is likely still operational; its monitor may just be delayed.
**Enforced at:** `groupApi/core.go:findBackend()` inside the `least_load` case.
**Example:** If `StatsValiditySecs` is 60 and a backend's stats are 120s old, its load is assumed to be 0 -- it will be preferred over a backend reporting load=5 with fresh stats. This can surprise operators expecting stale backends to be deprioritized.

### C4: Validation is entirely commented out

**Rule:** The `validation.go` file contains only commented-out code. There is no runtime validation on Group creation/update parameters.
**Why:** Likely deferred during initial development and never re-enabled. The protobuf layer provides type-level validation (enum for strategy), but no business-rule validation (e.g., backends must exist, strategy must be valid string).
**Enforced at:** Nowhere currently.
**Example:** Creating a group with a non-existent backend ID will succeed at the Group level but the backend won't be found during routing (filtered out by `GetAllActiveByIDs`).

### C5: DefaultRoutingGroup must exist or routing fails hard

**Rule:** If no eligible group has an active backend, the system falls back to `Config.Gateway.DefaultRoutingGroup`. If that group doesn't exist or has no active backends, routing returns an error.
**Why:** The fallback is a safety net, not a guarantee. It assumes operators have configured a valid default group.
**Enforced at:** `groupApi/core.go:EvaluateBackendForGroups()` -- returns error "unable to find Backend for Default Routing Group".
**Example:** Misconfiguring `DefaultRoutingGroup` in the TOML config will cause all requests to fail when primary groups have no active backends.

---

## Flow Map

### EvaluateBackendForGroups (core routing decision)

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Happy path (most traffic) | Router calls with group IDs from policy evaluation | `core.go:EvaluateBackendForGroups()` -> `core.go:findBackend()` | DP1, DP2 | Returns backend_id + group_id |
| No eligible backends in requested groups (uncommon) | All backends in matched groups are inactive | `core.go:EvaluateBackendForGroups()` -> `core.go:findBackend()` on fallback group | DP3 | Falls back to DefaultRoutingGroup; metric incremented |
| Fallback group also has no backends (rare, outage) | DefaultRoutingGroup misconfigured or all backends down | `core.go:EvaluateBackendForGroups()` | DP3 | Hard error returned to router |

**Decision Points:**
- **DP1: Group intersection** -- Intersects the caller-provided group IDs with all active groups. Only groups that are both requested AND enabled are eligible. Why: Policies may reference groups that have been disabled for maintenance; disabled groups must be silently skipped, not error.
- **DP2: Strategy evaluation** -- Selects backend within group based on strategy (LEAST_LOAD / ROUND_ROBIN / RANDOM). Why: Different workload profiles need different distribution; the strategy is a per-group operational lever.
- **DP3: Fallback group invocation** -- When no eligible group yields a backend, falls back to `Config.Gateway.DefaultRoutingGroup`. Why: Ensures queries are never dropped due to transient group/backend unavailability; operators configure a "catch-all" group. Emits `trino_gateway_fallback_group_invoked_total` metric for alerting.

### Create/Update Group

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Create new group | API call, group ID doesn't exist | `core.go:CreateOrUpdateGroup()` -> `repo/group.go:Create()` | DP4 | Group + mappings created |
| Update existing group | API call, group ID already exists | `core.go:CreateOrUpdateGroup()` -> `repo/group.go:Update()` | DP4, DP5 | Group fields + backend mappings replaced |

**Decision Points:**
- **DP4: Exists check** -- `Find` is called first; error means "not found" so create, nil means "found" so update. Why: Provides idempotent "ensure" semantics for automation/CI callers.
- **DP5: Association replacement** -- On update, backend mappings are fully replaced (not merged) via `ReplaceAssociations`. Why: The caller sends the complete desired state; partial updates would require diff logic and create inconsistency risks.

### Enable/Disable Group

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Enable | API call | `repo/group.go:Enable()` | DP6 | Group becomes eligible for routing |
| Disable | API call | `repo/group.go:Disable()` | DP6 | Group excluded from active set |
| Already in target state | API call on already-enabled/disabled group | `repo/group.go:Enable()` or `Disable()` | DP6 | Error returned ("already active"/"already inactive") |

**Decision Points:**
- **DP6: Idempotency guard** -- Enable/Disable check current state and error if already in the target state. Why: Prevents silent no-ops; callers should know if their action had no effect (debatable design -- could also be silently idempotent).

---

## Service Contracts

### Group <-> Backend (many-to-many via GroupBackendsMapping)

- A Group references multiple Backends through `group_backends_mappings`
- Backend deletion cascades to remove mappings (group loses that backend silently)
- Group deletion does NOT cascade to backends (backends survive independently)
- During routing, `findBackend()` calls `backendRepo.GetAllActiveByIDs()` to filter only active, healthy backends from the group's mapping

### Group <-> Policy (referenced as primary and fallback)

- Policy has `group_id` (required) -- the primary group to route to
- Policy has `fallback_group_id` (optional) -- used when primary group evaluation fails
- Both are FK-constrained to `groups_.id` -- a Group cannot be deleted while any Policy references it
- If `fallback_group_id` is not set on a Policy, `policyApi/core.go` defaults it to `Config.Gateway.DefaultRoutingGroup`

### Group <-> Query (audit trail)

- Query records store `group_id` to track which group was chosen for each routed query
- FK from `queries.group_id` to `groups_.id` -- prevents group deletion if queries reference it

### Group <-> Router (runtime consumer)

- The Router calls `Policy.EvaluateGroupsForClient()` first to get candidate group IDs, then calls `Group.EvaluateBackendForGroups()` with those IDs
- The Router receives back a `(backend_id, group_id)` tuple
- This is a Twirp RPC call (in-process when co-located, but uses the protobuf contract)
- See `router/request.go` for the call chain

### Group <-> Monitor (indirect dependency)

- The Monitor service periodically updates `Backend.ClusterLoad` and `Backend.StatsUpdatedAt`
- Group's LEAST_LOAD strategy reads these fields at routing time
- If the Monitor is down, stats go stale and LEAST_LOAD degrades to treating all loads as 0 (see constraint C3)

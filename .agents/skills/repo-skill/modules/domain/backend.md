---
sources:
  - internal/gatewayserver/models/backend.go
  - internal/gatewayserver/backendApi/core.go
  - internal/gatewayserver/backendApi/server.go
  - internal/monitor/core.go
  - internal/monitor/trino.go
extracted_at: 2026-05-05
---

# Backend

A Backend represents a Trino cluster coordinator instance that can receive and execute SQL queries. The gateway routes incoming client requests to one of several Backends based on group membership, routing strategy, and health/load state. Backends are the fundamental unit of capacity in the system.

---

## Decisions

### D1: Two-Tier Health Model (IsEnabled vs IsHealthy)

**Context:** Need to distinguish operator intent from automated health state. An operator may want to drain a cluster for maintenance without the monitor automatically re-enabling it.
**Decision:** Two independent boolean flags: `IsEnabled` (operator-controlled) and `IsHealthy` (monitor-controlled). A backend serves traffic only when BOTH are true.
**Alternatives considered:**
- **Single status field (e.g., ACTIVE/INACTIVE/DRAINING):** Rejected because automated health checks and manual overrides would fight over the same field, causing race conditions during maintenance windows.
- **Priority-based override:** Rejected as overly complex for the current use case.
**Trade-offs:**
- Gained: Clean separation of concerns; operators can disable without worrying about monitor re-enabling
- Lost: Queries to find "active" backends require checking two columns (`repo/backend.go:GetAllActiveByIDs()`)
**Code:** `models/backend.go:Backend{}`, `repo/backend.go:GetAllActiveByIDs()`
**Revisit if:** Need more granular states (e.g., DRAINING that accepts no new queries but finishes existing ones).

### D2: Health Check Uses Both HTTP API and SQL Query

**Context:** A Trino coordinator can respond to HTTP health endpoints while being unable to actually execute queries (e.g., catalog misconfiguration, resource manager issues).
**Decision:** Two-phase health check: first call Trino's `/v1/info` HTTP API to verify the coordinator is up and not starting, then execute a configurable SQL health-check query via the Trino Go client.
**Alternatives considered:**
- **HTTP-only check:** Rejected because HTTP liveness does not prove query execution capability.
- **SQL-only check:** Rejected because a hung coordinator might not respond to new SQL connections, and the SQL client connection setup is heavier.
**Trade-offs:**
- Gained: High confidence that a "healthy" backend can actually serve queries
- Lost: Each health check cycle creates a new SQL connection per backend (no connection pooling across cycles)
**Code:** `monitor/core.go:isBackendUp()` (HTTP) -> `monitor/trino.go:IsClusterUp()` + `monitor/trino.go:IsClusterHealthy()`
**Revisit if:** Health check latency becomes a problem; consider persistent connections or reducing SQL check frequency.

### D3: Load Formula Weights Running Queries Higher Than Queued

**Context:** Need a single numeric load score to compare backends for least-load routing.
**Decision:** `load = (running * 2) + (queued / 3)` where "running" includes RUNNING, PLANNING, FINISHING, DISPATCHING states and "queued" includes QUEUED, STARTING states.
**Alternatives considered:**
- **Simple count of all active queries:** Rejected because running queries consume actual cluster resources (CPU/memory) while queued queries only consume queue slots.
- **Include node count, CPU, queue time:** Marked as TODO in code but not implemented yet.
**Trade-offs:**
- Gained: Running queries (which hold resources) have 6x the weight of queued queries, preventing routing to clusters that are CPU-saturated but have short queues
- Lost: Does not account for query complexity, node count, or actual resource utilization
**Code:** `monitor/core.go:computeClusterLoad()`, `monitor/core.go:getBackendLoad()`
**Revisit if:** Heterogeneous cluster sizes are introduced (load score should normalize by node count), or when Prometheus/VictoriaDB Trino connector is available for CPU metrics.

### D4: UptimeSchedule Controls Time-Based Backend Availability

**Context:** Some Trino clusters are cost-optimized to run only during business hours or specific windows.
**Decision:** Each backend has a cron expression (`UptimeSchedule`) that defines when it should be considered for health checks. Outside the cron window, the backend is automatically marked unhealthy regardless of actual cluster state. Default is `* * * * *` (always eligible).
**Alternatives considered:**
- **External scheduler (e.g., K8s CronJob) to enable/disable:** Rejected to keep scheduling self-contained within the gateway.
- **Start/stop time range:** Rejected as less flexible than cron expressions.
**Trade-offs:**
- Gained: Fine-grained scheduling without external dependencies; clusters can be auto-scaled down during off-hours
- Lost: Cron parsing errors cause the backend to be marked unhealthy (fail-closed behavior)
**Code:** `monitor/core.go:EvaluateBackendNewState()`, `monitor/core.go:isCurrentTimeInCron()`, `utils/utils.go:IsTimeInCron()`
**Revisit if:** Need sub-minute granularity or more complex scheduling rules.

### D5: Monitor Communicates via Twirp RPC, Not Direct DB Access

**Context:** The monitor runs as a separate process/goroutine that needs to read and update backend state.
**Decision:** Monitor uses the gateway's Twirp RPC client (`gatewayv1.BackendApi`) to read backends and update health/load, rather than accessing the database directly.
**Alternatives considered:**
- **Direct DB access from monitor:** Rejected to maintain a single source of truth through the gateway API and avoid bypassing business logic.
**Trade-offs:**
- Gained: Consistent state management; monitor respects same validation as API clients
- Lost: Extra network hop per health check cycle; if gateway API is down, monitor cannot update health state
**Code:** `monitor/core.go:Core{}` (holds `gatewayv1.BackendApi`), `monitor/monitor.go:Execute()`
**Revisit if:** Monitor and gateway are always co-located and the RPC overhead becomes measurable.

---

## Non-Obvious Constraints

### C1: Active Backend = IsEnabled AND IsHealthy

**Rule:** A backend is eligible for traffic routing only when both `is_enabled = true` AND `is_healthy = true`.
**Why:** `IsEnabled` is the operator's intent (manual toggle); `IsHealthy` is the system's assessment (automated). Both must agree.
**Enforced at:** `repo/backend.go:GetAllActiveByIDs()` -- queries with both conditions.
**Examples:** Disabling a backend for maintenance keeps `is_healthy = true` but `is_enabled = false`. A cluster crash sets `is_healthy = false` but `is_enabled` remains `true`.

### C2: UptimeSchedule Cron Uses Standard 5-Field Format

**Rule:** The `uptime_schedule` field must be a valid 5-field cron expression (minute, hour, day-of-month, month, day-of-week). Default is `* * * * *`.
**Why:** Uses `robfig/cron/v3` with `ParseStandard()` which expects exactly 5 fields (no seconds field).
**Enforced at:** `utils/utils.go:IsTimeInCron()` -- parse failure causes backend to be marked unhealthy (fail-closed).
**Examples:** `0 9-17 * * 1-5` = weekdays 9am-5pm UTC. Invalid expression = backend marked unhealthy.

### C3: ClusterLoad Stats Have a Validity Window

**Rule:** When routing with `least_load` strategy, load stats older than `Monitor.StatsValiditySecs` are treated as 0 (stale stats are ignored).
**Why:** If the monitor is down or a backend was just re-added, stale load numbers would cause incorrect routing. Treating stale stats as 0 gives the backend a chance to receive traffic.
**Enforced at:** `groupApi/core.go:findBackend()` -- the `load()` closure checks `StatsUpdatedAt` against the config validity window.
**Examples:** If `StatsValiditySecs = 300` and `StatsUpdatedAt` was 10 minutes ago, effective load = 0.

### C4: ThresholdClusterLoad = 0 Means No Threshold

**Rule:** If `threshold_cluster_load` is 0, load-based health gating is disabled -- the backend is healthy regardless of load.
**Why:** Allows backends to be configured without load limits, useful when load-based routing is handled at the group strategy level rather than health level.
**Enforced at:** `monitor/core.go:isBackendHealthy()` -- `if threshold == 0 || load <= threshold`.

### C5: Monitor Health Check Excludes Its Own Queries

**Rule:** The load calculation query filters out queries from the monitor's own Trino user.
**Why:** Without this filter, the monitor's health-check queries would count toward the cluster load, creating a feedback loop where monitoring increases apparent load.
**Enforced at:** `monitor/core.go:getBackendLoad()` -- SQL WHERE clause `user != '<monitor_user>'`.

### C6: CreateOrUpdateBackend Is an Upsert

**Rule:** The `CreateOrUpdateBackend` API attempts a find-by-ID first; if found it updates, otherwise creates.
**Why:** Simplifies client integration -- callers don't need to track whether a backend was previously registered.
**Enforced at:** `backendApi/core.go:CreateOrUpdateBackend()`

### C7: Enable/Disable and MarkHealthy/MarkUnhealthy Are Idempotent

**Rule:** Calling Enable on an already-enabled backend (or Disable on already-disabled) is a no-op that succeeds silently.
**Why:** Prevents unnecessary DB writes and allows callers to be retry-safe.
**Enforced at:** `repo/backend.go:Enable()`, `repo/backend.go:Disable()`, `repo/backend.go:MarkHealthy()`, `repo/backend.go:MarkUnhealthy()`

---

## Flow Maps

### Health Check (Monitor Cycle)

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Healthy backend (most traffic) | Scheduled interval via gocron | `monitor.go:Execute()` -> `core.go:EvaluateBackendNewState()` -> `core.go:isBackendHealthy()` | DP1, DP2, DP3, DP4 | Backend marked healthy, load stats updated |
| Outside uptime window (common for scheduled clusters) | Scheduled interval | `core.go:EvaluateBackendNewState()` -> `core.go:isCurrentTimeInCron()` | DP1 | Backend marked unhealthy (skips actual health check) |
| Cluster down (rare) | Scheduled interval | `core.go:isBackendUp()` -> `trino.go:IsClusterUp()` | DP2 | Backend marked unhealthy |
| Cluster up but SQL fails (rare) | Scheduled interval | `trino.go:IsClusterUp()` -> `trino.go:IsClusterHealthy()` | DP3 | Backend marked unhealthy |
| Load above threshold (occasional) | Scheduled interval | `core.go:getBackendLoad()` -> `core.go:computeClusterLoad()` | DP4 | Backend marked unhealthy, load stats still updated |
| Invalid cron expression (rare, config error) | Scheduled interval | `core.go:isCurrentTimeInCron()` | DP1 | Backend marked unhealthy (fail-closed) |

**Decision Points:**
- **DP1: UptimeSchedule check** -- Is current time within the backend's cron schedule? Why: cost-optimized clusters should not be probed or marked healthy outside their operating window.
- **DP2: HTTP /v1/info check** -- Is the Trino coordinator responding, not starting, and actually a coordinator node? Why: catches basic unavailability without the overhead of a SQL connection.
- **DP3: SQL health-check query** -- Can the cluster execute a configurable SQL query end-to-end? Why: proves full query execution path works (coordinator -> catalog -> workers).
- **DP4: Load vs threshold** -- Is `computeClusterLoad()` result <= `ThresholdClusterLoad`? Why: prevents routing new queries to overloaded clusters. Threshold of 0 disables this gate.

### Backend Selection for Query Routing

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| New query, least_load strategy (most traffic) | POST /v1/statement | `router/request.go:evaluateRoutingBackend()` -> `groupApi/core.go:EvaluateBackendForGroups()` -> `groupApi/core.go:findBackend()` | DP5, DP6 | Query routed to least-loaded active backend |
| New query, round_robin strategy (common) | POST /v1/statement | Same as above | DP5, DP7 | Query routed to next backend in sorted order |
| No eligible backends in matched groups (rare) | POST /v1/statement | `groupApi/core.go:EvaluateBackendForGroups()` | DP8 | Falls back to DefaultRoutingGroup |
| Existing query (follow-up request) (common) | GET/POST/DELETE with queryId | `router/request.go:ProcessRequest()` -> `QueryApi.FindBackendForQuery()` | -- | Routed to same backend that started the query |

**Decision Points:**
- **DP5: Filter active backends** -- Only backends where `is_enabled = true AND is_healthy = true` from the group's mapped backends. Why: unhealthy or disabled backends must not receive new queries.
- **DP6: Least load selection** -- Compare `ClusterLoad` values, but only if `StatsUpdatedAt` is within `StatsValiditySecs`. Stale stats treated as load 0. Why: stale data would cause misinformed routing; treating as 0 gives benefit of the doubt.
- **DP7: Round robin selection** -- Sort active backend IDs alphabetically, find `LastRoutedBackend`, pick next. Why: deterministic ordering ensures even distribution without shared state beyond `LastRoutedBackend`.
- **DP8: Fallback group** -- When no eligible backend found in any matched group, use `Gateway.DefaultRoutingGroup`. Why: ensures queries are never dropped if specific groups are fully unhealthy.

### Backend Enable/Disable (Manual)

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Enable backend (common, post-maintenance) | API call | `backendApi/server.go:EnableBackend()` -> `repo/backend.go:Enable()` | DP9 | `is_enabled = true`; backend eligible for traffic on next monitor cycle if also healthy |
| Disable backend (common, pre-maintenance) | API call | `backendApi/server.go:DisableBackend()` -> `repo/backend.go:Disable()` | DP9 | `is_enabled = false`; immediately ineligible for new query routing |

**Decision Points:**
- **DP9: Idempotency guard** -- If already in target state, return success without DB write. Why: safe for retries and prevents unnecessary DB updates.

### Cluster Load Update

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Load update during health check (most traffic) | Monitor health check | `monitor/core.go:getBackendLoad()` -> `core.go:computeClusterLoad()` -> `core.go:updateBackendClusterLoad()` | -- | `cluster_load` and `stats_updated_at` updated; Prometheus gauge emitted |
| Manual load update via API (rare) | API call | `backendApi/server.go:UpdateClusterLoadBackend()` | -- | `cluster_load` and `stats_updated_at` updated |

---

## Service Contracts

### Backend <-> Group (via GroupBackendsMapping)

- A Backend belongs to one or more Groups through the `group_backends_mappings` join table.
- Groups evaluate their routing strategy ONLY against backends in their mapping that pass the active filter (`is_enabled = true AND is_healthy = true`).
- The Group's `findBackend()` calls `backendRepo.GetAllActiveByIDs()` with the mapped backend IDs.
- **What breaks:** If a backend ID is referenced in `group_backends_mappings` but the backend is deleted, the mapping becomes orphaned. No cascade delete is enforced at the application level.

### Backend <-> Monitor

- Monitor reads ALL backends (not just enabled ones) via `ListAllBackends` Twirp RPC.
- Monitor evaluates uptime schedule and health for every backend, then calls `MarkHealthyBackend` or `MarkUnhealthyBackend` via Twirp RPC.
- Monitor also calls `UpdateClusterLoadBackend` to persist computed load scores.
- Monitor runs on a configurable interval (`Monitor.Interval`) with `SetMaxConcurrentJobs(1)` -- only one monitor cycle runs at a time.
- Health/unhealthy updates within a single cycle are parallelized via goroutines + WaitGroup.
- **What breaks:** If the gateway API is unreachable, the monitor cannot update health state, and backends retain their last-known health status indefinitely.

### Backend <-> Router

- Router never talks to Backend directly for routing decisions. The chain is: Router -> PolicyApi (resolve groups) -> GroupApi (evaluate backend within groups) -> BackendRepo (filter active backends).
- Router calls `BackendApi.GetBackend()` to resolve the backend's hostname/scheme for HTTP reverse proxy forwarding.
- For UI requests, Router uses `ExternalUrl`; for query requests, Router uses `Hostname`.
- **What breaks:** If a backend's hostname is unreachable, the reverse proxy returns 502 Bad Gateway. The backend remains "healthy" until the next monitor cycle detects the failure.

---

## Gotchas

- **No connection pooling for health checks:** Each monitor cycle creates a new `TrinoClient` (and underlying `sql.DB`) per backend, then tears it down. For many backends with frequent checks, this creates connection churn.
- **HTTP status check has a bug:** In `trino.go:IsClusterUp()`, the condition `!(200 <= resp.StatusCode || resp.StatusCode <= 300)` is always false for any status code (since every int is either >= 200 or <= 300). This means non-2xx responses are not caught by this check. The check still works because subsequent JSON parsing catches truly broken responses.
- **Monitor checks ALL backends including disabled ones:** The monitor evaluates health for disabled backends too. This means a disabled backend will still have its `is_healthy` flag toggled by the monitor. This is harmless but creates unnecessary Trino API calls and log noise.
- **Round-robin depends on string sort order:** The round-robin strategy sorts backend IDs alphabetically. Adding a new backend can shift the rotation order for all existing backends.
- **ExternalUrl vs Hostname:** These serve different purposes -- `Hostname` is the internal cluster address (used for query routing), while `ExternalUrl` is the public-facing address (used for UI redirects and `X-Forwarded-Host`). Misconfiguring these causes queries to route correctly but UI links to break, or vice versa.

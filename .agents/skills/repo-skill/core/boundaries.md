# Trino Gateway - Service Boundaries

## Purpose

Trino Gateway is a Go-based HTTP reverse proxy and load balancer for Trino clusters. It routes incoming Trino client queries to appropriate backend Trino clusters based on configurable routing policies (port, hostname, client tags, connection properties). It also monitors backend health, enforces uptime schedules, and records a query audit trail for routing continuity.

## Ownership Boundaries

### Owns

- **Query routing** -- Intercepts Trino HTTP protocol requests and proxies them to a selected backend cluster. Routing decisions use policy-based group resolution then strategy-based backend selection within the group.
- **Routing policy management** -- CRUD for policies that map client attributes (listening port, hostname, client tags, connection properties) to backend groups. Policies support enable/disable, fallback groups, auth delegation per port, and source header injection.
- **Backend group management** -- CRUD for groups of backends with load-balancing strategies: `round_robin`, `least_load`, or random (default fallback).
- **Backend registry** -- CRUD for Trino backend definitions (hostname, scheme, external URL, uptime schedule as cron, cluster load threshold).
- **Backend health monitoring** -- Scheduled job (configurable interval, default 10m) that checks each backend via Trino HTTP API (`/v1/info`) and SQL driver (`SELECT 1`), evaluates cron-based uptime schedules, measures cluster load (active query count from `system.runtime.queries`), and marks backends healthy/unhealthy.
- **Query audit trail** -- Records query metadata (ID, text, username, client IP, backend ID, group ID, submitted timestamp) after Trino responds. Used to route subsequent requests for the same query (e.g., status polling, kill_query) back to the originating backend.
- **Delegated authentication** -- Optional per-port auth delegation to an external validation provider, with in-memory TTL cache for validated credentials.

### Does NOT Own

- **Trino query execution** -- The gateway never parses, plans, or executes SQL. It is a transparent HTTP proxy.
- **Trino cluster provisioning or scaling** -- Backends are registered manually via the management API. The gateway does not create, destroy, or resize clusters.
- **Data storage or processing** -- No interaction with the data layer (S3, HDFS, Hive Metastore) that Trino clusters read from.
- **User identity management** -- Authentication is either passthrough or delegated to an external provider. The gateway does not store user accounts.
- **Trino catalog/schema configuration** -- Backend Trino clusters manage their own catalogs independently.

## Entry Points

| Entry Point | Port (default) | Purpose | Key Code |
|---|---|---|---|
| **Twirp RPC API server** | 8000 | Management APIs for backends, groups, policies, queries, health check. Protected by `X-Auth-Key` header (read ops exempt). | `main.go:startApiServer()` |
| **Reverse proxy servers** | 8080, 8081 (configurable list) | Actual Trino query routing. Each port is an independent `httputil.ReverseProxy`. Multiple ports allow per-port routing policies. | `main.go:startGatewayServers()` |
| **Prometheus metrics** | 8002 | Standard `/metrics` endpoint for scraping. | `main.go:startMetricsServer()` |
| **SwaggerUI** | 8000 (path `/admin/swaggerui/`) | API documentation served from static files. Root `/` redirects here. | `main.go:startApiServer()` |

## Key Subsystems

### gatewayserver/ -- Management API

Five Twirp RPC services, each with core (business logic), server (Twirp handler), validation, and repo layers:

- **BackendApi** -- Register, update, activate, deactivate, delete Trino backends. Health/load state updates from monitor.
- **GroupApi** -- Manage groups of backends with routing strategies. `groupApi/core.go:EvaluateBackendForGroups()` implements round-robin, least-load, and random strategy selection.
- **PolicyApi** -- Manage routing rules. `policyApi/core.go:EvaluateGroupsForClient()` resolves which groups match a client request by intersecting policy rule matches across four dimensions (port, host, client tags, connection properties).
- **QueryApi** -- Query audit trail CRUD. `FindBackendForQuery()` enables routing continuity for in-flight queries.
- **HealthCheckApi** -- Simple liveness probe (no auth required).

Common Twirp hooks: `hooks/metric.go:Metric()`, `hooks/requestid.go:RequestID()`, `hooks/auth.go:Auth()`, `hooks/ctx.go:Ctx()`.

### router/ -- Reverse Proxy

The core routing engine built on Go's `httputil.ReverseProxy`:

- `router.go:Server()` -- Creates per-port reverse proxy with Director (pre-routing) and ModifyResponse (post-routing) hooks.
- `request.go:ParseClientRequest()` -- Classifies incoming HTTP into `QueryRequest` (POST /v1/statement), `QueryApiRequest` (DELETE /v1/query), `UiRequest` (GET ui/), or `ApiRequest` (GET v1/info|status).
- `request.go:evaluateRoutingBackend()` -- Two-phase routing: (1) Policy evaluation resolves groups, (2) Group evaluation selects a backend.
- `response.go:ProcessResponse()` -- Extracts query ID from Trino response, asynchronously saves query record for routing continuity.
- `auth.go:AuthHandler()` -- Per-request auth middleware. Checks if port has delegated auth enabled. Supports BasicAuth and Trino custom headers (X-Trino-User/X-Trino-Password). Has a hardcoded exempted users list for legacy compatibility.

### monitor/ -- Backend Health Checker

- `monitor.go:Schedule()` -- Uses gocron scheduler, max 1 concurrent execution.
- `monitor.go:Execute()` -- Fetches all backends, evaluates health, marks healthy/unhealthy via BackendApi.
- `core.go:EvaluateBackendNewState()` -- For each backend: (1) check cron uptime schedule, (2) check cluster reachability via Trino HTTP API, (3) check cluster health via SQL query, (4) measure load from `system.runtime.queries`, (5) compare load against threshold.
- `core.go:computeClusterLoad()` -- Load formula: `(running * 2) + (queued / 3)` where running includes RUNNING+PLANNING+FINISHING+DISPATCHING states and queued includes QUEUED+STARTING states.

### frontend/ -- GopherJS Web UI

Currently disabled (commented out in `main.go`). Built with GopherJS and Vecty. The GUI server would share the API server port.

## Dependencies

| Dependency | Purpose | Failure Impact |
|---|---|---|
| **MySQL 8+** | Persistence for backends, groups, policies, queries, group-backend mappings | Total service failure -- no routing decisions possible |
| **Trino clusters** (backends) | Query execution targets | Individual backend marked unhealthy; traffic routes to remaining healthy backends. If all unhealthy, queries fail. |
| **External auth validation provider** (optional) | Delegated user authentication | Auth failures return 404 to clients. In-memory cache provides brief resilience. |

## Deployment

- Docker container (see `build/docker/prod/Dockerfile`, `build/docker/prod/Dockerfile.razorpay`)
- Kubernetes deployment (referenced in project CLAUDE.md)
- Configuration via TOML files (`config/default.toml`, environment-specific overlays) loaded by Viper
- Environment selected via `APP_ENV` environment variable (maps to config file name)
- Database migrations managed by Goose (`internal/gatewayserver/database/migrations/`)

## Non-Obvious Boundaries

- **The router communicates with the management API via local Twirp RPC calls** -- The reverse proxy servers (`startGatewayServers`) create Twirp protobuf clients pointing at `localhost:{api_port}`. This means routing decisions always go through the full API layer including auth hooks. The router authenticates to its own API server using the configured auth token.

- **Multiple gateway ports are a routing primitive, not just load distribution** -- Each port in `gateway.ports` can have different policies attached (different group routing, different auth delegation, different source headers). This is how the service separates e.g. adhoc vs scheduled query traffic.

- **Query routing continuity depends on the query audit trail** -- When a client polls for query status or cancels a query, the gateway must route that request to the same backend that received the original query. This lookup happens via `QueryApi.FindBackendForQuery()`. If the query record is missing (e.g., async save failed), the routing falls back and may hit the wrong backend.

- **Hardcoded exempted users list in router auth** -- `auth.go:AuthHandler()` contains a hardcoded list of service accounts (capital-scorecard, care, datum, etc.) that bypass auth validation when auth delegation is NOT enabled. This is legacy behavior for services that send BasicAuth but should not be validated against the external provider.

- **Monitor uses the gateway's own API, not direct DB access** -- The monitor subsystem calls `BackendApi` via Twirp to list backends and update health status, rather than accessing the database directly. This means monitor operations go through the same auth and validation path as external API calls.

- **Cluster load computation is approximate** -- The load formula in `core.go:computeClusterLoad()` only factors in query state counts. TODOs exist for incorporating average queue time, CPU load, and active node count, but these are not yet implemented.

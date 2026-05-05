# External Dependencies

What trino-gateway calls, how it connects, and what happens when things break.

---

## 1. Trino Clusters (Backend Health + Query Proxy)

**Purpose:** The gateway proxies SQL traffic to Trino clusters AND actively monitors their health to route traffic away from unhealthy backends.

### 1a. HTTP Health Check — `/v1/info`

| Attribute | Value |
|-----------|-------|
| Transport | HTTP GET |
| Endpoint | `{scheme}://{host}/v1/info` |
| Auth | `X-Trino-User` header (configured user) |
| Timeout | 5s dial, 5s TLS handshake, 2s expect-continue |
| Retry | None — single attempt per monitor cycle |
| Failure mode | **Fail-closed** — backend marked unhealthy on any error |
| Config | `[monitor.trino]` in config TOML |

**What it checks:** `trino.go:IsClusterUp()` calls `/v1/info` and validates:
- Response has non-empty `nodeVersion.version` (rejects malformed responses)
- `coordinator == true` AND `starting == false` (cluster must be fully ready)

**Gotcha — Bug in status code check:** Line 115 of `trino.go:IsClusterUp()` has `!(200 <= resp.StatusCode || resp.StatusCode <= 300)` which is always false (every int satisfies one of those conditions). This means non-2xx responses are NOT caught by the status check — the function relies on JSON parsing to catch bad responses instead.

**Connection pooling:** Each health check creates a fresh `http.Client` with `MaxIdleConns: 3`, `IdleConnTimeout: 10s`. Clients are NOT reused across monitor cycles — a new `TrinoClient` is created per backend per cycle in `core.go:isBackendUp()`.

### 1b. SQL Health Check — Configurable Query

| Attribute | Value |
|-----------|-------|
| Transport | `database/sql` via `trino-go-client` driver |
| DSN format | `{scheme}://{user}:{pass}@{host}?catalog=system&schema=runtime&custom_client={key}` |
| Query (default) | `SELECT 1` |
| Query (prod) | `SHOW SCHEMAS FROM hive` |
| Timeout | Inherits HTTP client timeouts (5s dial) |
| Retry | None |
| Failure mode | **Fail-closed** — backend marked unhealthy |
| Config | `monitor.healthCheckSql` in config TOML |

**Decision: Why two health checks (HTTP + SQL)?**
- HTTP check (`/v1/info`) verifies the coordinator process is up and accepting connections
- SQL check (`trino.go:IsClusterHealthy()`) verifies the query engine actually works end-to-end
- A cluster can pass HTTP check but fail SQL check (e.g., catalog misconfigured, metastore down)
- Both must pass for a backend to be marked healthy

**Decision: Prod uses `SHOW SCHEMAS FROM hive` instead of `SELECT 1`**
- `SELECT 1` only validates the coordinator process
- `SHOW SCHEMAS FROM hive` validates the Hive Metastore connection is also working, which is what real queries need
- Revisit if: health check query latency becomes a bottleneck in monitor cycles

### 1c. SQL Load Query — `system.runtime.queries`

| Attribute | Value |
|-----------|-------|
| Transport | Same `trino-go-client` driver as health check |
| Query | `SELECT state, count(*) FROM system.runtime.queries WHERE user != '{monitor_user}' AND state NOT IN ('FINISHED', 'FAILED') GROUP BY state` |
| Purpose | Compute cluster load score for routing decisions |
| Failure mode | **Fail-closed** — backend marked unhealthy if query fails |

**Load computation** (`core.go:computeClusterLoad()`):
- `load = (running * 2) + (queued * 1) / 3` where running includes RUNNING/PLANNING/FINISHING/DISPATCHING and queued includes QUEUED/STARTING
- Load is compared against per-backend `ThresholdClusterLoad`; if threshold is 0 or load <= threshold, backend is healthy
- Several planned metrics (AvgQueueTimeMs, AvgCpuLoad, ActiveNodes) are stubbed as TODO

### 1d. Query Proxy (Reverse Proxy)

| Attribute | Value |
|-----------|-------|
| Transport | `net/http/httputil.ReverseProxy` |
| Direction | Client -> Gateway -> Trino Backend |
| Timeout | No explicit timeout on proxy transport (uses Go defaults) |
| Failure mode | Returns HTTP 502 Bad Gateway to client |

**Non-obvious:** The reverse proxy `Transport` is set to `nil` in `router.go:Server()`, meaning it uses `http.DefaultTransport`. There are no custom timeouts, retry logic, or circuit breakers on the proxy path — it is a raw pass-through.

---

## 2. MySQL Database

**Purpose:** Stores all gateway state — backends, groups, policies, queries.

| Attribute | Value |
|-----------|-------|
| Transport | TCP via GORM (`gorm.io/driver/mysql`) |
| DSN format | `{user}:{pass}@tcp({host}:{port})/{db}?charset=utf8&parseTime=True&loc=Local` |
| Default dialect | `mysql` (also supports `postgres` via config) |
| Pool — MaxOpen | 5 (default config) |
| Pool — MaxIdle | 5 (default config) |
| Pool — ConnMaxLifetime | 0 (no limit, default config) |
| Failure mode | **Fail-closed** — `boot.init()` calls `log.Fatal` if DB connect fails at startup |
| Migration tool | goose v3 (`cmd/migration/main.go`) |
| Config | `[db]` in config TOML |

**Decision: GORM with specific settings**
- `SkipDefaultTransaction: true` — queries do NOT auto-wrap in transactions (performance over safety)
- `PrepareStmt: true` — prepared statements are cached and reused
- `AllowGlobalUpdate: false` — prevents accidental mass updates
- Debug logging is `Silent` unless `db.Debug = true`, then `Info` level
- Code ref: `db.go:NewDb()`

**Decision: DB init happens in Go `init()` function**
- `boot.go:init()` runs at import time, before `main()`, establishing DB connection
- This means **any import of the `boot` package triggers DB connection** — affects test setup and binary startup
- Revisit if: you need to test packages that import boot without a live DB

**Prometheus integration:** `boot.go:initialize()` registers a `sqlstats` collector that exposes DB connection pool metrics (open connections, idle, wait count, etc.) via the `/metrics` endpoint.

---

## 3. Auth Validation Provider

**Purpose:** External token validation service for delegated authentication on gateway proxy ports.

| Attribute | Value |
|-----------|-------|
| Transport | HTTP POST |
| Endpoint | Configured via `auth.router.delegatedAuth.validationProviderURL` |
| Auth header | `X-Auth-Token: {validationProviderToken}` |
| Request body | `{"email": "{username}", "token": "{password}"}` |
| Response body | `{"ok": true}` or `{"ok": false}` |
| Timeout | **None configured** — uses default `http.Client{}` with no timeout |
| Retry | None |
| Cache | In-memory, keyed by username, TTL from `cacheTTLMinutes` (default 10m) |
| Failure mode | **Fail-closed** — auth error returns HTTP 404 to client |
| Config | `[auth.router.delegatedAuth]` in config TOML |

**Decision: Fail-closed on auth errors, NOT fail-open**
- If the validation provider is unreachable, `auth.go:Authenticate()` returns `(false, err)` which triggers HTTP 404
- This is intentional — better to block users than allow unauthenticated access
- Code ref: `auth.go:AuthHandler()` lines 166-170

**Decision: In-memory cache per context, not shared**
- `auth.go:GetInMemoryAuthCache()` stores the cache in the request context
- Cache is created lazily on first auth check and lives for the lifetime of the context
- Cache key is username, value is password — on cache hit, the validation provider is NOT called
- Revisit if: cache grows unbounded in long-lived contexts

**Gotcha — No HTTP timeout on auth client:** `auth.go:ValidateFromValidationProvider()` creates `&http.Client{}` with zero timeout. If the validation provider hangs, the proxy request hangs indefinitely. This contrasts with the Trino health check client which has explicit 5s timeouts.

**When auth is triggered:** Only when `isAuthDelegated()` returns true, which calls the internal Policy API (`EvaluateAuthDelegationForClient`). If that policy call fails, delegation is assumed disabled and auth falls through to a secondary path with a hardcoded exempted users list.

**Gotcha — Hardcoded exempted users:** When auth delegation is disabled, `auth.go:AuthHandler()` contains a hardcoded list of exempted service accounts (capital-scorecard, care, datum, disputes, etc.) that bypass authentication entirely. Adding new service accounts requires a code change.

---

## 4. Prometheus Metrics (Pull Model)

**Purpose:** Expose application metrics for scraping.

| Attribute | Value |
|-----------|-------|
| Model | **Pull** — exposes HTTP endpoint, scraped by Prometheus |
| Transport | HTTP (dedicated server) |
| Port | `app.metricsPort` (default 8002) |
| Handler | `promhttp.Handler()` at root path |
| Library | `client_golang/prometheus` + `promauto` |
| Failure mode | **N/A** — if Prometheus stops scraping, gateway is unaffected |

**Metrics exposed:**

| Metric | Type | Source |
|--------|------|--------|
| `trino_gateway_monitor_executions_total` | Counter | `monitor/metric.go:initMetrics()` |
| `trino_gateway_monitor_execution_last_run_at` | Gauge | `monitor/metric.go:initMetrics()` |
| `trino_gateway_monitor_execution_seconds_histogram` | Histogram | `monitor/metric.go:initMetrics()` |
| `trino_gateway_monitor_backend_load` | Gauge | `monitor/metric.go:initMetrics()` |
| DB connection pool stats | Various | `boot.go:initialize()` via `sqlstats` |
| Router request/response metrics | Various | `router/metric.go:initMetrics()` |
| Gateway server RPC metrics | Various | `gatewayserver/hooks/metric.go` |

**Non-obvious:** The metrics server is a completely separate HTTP server from the API server (port 8002 vs 8000). This is standard for Kubernetes deployments where the metrics port has different network policies than the application port.

---

## 5. Self-Referential Twirp API (Internal Loopback)

**Purpose:** The monitor and router components call the gateway's own Twirp API over HTTP loopback.

| Attribute | Value |
|-----------|-------|
| Transport | HTTP (Twirp Protobuf) to `http://localhost:{app.port}` |
| Auth | `X-Auth-Key` header with configured token |
| Consumers | Monitor (BackendApi), Router (GroupApi, PolicyApi, BackendApi, QueryApi) |
| Failure mode | Monitor fails to update backend state; Router fails to resolve routing |

**Decision: Why loopback HTTP instead of direct function calls?**
- Monitor and Router are started as separate goroutines in the same process
- They communicate via Twirp RPC over loopback HTTP rather than direct Go function calls
- This keeps the same API contract whether components are co-located or split into separate services
- Code ref: `main.go:startMonitor()` and `main.go:startGatewayServers()` both construct Twirp protobuf clients pointed at localhost
- Trade-off: Added latency per request for routing decisions (policy evaluation, group lookup, query logging all go through HTTP)
- Revisit if: Latency on the routing hot path becomes a concern

---

## Dependency Failure Summary

| Dependency | Fails at | Impact |
|------------|----------|--------|
| MySQL | Startup | **Fatal** — process exits (`log.Fatal`) |
| MySQL | Runtime | Twirp API errors, routing decisions fail, query logs lost |
| Trino cluster (single) | Runtime | That backend marked unhealthy, traffic rerouted to others |
| Trino cluster (all) | Runtime | All backends unhealthy, monitor logs error, no healthy routing targets |
| Auth validation provider | Runtime | Users get HTTP 404 on authenticated ports (fail-closed) |
| Auth validation provider (slow) | Runtime | Proxy requests hang indefinitely (no timeout) |
| Prometheus | Runtime | No impact on gateway operation |
| Self loopback API | Runtime | Monitor cannot update backends; Router cannot resolve routes |

---
sources:
  - rpc/gateway/service.proto
  - cmd/gateway/main.go
  - internal/gatewayserver/hooks/auth.go
extracted_at: 2026-05-05
---

# Service Contracts

Trino Gateway exposes two distinct interfaces: a Twirp RPC management API and a reverse proxy for Trino client traffic. They run on separate ports and have independent auth models.

---

## Interface Overview

| Interface | Port Config | Protocol | Purpose |
|-----------|-------------|----------|---------|
| Twirp Management API | `Config.App.Port` | HTTP + Twirp (Protobuf/JSON) | CRUD for backends, groups, policies, queries |
| Reverse Proxy | `Config.Gateway.Ports` (multiple) | HTTP reverse proxy | Transparent proxy for Trino clients (JDBC, CLI, UI) |
| Prometheus Metrics | `Config.App.MetricsPort` | HTTP | `/metrics` endpoint via `promhttp` |
| Swagger UI | Same as API port | HTTP | Static files at `/admin/swaggerui/` |

---

## 1. Twirp Management API

Base path: `/twirp/`
Proto definition: `rpc/gateway/service.proto`
Registration: `cmd/gateway/main.go:startApiServer()`

### Auth Model (Non-Obvious)

Auth is enforced via two layers working together:

1. **HTTP middleware** (`hooks.go:WithAuth()`) -- extracts `X-Auth-Key` header token and URL path into request context
2. **Twirp hook** (`hooks.go:Auth()`) -- the actual decision point. Checks if URL path contains `/Get` or `/List` and **skips auth entirely** for those. All other methods require the token to match `Config.Auth.Token`.

**Gotcha:** The HealthCheckAPI handler is registered WITHOUT `hooks.WithAuth()` wrapping, so it bypasses auth entirely -- not just for reads but for all methods. The other four services (Backend, Group, Policy, Query) go through `WithAuth`.

**Gotcha:** Auth bypass is pattern-matched on URL path substrings `/Get` and `/List`, not on HTTP method. The `Evaluate*` and `Find*` RPCs require auth even though they are read-only operations. This is because their URL paths don't contain `/Get` or `/List`.

### Auth Summary by RPC

| Service | Method | Auth Required | Why |
|---------|--------|---------------|-----|
| HealthCheckAPI | Check | No | Handler not wrapped with `WithAuth` at all |
| BackendApi | GetBackend | No | Path contains `/Get` |
| BackendApi | ListAllBackends | No | Path contains `/List` |
| BackendApi | CreateOrUpdateBackend | Yes | Write operation |
| BackendApi | DeleteBackend | Yes | Write operation |
| BackendApi | EnableBackend | Yes | Write operation |
| BackendApi | DisableBackend | Yes | Write operation |
| BackendApi | MarkHealthyBackend | Yes | Write operation |
| BackendApi | MarkUnhealthyBackend | Yes | Write operation |
| BackendApi | UpdateClusterLoadBackend | Yes | Write operation |
| GroupApi | GetGroup | No | Path contains `/Get` |
| GroupApi | ListAllGroups | No | Path contains `/List` |
| GroupApi | CreateOrUpdateGroup | Yes | Write operation |
| GroupApi | DeleteGroup | Yes | Write operation |
| GroupApi | EnableGroup | Yes | Write operation |
| GroupApi | DisableGroup | Yes | Write operation |
| GroupApi | EvaluateBackendForGroups | **Yes** | Path is `/Evaluate*`, no Get/List match |
| PolicyApi | GetPolicy | No | Path contains `/Get` |
| PolicyApi | ListAllPolicies | No | Path contains `/List` |
| PolicyApi | CreateOrUpdatePolicy | Yes | Write operation |
| PolicyApi | DeletePolicy | Yes | Write operation |
| PolicyApi | EnablePolicy | Yes | Write operation |
| PolicyApi | DisablePolicy | Yes | Write operation |
| PolicyApi | EvaluateGroupsForClient | **Yes** | Path is `/Evaluate*` |
| PolicyApi | EvaluateAuthDelegationForClient | **Yes** | Path is `/Evaluate*` |
| PolicyApi | EvaluateRequestSourceForClient | **Yes** | Path is `/Evaluate*` |
| QueryApi | GetQuery | No | Path contains `/Get` |
| QueryApi | ListQueries | No | Path contains `/List` |
| QueryApi | CreateOrUpdateQuery | Yes | Write operation |
| QueryApi | FindBackendForQuery | **Yes** | Path is `/Find*`, no Get/List match |

### Internal Self-Consumption

The reverse proxy layer is itself a consumer of the Twirp API. `cmd/gateway/main.go:startGatewayServers()` creates Protobuf clients pointing at `localhost:{App.Port}` and injects `Config.Auth.Token` into the header so the proxy can call authenticated Evaluate/Find endpoints.

Services consumed internally by the proxy:
- `PolicyApi.EvaluateGroupsForClient` -- determine which group handles a request
- `PolicyApi.EvaluateAuthDelegationForClient` -- check if port has delegated auth
- `PolicyApi.EvaluateRequestSourceForClient` -- get `X-Trino-Source` header value
- `GroupApi.EvaluateBackendForGroups` -- pick backend using routing strategy
- `BackendApi.GetBackend` -- resolve backend hostname/scheme for proxying
- `QueryApi.FindBackendForQuery` -- route UI/kill_query requests to correct backend
- `QueryApi.CreateOrUpdateQuery` -- save query metadata after successful routing

### Additional HTTP Endpoints (Non-Twirp)

| Path | Method | Auth | Handler | Purpose |
|------|--------|------|---------|---------|
| `/commit.txt` | GET | No | Inline in `startApiServer()` | Returns `Config.App.GitCommitHash` |
| `/admin/swaggerui/` | GET | No | Static file server (`third_party/swaggerui/`) | OpenAPI/Swagger UI |
| `/` | GET | No | Redirect to `/admin/swaggerui/` | Root redirect |

---

## 2. Reverse Proxy Interface

**Code:** `internal/router/router.go:Server()` creates one `httputil.ReverseProxy` per gateway port.
**Ports:** Configured via `Config.Gateway.Ports` (array of ints). Multiple ports enable different routing policies per port (policies match on `listening_port`).

### Trino Protocol Compatibility

The gateway understands both Trino and Presto header prefixes. `trinoheaders/trino.go:Get()` checks for `X-Trino-{key}` and `X-Presto-{key}`, returning whichever is present first. This means clients using either Trino or legacy Presto JDBC drivers work transparently.

Headers consumed from client requests:
- `X-Trino-User` / `X-Presto-User` -- username (required for query submission)
- `X-Trino-Password` / `X-Presto-Password` -- password (used for delegated auth)
- `X-Trino-Client-Tags` / `X-Presto-Client-Tags` -- used for policy routing
- `X-Trino-Connection-Properties` / `X-Presto-Connection-Properties` -- used for policy routing
- `X-Trino-Transaction-Id` / `X-Presto-Transaction-Id` -- checked but **transactions are not supported** (request rejected unless value is empty or `"NONE"`)
- `X-Trino-Prepared-Statement` / `X-Presto-Prepared-Statement` -- read but currently unused (TODO in code)

Headers set on proxied requests:
- `X-Forwarded-Host` -- set to backend hostname
- `X-Trino-Source` -- set if policy has `set_request_source` configured for the port

### Request Classification

`request.go:ParseClientRequest()` classifies incoming requests into four types:

| Type | Match Condition | Routing Strategy |
|------|----------------|------------------|
| `UiRequest` | `GET` + path contains `ui/` | Looks up query ID from URL, finds backend via `QueryApi.FindBackendForQuery`, routes to backend's `external_url` |
| `ApiRequest` | `GET` + path contains `v1/info` or `v1/status` | Currently returns nil (TODO -- not fully implemented) |
| `QueryRequest` | `POST` | If body contains `kill_query` procedure, extracts query ID and routes to that backend. Otherwise evaluates policies -> groups -> backend. |
| `QueryApiRequest` | `DELETE` + path starts with `/v1/query` | Extracts query ID from path, finds backend via `QueryApi.FindBackendForQuery` |

### Delegated Auth on Proxy (Non-Obvious)

Auth delegation is **per-port**, controlled by policy configuration (`Policy.is_auth_delegated`). When enabled for a port:

1. Gateway extracts credentials via Basic Auth or custom Trino headers
2. Validates against an external auth provider (`Config.Auth.Router.DelegatedAuth.ValidationProviderURL`)
3. Caches valid auth in-memory with configurable TTL (`Config.Auth.Router.DelegatedAuth.CacheTTLMinutes`)
4. Strips `Authorization` header before forwarding to Trino backend

**Gotcha (Exempted Users):** When auth delegation is NOT enabled for a port, there is still a hardcoded list of exempted usernames that bypass authentication entirely. See `auth.go:AuthHandler()`. For all other users with Basic Auth on non-delegated ports, credentials are still validated against the external provider.

### Query Recording (Async, Fire-and-Forget)

After a successful `QueryRequest` routing, the response handler (`response.go:ProcessResponse()`) extracts the query ID from the Trino server's JSON response and saves it via `QueryApi.CreateOrUpdateQuery` in a **goroutine** (fire-and-forget). Failures are logged but do not affect the client response.

---

## 3. Prometheus Metrics

Port: `Config.App.MetricsPort` (separate from API and gateway ports)
Handler: `promhttp.Handler()` (standard Prometheus)
Registration: `cmd/gateway/main.go:startMetricsServer()`

Metrics are registered in:
- `internal/router/metric.go` -- proxy-layer metrics (requests received, routed, response times)
- `internal/gatewayserver/hooks/metric.go` -- Twirp API metrics
- DB connection pool stats via `sqlstats` collector

---

## 4. Consumer Expectations

### For Twirp API consumers (admin tools, scripts, CI/CD)

- **Content-Type:** `application/protobuf` or `application/json` (Twirp supports both)
- **Auth:** Set `X-Auth-Key` header (key name is configurable via `Config.Auth.TokenHeaderKey`) with the shared secret for write operations
- **Read operations** (Get*, List*) do not require auth
- **Evaluate/Find operations** require auth despite being read-only
- **Error format:** Standard Twirp error responses with error codes (`twirp.Unauthenticated`, etc.)

### For Trino clients (JDBC, CLI, BI tools)

- Connect to any configured gateway port
- Use standard Trino or Presto headers -- both prefixes accepted
- `X-Trino-User` is **required** for POST (query submission) and DELETE (query cancel)
- Transactions are **not supported** -- `X-Trino-Transaction-Id` must be empty or `"NONE"`
- If delegated auth is enabled on the port, provide credentials via Basic Auth or `X-Trino-Password`
- Redirects from Trino are rewritten to point back through the gateway hostname (`Config.App.ServiceExternalHostname`)

### For the health monitor (internal)

`cmd/gateway/main.go:startMonitor()` creates a `BackendApi` Protobuf client to `localhost:{App.Port}` with the auth token. It periodically health-checks backends and calls `MarkHealthyBackend` / `MarkUnhealthyBackend`. The interval is `Config.Monitor.Interval`.

---

## 5. What Breaks If Dependencies Are Down

| Dependency | Impact |
|------------|--------|
| MySQL database | All Twirp API calls fail. Proxy routing fails because policy/group/backend evaluation calls the API which queries DB. |
| External auth provider (`ValidationProviderURL`) | Users not in the in-memory cache cannot authenticate on delegated-auth ports. Cached users continue to work until TTL expires. |
| Trino backend cluster | Proxy returns 502 Bad Gateway. Gateway itself stays healthy. Monitoring marks backend unhealthy. |
| Twirp API (localhost) from proxy | Proxy cannot route any requests -- policy evaluation, backend resolution all fail. Returns 400 or 502 to clients. |

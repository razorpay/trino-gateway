---
sources:
  - internal/router/router.go
  - internal/router/request.go
  - internal/router/response.go
  - internal/router/auth.go
  - internal/router/metric.go
extracted_at: 2026-05-05
---

# Router — Reverse Proxy Subsystem

The Router is the HTTP reverse proxy that sits between Trino clients and Trino backends. It classifies incoming requests, evaluates routing policies via self-calls to the gateway's own Twirp API, selects a backend, proxies the request, and records query metadata on the response path. Each configured gateway port spawns an independent `http.Server` with its own `httputil.ReverseProxy`.

---

## Decisions

### D1: Standard HTTP Reverse Proxy (httputil.ReverseProxy) over Protocol-Aware Proxy

**Context:** The gateway needs to intercept Trino client traffic and route it to the correct backend cluster. Trino uses HTTP as its client protocol, but has protocol-specific headers (X-Trino-User, X-Trino-Transaction-Id, etc.).
**Decision:** Use Go's `httputil.ReverseProxy` with Director/ModifyResponse/ErrorHandler hooks rather than building a protocol-aware Trino proxy.
**Alternatives considered:**
- **Protocol-aware proxy:** Parse Trino wire protocol deeply, rewrite query plans — rejected because Trino's client protocol is plain HTTP+JSON; deep parsing adds complexity with no routing benefit.
- **L4 TCP proxy:** Lower overhead — rejected because routing decisions require inspecting HTTP headers (username, client tags, connection properties) and request bodies (query text, kill_query extraction).
**Trade-offs:**
- Gained: Simplicity; leverages Go stdlib; Director hook is sufficient for header-based routing decisions.
- Lost: Cannot inspect or modify Trino-specific protocol semantics (e.g., prepared statement lifecycle) without body parsing. Transaction support is explicitly unsupported (see C3).
**Code:** `router.go:Server()`
**Revisit if:** Trino moves away from HTTP client protocol, or transaction/prepared-statement routing becomes required.

### D2: Self-Calls to Own Twirp API for Routing Decisions

**Context:** Routing decisions (policy evaluation, group selection, backend selection) are implemented as Twirp RPC services. The router needs to invoke these.
**Decision:** The router creates Twirp protobuf clients pointed at `http://localhost:{app_port}` and calls its own API server for every routing decision.
**Alternatives considered:**
- **Direct function calls:** Import service layer directly — rejected because it would couple the router to service internals and bypass API middleware (auth, logging, validation).
- **Separate sidecar service:** — rejected as unnecessary operational complexity for a single-binary deployment.
**Trade-offs:**
- Gained: Clean separation; routing logic goes through the same API path as external callers; API middleware (auth tokens via Twirp headers) applies uniformly.
- Lost: Every proxied request incurs multiple localhost HTTP round-trips (policy eval + group eval + backend lookup + optionally query lookup). This is visible in pre-routing latency metrics.
**Code:** `cmd/gateway/main.go:startGatewayServers()` creates clients; `request.go:evaluateRoutingBackend()` chains the calls.
**Revisit if:** Pre-routing latency becomes a bottleneck; at that point consider direct service-layer calls or caching policy evaluations.

### D3: Async Query Record Saves (Fire-and-Forget Goroutine)

**Context:** After proxying a new query submission to Trino and receiving the response, the gateway needs to record the query ID and backend mapping for future routing of follow-up requests (UI requests, kill_query, DELETE /v1/query).
**Decision:** Save query records in a detached goroutine (`go func() { ... }()`) — fire-and-forget, no error propagation to the client.
**Alternatives considered:**
- **Synchronous save:** Block response until query is recorded — rejected because it adds latency to every query submission response and the client does not need to know about gateway bookkeeping.
- **Background queue/channel:** Buffer writes — rejected as over-engineering for the current scale; the Twirp call to CreateOrUpdateQuery is already a localhost call.
**Trade-offs:**
- Gained: Zero additional latency on the client response path.
- Lost: If the save fails, subsequent UI/kill requests for that query will fail to resolve a backend. Errors are logged but not retried.
**Code:** `response.go:ProcessResponse()` — the `go func()` block.
**Revisit if:** Query record save failures become frequent; add a retry queue or synchronous fallback.

### D4: Hardcoded Exempted Users List in Auth Handler

**Context:** When auth delegation is NOT enabled for a port, BasicAuth users are still validated — except for a hardcoded list of service accounts.
**Decision:** A hardcoded string slice of exempted usernames bypasses authentication entirely (no password check, no validation provider call).
**Alternatives considered:**
- **Config-driven exemption list:** Move to config file — likely the intended future state (there is a `// whacky stuff` comment).
- **Policy-based exemption:** Use the policy API — would require a new policy type.
**Trade-offs:**
- Gained: Quick unblock for service accounts that cannot integrate with the auth provider.
- Lost: Requires code change + redeploy to add/remove exempted users. Security risk: any request with an exempted username and BasicAuth skips all authentication.
**Code:** `auth.go:AuthHandler()` — the `exemptedUsers` slice in the `else` branch.
**Revisit if:** Any new service account needs exemption (currently requires code change), or security audit flags this pattern.

### D5: Context-Based Data Sharing Between Director and ModifyResponse

**Context:** `httputil.ReverseProxy` splits request processing (Director) from response processing (ModifyResponse/ErrorHandler). Data must flow between them.
**Decision:** Use `context.WithValue` on `req.Context()` to share a `ContextSharedObject` containing the parsed client request, timer, and error pointers.
**Alternatives considered:**
- **Pointer sharing:** Prone to synchronization bugs with concurrent requests (noted in code comment).
- **Goroutine + channel:** Cleaner but adds overhead (noted in code comment).
**Trade-offs:**
- Gained: Works with Go's request-scoped context model; each request gets its own context.
- Lost: Must be careful not to modify the `http.Server` context (only `req.Context()`); type assertion can fail silently if context key is wrong.
**Code:** `router.go:handleClientRequest()` sets it; `router.go:extractSharedRequestCtxObject()` retrieves it.

---

## Non-Obvious Constraints

### C1: Dual Header Prefix Support (Trino and Presto)

**Rule:** Every Trino header lookup checks both `X-Trino-{key}` and `X-Presto-{key}` prefixes, returning the first non-empty value.
**Why:** Backward compatibility with clients using the Presto protocol (pre-rename). Some clients like Looker use Presto headers.
**Enforced at:** `trinoheaders/trino.go:Get()` — central header accessor used by all router code.
**Example:** `X-Trino-User` and `X-Presto-User` are both accepted for username extraction.

### C2: kill_query Routing Requires Body Parsing with Regex

**Rule:** When a POST body contains `CALL system.runtime.kill_query(...)`, the gateway extracts the target query ID from the procedure arguments and routes the request to the backend that originally handled that query.
**Why:** kill_query is a stored procedure call, not a Trino API endpoint. The query ID is embedded in SQL text, not in headers. Without extraction, the kill would be routed to a random backend which does not own the query.
**Enforced at:** `request.go:extractQueryId()` with regex matching.
**Example:** `CALL system.runtime.kill_query('20230101_abc123')` extracts `20230101_abc123`, looks up the backend via `QueryApi.FindBackendForQuery`, and routes there.

### C3: Transactions Are Explicitly Unsupported

**Rule:** Any request with a `X-Trino-Transaction-Id` (or X-Presto-) that is not empty or `"NONE"` is rejected with a validation error.
**Why:** The gateway cannot guarantee transaction affinity across requests since it may route different requests to different backends. Looker's Presto client sends `NONE` as a default, so that value is allowed.
**Enforced at:** `request_type.go:QueryRequest.Validate()` and `request_type.go:QueryApiRequest.Validate()`.

### C4: Port-Based Policy Evaluation

**Rule:** Routing policies (group evaluation, auth delegation, request source tagging) are evaluated based on the incoming port number, not any request attribute.
**Why:** Different ports serve different client classes (e.g., interactive users vs. batch ETL). The port is the primary discriminator for which policy rules apply.
**Enforced at:** `request.go:evaluateRoutingBackend()` sends `incomingPort` to `PolicyApi.EvaluateGroupsForClient`; `auth.go:isAuthDelegated()` sends port to `PolicyApi.EvaluateAuthDelegationForClient`.

### C5: UiRequest Uses ExternalUrl; QueryRequest Uses Internal Hostname

**Rule:** When preparing the proxy target, UI requests (`UiRequest`) use `backend.GetExternalUrl()` while query requests (`QueryRequest`, `QueryApiRequest`) use `backend.GetHostname()`.
**Why:** UI requests (Trino Web UI) may need to be routed through an external-facing URL (e.g., load balancer), while query protocol traffic goes directly to the internal Trino coordinator hostname for lower latency.
**Enforced at:** `request.go:prepareReqForRouting()` — the switch on client request type.

### C6: Error Routing via Invalid Host

**Rule:** When request processing fails in the Director, the request URL is set to `http://invalid:8080` — a non-routable host that will trigger the ErrorHandler.
**Why:** `httputil.ReverseProxy.Director` cannot return an error. The only way to signal failure is to set an unroutable target, which causes a connection error that invokes ErrorHandler, where the pre-routing error is extracted from context and returned as an appropriate HTTP status.
**Enforced at:** `router.go:handleClientRequestRoutingError()` sets the invalid host; `router.go:Server()` ErrorHandler checks `preRoutingErr`.

---

## Flow Map

### Request Routing (New Query Submission — POST without existing query ID)

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Happy path (most traffic) | POST to gateway port | `router.go:handleClientRequest()` -> `request.go:ProcessRequest()` -> `request.go:evaluateRoutingBackend()` -> `request.go:prepareReqForRouting()` | DP1, DP2, DP3 | Request proxied to selected Trino backend |
| kill_query procedure (rare) | POST with CALL kill_query | `request.go:extractQueryId()` -> `QueryApi.FindBackendForQuery()` -> `request.go:prepareReqForRouting()` | DP4 | Routed to backend owning the target query |
| Validation failure (rare) | Missing username or query text | `request_type.go:QueryRequest.Validate()` | - | 400 Bad Request via ErrorHandler |
| Backend unreachable (rare) | Trino backend down | ErrorHandler in `router.go:Server()` | DP5 | 502 Bad Gateway |

### Request Routing (Follow-Up — UI, DELETE, or kill_query with known ID)

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| UI request (common) | GET with `ui/` in path | `request.go:ParseClientRequest()` -> `QueryApi.FindBackendForQuery()` -> `request.go:prepareReqForRouting()` | DP4 | Proxied to backend that owns the query |
| Query cancel/DELETE (common) | DELETE /v1/query/{id} | `request.go:ParseClientRequest()` -> `QueryApi.FindBackendForQuery()` -> `request.go:prepareReqForRouting()` | DP4 | Proxied to backend that owns the query |
| Query ID not found (rare) | Unknown query ID | `QueryApi.FindBackendForQuery()` returns error | - | Error propagated, routing fails |

### Response Processing

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Successful query submission (most traffic) | 2xx from Trino | `router.go:handleServerResponse()` -> `response.go:ProcessResponse()` | DP6 | Query record saved async, response forwarded |
| Redirect (rare) | 3xx from Trino | `response.go:handleRedirect()` | - | Location header rewritten to gateway hostname |
| Server error (rare) | 4xx/5xx from Trino | `response.go:ProcessResponse()` | - | Error logged, response forwarded as-is |

### Auth Delegation

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Delegated auth enabled (common) | Policy says port requires auth | `auth.go:AuthHandler()` -> `auth.go:isAuthDelegated()` -> `auth.go:Authenticate()` | DP7, DP8 | Validated via external provider or cache |
| Non-delegated + exempted user (common) | BasicAuth with exempted username | `auth.go:AuthHandler()` | DP9 | Bypasses auth entirely |
| Non-delegated + normal user (common) | BasicAuth with non-exempted username | `auth.go:AuthHandler()` -> `auth.go:Authenticate()` | DP7, DP8 | Validated via external provider or cache |
| No password (rare) | Empty password field | `auth.go:AuthHandler()` | - | 401 Unauthorized |

**Decision Points:**
- **DP1: Request Classification** — HTTP method + URL path determines type (ApiRequest, UiRequest, QueryRequest, QueryApiRequest). Why: Different request types have fundamentally different routing strategies (new query vs. follow-up vs. UI).
- **DP2: Policy Group Evaluation** — `PolicyApi.EvaluateGroupsForClient()` determines which backend groups are eligible based on incoming port, host, connection properties, and client tags. Why: Multi-tenant routing — different client classes get different backend pools.
- **DP3: Backend Selection** — `GroupApi.EvaluateBackendForGroups()` picks a specific backend from the eligible groups. Why: Load distribution across backends within a group.
- **DP4: Query-to-Backend Lookup** — `QueryApi.FindBackendForQuery()` finds which backend originally handled a query. Why: Follow-up requests (UI, cancel, kill) must go to the same backend that owns the query state.
- **DP5: Pre vs Post Routing Error** — ErrorHandler distinguishes pre-routing errors (400) from post-routing errors (502) from server unreachable (502) via `ContextSharedObject` error pointers. Why: Client gets meaningful error codes rather than generic 502 for all failures.
- **DP6: Query Record Creation** — Only `QueryRequest` (new query submissions) trigger async saves; `UiRequest`, `ApiRequest`, `QueryApiRequest` do not. Why: Only new queries need their backend mapping recorded for future follow-up routing.
- **DP7: Auth Cache Check** — In-memory cache with configurable TTL checked before calling external validation provider. Why: Avoid round-trip to auth provider on every request from the same user.
- **DP8: External Auth Validation** — Delegates to an external token validation service (`ValidationProviderURL`). Why: Gateway does not own user credentials; auth is managed by a central identity service.
- **DP9: Exempted User Bypass** — Hardcoded service account list skips all auth. Why: Legacy service accounts that cannot integrate with the auth provider (see D4).

---

## Service Contracts

### Outbound — Router calls Gateway Twirp APIs (localhost)

| API | Method | When Called | What Breaks If Down |
|-----|--------|-------------|---------------------|
| `PolicyApi` | `EvaluateGroupsForClient()` | Every new query submission | All new queries fail to route |
| `PolicyApi` | `EvaluateAuthDelegationForClient()` | Every request (auth check) | Auth defaults to non-delegated (graceful degradation with error log) |
| `PolicyApi` | `EvaluateRequestSourceForClient()` | Every routed request | X-Trino-Source header not set (non-fatal) |
| `GroupApi` | `EvaluateBackendForGroups()` | Every new query submission | All new queries fail to route |
| `BackendApi` | `GetBackend()` | Every routed request | Cannot resolve backend hostname; request fails |
| `QueryApi` | `FindBackendForQuery()` | UI requests, DELETE requests, kill_query, query submissions with extracted ID | Follow-up requests cannot find their backend |
| `QueryApi` | `CreateOrUpdateQuery()` | After successful query submission response | Future follow-up requests for this query will fail (silent — async goroutine) |

### Outbound — Router calls External Services

| Service | When Called | What Breaks If Down |
|---------|-------------|---------------------|
| Auth Validation Provider (`ValidationProviderURL`) | User authentication when auth is delegated or non-exempted | Users cannot authenticate; requests get 404 (error) or 401 |

### Outbound — Router proxies to Trino Backends

| Target | When | What Breaks If Down |
|--------|------|---------------------|
| Trino Coordinator (internal hostname) | Query and API requests | 502 Bad Gateway to client |
| Trino Coordinator (external URL) | UI requests | 502 Bad Gateway to client |

### Inbound — Who calls the Router

| Caller | Protocol | Port Selection |
|--------|----------|----------------|
| Trino CLI / JDBC clients | HTTP POST (query submission), DELETE (cancel) | Configured gateway ports |
| Trino Web UI (via browser/Looker) | HTTP GET (ui/ paths) | Configured gateway ports |
| Health checks / monitoring | HTTP GET /v1/info, /v1/status | Configured gateway ports |

---

## Gotchas

1. **Director cannot return errors.** The `httputil.ReverseProxy.Director` function signature is `func(*http.Request)` — no error return. The workaround is setting `req.URL.Host = "http://invalid:8080"` to force a connection failure that triggers ErrorHandler. This is non-obvious and the error classification logic in ErrorHandler depends on `ContextSharedObject` pointers being correctly set.

2. **Auth cache lives on `*ctx` (server context), not request context.** `auth.go:GetInMemoryAuthCache()` stores the cache in the server-level context via pointer mutation (`*ctx = context.WithValue(...)`). This means the cache is shared across ALL requests on that port. This is intentional but fragile — the code comment in `router.go` warns about this pattern.

3. **Multiple localhost round-trips per request.** A single new query submission can trigger up to 5 localhost Twirp calls: `EvaluateAuthDelegation` + `EvaluateGroupsForClient` + `EvaluateBackendForGroups` + `GetBackend` + `EvaluateRequestSourceForClient`. These are sequential. Monitor `trino_gateway_router_http_pre_routing_delay_ms_histogram` for latency impact.

4. **Redirect rewriting is incomplete.** `response.go:handleRedirect()` has a comment "This needs more testing with looker" and forces `http://` scheme regardless of the original scheme. This may break HTTPS setups.

5. **BasicAuth username mismatch check has different behavior.** When auth IS delegated: mismatch between BasicAuth username and X-Trino-User header returns 401. When auth is NOT delegated: exempted users skip this check entirely, meaning the X-Trino-User header could be anything.

6. **ApiRequest (GET /v1/info, /v1/status) returns nil.** `request.go:ProcessRequest()` returns `(nil, nil)` for ApiRequest — no routing happens. This means health-check endpoints currently do not get proxied to any backend. The `// TODO` comment confirms this is incomplete.

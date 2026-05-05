---
sources:
  - pkg/spine/model.go
  - pkg/spine/db/db.go
  - internal/boot/boot.go
  - internal/gatewayserver/hooks/auth.go
  - internal/config/config.go
  - pkg/config/config.go
extracted_at: 2026-05-05
---

# Technical Patterns - trino-gateway

Non-obvious infrastructure patterns, decisions, and gotchas for the trino-gateway Go service.

---

## Infrastructure Decisions

### D1: Spine ORM Wrapper Over Raw GORM

**Context:** Service needs a DB layer with Razorpay-standard entity conventions (14-char alphanumeric IDs, unix timestamps, validation-on-write).
**Decision:** Custom `pkg/spine/` package wraps GORM with `IModel` interface enforcing `TableName()`, `EntityName()`, `GetID()`, `Validate()`, `SetDefaults()` on every entity.
**Alternatives considered:**
- **Raw GORM:** Rejected because it doesn't enforce Razorpay ID format validation (`^[a-zA-Z0-9]{14}$`) or consistent timestamp validation at the ORM layer.
- **sqlx / raw SQL:** Rejected because the team wanted migration support and association handling that GORM provides.
**Trade-offs:**
- Gained: Every `Create()` call auto-runs `SetDefaults()` then `Validate()` before insert (`spine/repository.go:Create()`). Impossible to insert invalid entities.
- Gained: Transaction propagation via context -- `spine/repository.go:Transaction()` stores `*gorm.DB` in context key, and `db/db.go:Instance()` transparently returns it. Downstream repos automatically participate in the transaction without explicit passing.
- Lost: All model types must implement the full `IModel` interface even when `SetDefaults()` and `Validate()` are no-ops (which they currently are for all gateway models).
**Code:** `pkg/spine/repository.go:Create()`, `pkg/spine/db/db.go:Instance()`
**Revisit if:** Models need more sophisticated validation, or if GORM overhead becomes a concern for high-throughput paths.

**Gotcha -- Transaction via Context:** The transaction DB instance is stored in `context.Value(ContextKeyDatabase)`. If you call a repo method with a context that has a transaction but the method should NOT be in that transaction, you will silently join it. There is no way to opt out except by creating a fresh context.

**Gotcha -- Update selectiveList:** `spine/repository.go:Update()` auto-appends `updated_at` to `selectiveList`. But if you pass an empty selectiveList, GORM updates ALL fields (full struct update). This is by design but non-obvious.

**Gotcha -- SkipDefaultTransaction:** GORM config sets `SkipDefaultTransaction: true` (`db/db.go:NewDb()`). Single creates/updates are NOT wrapped in implicit transactions. You must explicitly use `repo.Transaction()` if you need atomicity.

### D2: Global Boot State via `init()`

**Context:** Application needs config, DB, and logger available before any handler code runs.
**Decision:** `internal/boot/boot.go` uses Go's `init()` function to load config and open DB connection into package-level vars (`boot.Config`, `boot.DB`). These are then accessed globally throughout the codebase.
**Alternatives considered:**
- **Dependency injection:** Would require threading config/DB through constructors. Rejected for simplicity in a small service.
- **Singleton with lazy init:** Rejected because DB connection failure should be fatal at startup, not deferred.
**Trade-offs:**
- Gained: Any package can import `boot` and access `boot.Config` or `boot.DB` directly.
- Lost: Testing is harder -- no way to inject mock DB without modifying package vars. Global state makes parallel test execution risky.
**Code:** `internal/boot/boot.go:init()`
**Revisit if:** Service grows to need test isolation or multiple DB connections per request.

**Gotcha -- Double init:** `boot.init()` runs config load AND DB connection. Then `boot.InitApi()` calls `initialize()` which re-initializes the logger and registers Prometheus DB stats collector. The `init()` already called `InitLogger()` once. If you add side-effecting code to `InitLogger()`, it runs twice.

**Gotcha -- APP_ENV default:** If `APP_ENV` is unset, it defaults to `"dev"`, which loads `config/dev.toml` (or whatever file matches). In production containers, `APP_ENV` MUST be set or you get dev config silently.

### D3: Twirp Over gRPC

**Context:** Service needs RPC framework for admin API (CRUD for backends, groups, policies, queries) and internal communication between router and API server.
**Decision:** Twirp (Twitch's RPC framework) over protobuf, not gRPC.
**Alternatives considered:**
- **gRPC:** Rejected because Twirp generates simple HTTP 1.1 handlers that work with standard `net/http` mux. The gateway reverse proxy is `httputil.ReverseProxy` which works over HTTP 1.1. gRPC would require HTTP/2 and a separate proxy approach.
- **REST/JSON:** Rejected because protobuf gives type-safe generated clients used by both the monitor and the router to call the API server internally.
**Trade-offs:**
- Gained: Router and Monitor use generated protobuf clients (`gatewayv1.NewBackendApiProtobufClient()`) to call the API server over localhost. Type-safe internal communication with zero extra infra.
- Gained: Twirp hooks system (`twirp.ChainHooks()`) provides clean middleware for auth, metrics, request ID, context enrichment.
- Lost: No streaming support (Twirp is unary-only). No HTTP/2 multiplexing.
**Code:** `cmd/gateway/main.go:twirpHooks()`, `cmd/gateway/main.go:startApiServer()`
**Revisit if:** Need streaming responses (e.g., live query progress) or if moving to a service mesh that expects gRPC.

---

## Configuration Pattern

### TOML Layering with Viper

Config loads in two phases via `pkg/config/config.go:Load()`:
1. Load `config/default.toml` (base values for all environments)
2. Load `config/{APP_ENV}.toml` (environment override, merges on top)

**Env var override:** Viper's `AutomaticEnv()` is enabled with prefix `TRINO-GATEWAY` (uppercased) and dot-to-underscore replacement. So config key `db.ConnectionConfig.url` can be overridden by env var `TRINO-GATEWAY_DB_CONNECTIONCONFIG_URL`.

**Gotcha -- WORKDIR:** Config path resolution in `pkg/config/config.go:NewDefaultOptions()` uses `$WORKDIR/config/` if `WORKDIR` is set, otherwise falls back to `runtime.Caller()` relative path (`../../config/`). The `runtime.Caller` path is resolved at **build time**, not runtime. If the binary is moved to a different directory without setting `WORKDIR`, config loading fails silently with a "file not found" error. Containers MUST set `WORKDIR`.

**Gotcha -- Env-specific files are sparse:** `config/dev-docker.toml` only overrides 4 keys. Every other value comes from `default.toml`. If you add a new config key, you MUST add the default to `default.toml`, not to environment files.

---

## Twirp Hook Chain

### Ordering: Metric -> RequestID -> Auth -> Ctx

Defined in `cmd/gateway/main.go:twirpHooks()`:
```
twirp.ChainHooks(hooks.Metric(), hooks.RequestID(), hooks.Auth(), hooks.Ctx())
```

**Why this order matters:**

| Hook | Phase | What it does |
|------|-------|-------------|
| Metric | `RequestReceived` | Marks request start time in context (must be first to capture full duration) |
| Metric | `RequestRouted` | Increments `requests_received_total` counter |
| Metric | `ResponseSent` | Records response duration and status |
| RequestID | `RequestRouted` | Reads `X-Request-ID` from context (set by HTTP middleware), generates one if missing |
| Auth | `RequestReceived` | Checks auth token -- **skips auth for `/Get` and `/List` paths** |
| Ctx | `RequestRouted` | Enriches context with logger containing reqId, method, service, package |

**Gotcha -- Two-layer auth:** Auth happens in TWO places:
1. **HTTP middleware layer** (`hooks.WithAuth()`) -- extracts token from header into context, wraps the Twirp handler
2. **Twirp hook layer** (`hooks.Auth()`) -- validates the token in `RequestReceived`

Both are needed. The HTTP middleware puts the token INTO context; the Twirp hook reads it FROM context. If you add a new API endpoint and forget to wrap it with `hooks.WithAuth()` in the mux, the Twirp auth hook will see an empty token but will PASS if the path contains `/Get` or `/List`.

**Gotcha -- Read endpoints are unauthenticated:** `hooks/auth.go:Auth()` skips auth for any path containing `/Get` or `/List`. This is a substring check (`strings.Contains`), not an exact match. A path like `/CustomGetterEndpoint` would also bypass auth.

---

## Router Architecture

### Self-Calling Pattern

The gateway process runs MULTIPLE HTTP servers in the same process:
- **API server** (port from `app.port`, default 8000) -- Twirp RPC handlers for CRUD
- **Gateway proxy servers** (ports from `gateway.ports`, default [8080, 8081]) -- `httputil.ReverseProxy` for Trino traffic
- **Metrics server** (port from `app.metricsPort`, default 8002) -- Prometheus metrics

**The proxy servers call the API server over localhost HTTP.** See `cmd/gateway/main.go:startGatewayServers()` -- it creates protobuf clients pointing at `http://localhost:{app.port}`. The monitor does the same thing (`cmd/gateway/main.go:startMonitor()`).

**Why:** This keeps routing logic (policy evaluation, group resolution, backend selection) in the Twirp API layer, reusable by both external admin clients and the internal reverse proxy. The proxy doesn't directly access the DB.

**Gotcha -- Auth token for internal calls:** The proxy and monitor set the auth token header on their internal HTTP clients (`boot.Config.Auth.TokenHeaderKey` = `boot.Config.Auth.Token`). If you change the auth token in config, both the API server AND the internal clients pick it up from the same config. But if you add token rotation, the internal clients would need to be updated too.

### Context Sharing Between Director and ModifyResponse

`router/router.go` documents this decision in a code comment. The `httputil.ReverseProxy` has a `Director` (pre-routing) and `ModifyResponse` (post-routing) that need to share state. They use `context.WithValue` on the **request context** (not the server context) via `ContextSharedObject`. This is critical -- using the server context would leak state across requests.

### Trino Header Compatibility

`internal/router/trinoheaders/trino.go:Get()` checks BOTH `X-Presto-*` and `X-Trino-*` header prefixes. This is because older Presto clients (e.g., Looker's Presto connector) send `X-Presto-*` headers while newer Trino clients send `X-Trino-*`. The gateway is transparent to both.

**Gotcha -- Transaction rejection:** `request_type.go:QueryRequest.Validate()` and `QueryApiRequest.Validate()` reject any request with a transaction ID that isn't empty or `"NONE"`. Transactions are explicitly unsupported. Looker sends `X-Presto-Transaction-Id: NONE` which is allowed.

---

## Cross-System Contracts

### Delegated Auth Contract

Router auth delegates to an external validation provider (`auth.router.delegatedAuth.validationProviderURL`).

**API contract (outbound):**
- POST to validation URL with `{"email": "<username>", "token": "<password>"}` and header `X-Auth-Token: <validationProviderToken>`
- Expected response: `{"ok": true}` or `{"ok": false}`

**In-memory cache:** Auth results are cached in a process-level in-memory map (`router/auth.go:GetInMemoryAuthCache()`) with TTL from `auth.router.delegatedAuth.cacheTTLMinutes`. Cache is stored in the server context (not request context), so it persists across requests but is lost on restart.

**Gotcha -- Cache is per-port:** The auth cache is initialized lazily and stored in the server context. Each gateway port gets its own server context, so each port has its own auth cache. A user authenticated on port 8080 must re-authenticate on port 8081.

### Exempted Users List

`router/auth.go:AuthHandler()` has a **hardcoded list of exempted usernames** that bypass auth validation when auth delegation is NOT enabled. These are service accounts (`capital-scorecard`, `care`, `datum`, `settlements`, etc.). This list is NOT in config -- it's in code. Adding/removing exempted users requires a code change and redeploy.

### Trino Health Check Contract

Monitor checks backend health via:
1. `TrinoClient.IsClusterUp()` -- HTTP call to Trino coordinator
2. `TrinoClient.IsClusterHealthy()` -- secondary health check
3. `TrinoClient.RunQuery()` -- executes `SELECT state, count(*) FROM system.runtime.queries WHERE user != '<monitor-user>' AND state NOT IN ('FINISHED', 'FAILED') GROUP BY state` to compute cluster load

**Load formula** (`monitor/core.go:computeClusterLoad()`): `load = (running * 2) + (queued / 3)` where `running` includes RUNNING + PLANNING + FINISHING + DISPATCHING states, and `queued` includes QUEUED + STARTING states. If `load > backend.ThresholdClusterLoad`, backend is marked unhealthy (unless threshold is 0, which disables load-based health).

### Backend Uptime Schedule

Backends have a `UptimeSchedule` field (cron expression). The monitor evaluates whether current time falls within the cron schedule. If outside schedule OR cron is invalid, backend is marked unhealthy. This enables time-based cluster scaling (e.g., dev clusters only active during business hours).

---

## GopherJS Frontend (Legacy)

`internal/frontend/main.go` compiles Go to JavaScript using GopherJS + Vecty framework. The comment in `config/default.toml` notes "gui & twirp app need to be on same port for now." The frontend server is currently **commented out** in `cmd/gateway/main.go` and replaced with a redirect to SwaggerUI. The frontend code exists but is not actively served.

---

## ID Format Convention

All spine model IDs must match `^[a-zA-Z0-9]{14}$` (Razorpay standard 14-char alphanumeric ID). This is validated by `pkg/spine/datatype/validation.go:IsRZPID()` on every `Create()`. IDs are NOT auto-generated by spine -- callers must provide them. Request IDs (for logging) use `xid` which generates globally unique, sortable IDs but these are different from entity IDs.

---

## DB Resolver Pattern

`pkg/spine/db/db.go` supports three DB targets via GORM's `dbresolver`:
- **Primary/Source** -- default for all writes
- **Replicas** -- read queries auto-routed here (random policy) if configured
- **WarmStorage** -- accessed explicitly via `repo.WarmStorageDBInstance()` using named resolver `"warm_storage"`

Currently, only the primary connection is configured. Replica and warm storage support exists in the framework but is unused in this service's config.

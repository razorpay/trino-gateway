# Policy

A Policy is a routing rule that maps incoming client requests to Trino backend Groups. Each policy binds a single rule (a type+value pair like "requests arriving on port 8443") to a target Group, with optional fallback. Policies are the core mechanism for multi-tenant routing -- different clients reach different Trino clusters based on how they connect.

---

## Decisions

### D1: Four Rule Types for Multi-Dimensional Routing

**Context:** The gateway needs to route different clients (BI tools, API services, ad-hoc users) to appropriate Trino clusters, but clients connect through various means -- some via dedicated ports, some via shared ports with distinguishing headers.

**Decision:** Four rule types defined as a protobuf enum: `header_connection_properties` (0), `header_client_tags` (1), `header_host` (2), `listening_port` (3). Each policy has exactly one rule type+value pair.

**Alternatives considered:**
- **Single composite rule:** One policy with multiple conditions -- rejected because it creates combinatorial complexity in rule management and makes it hard to reason about which rules match.
- **Regex-based matching:** Pattern matching on headers -- rejected because exact-match lookups via DB queries are simpler and faster than loading all policies and evaluating regex in-memory.

**Trade-offs:**
- Gained: Simple DB-level filtering (`WHERE rule_type = X AND rule_value = Y`), each policy is independently manageable
- Lost: Cannot express complex conditions (e.g., "port 8443 AND client_tags=etl") in a single policy -- requires multiple policies whose results get intersected

**Code:** `service.proto:Policy.Rule.RuleType` enum, `core.go:EvaluateGroupsForClient()`

**Revisit if:** Need for compound rules (AND/OR logic within a single policy) becomes common, or rule count grows unmanageable.

### D2: Set Intersection Semantics for Multi-Rule Evaluation

**Context:** When a request matches policies across multiple rule types (e.g., both a `listening_port` policy and a `header_client_tags` policy), the system needs to decide which group(s) to route to.

**Decision:** Evaluate each rule type independently, collect matching group IDs into sets, then intersect all non-nil sets. A nil set (no policies of that type matched) acts as a wildcard -- it does not constrain the result. If ALL sets are nil (no policies match at all), the result is empty (falls through to default group at the Group layer).

**Alternatives considered:**
- **First-match-wins with priority:** Ordered rule evaluation stopping at first match -- rejected because it cannot express "must match on BOTH port AND client tags."
- **Union semantics:** Combine all matching groups -- rejected because it would be too permissive; a request matching a broad port rule would gain access to groups meant only for specific tagged clients.

**Trade-offs:**
- Gained: Policies across different rule types act as an AND filter -- more precise routing
- Lost: Intersection can produce empty results if policies are misconfigured (e.g., port policy maps to group A, tag policy maps to group B, intersection is empty)

**Code:** `core.go:EvaluateGroupsForClient()`, `core.go:setIntersection()`

**Revisit if:** Empty intersection results cause routing failures in production and operators need union/fallback semantics.

### D3: Auth Delegation as a Per-Policy Flag, Port-Scoped

**Context:** Some Trino listeners should enforce authentication (e.g., user-facing ports) while others should not (e.g., internal service ports). The gateway delegates auth to an external validation provider.

**Decision:** `IsAuthDelegated` is a boolean on each Policy, but auth delegation is evaluated only by `listening_port` rule type. If ANY enabled policy for a port has `IsAuthDelegated=true`, auth is enforced for that port.

**Alternatives considered:**
- **Global auth toggle:** Single config flag -- rejected because different ports serve different trust boundaries.
- **Separate auth configuration entity:** Decoupled from policies -- rejected to keep routing and auth configuration co-located (one place to configure a listener's behavior).

**Trade-offs:**
- Gained: Auth behavior is configured alongside routing, reducing configuration drift
- Lost: Auth delegation is coupled to the policy entity even though it is conceptually a port-level concern, not a routing rule concern

**Code:** `core.go:EvaluateAuthDelegation()`, `auth.go:isAuthDelegated()`, `auth.go:AuthHandler()`

**Revisit if:** Auth requirements become more granular (per-user, per-group) rather than per-port.

### D4: CreateOrUpdate (Upsert) Instead of Separate Create and Update

**Context:** Policy management via API.

**Decision:** Single `CreateOrUpdatePolicy` endpoint that checks if a policy with the given ID exists. If yes, updates it; if no, creates it. The caller always provides the full policy definition including the ID.

**Alternatives considered:**
- **Separate Create/Update endpoints:** Standard REST pattern -- rejected for operational simplicity; callers can declaratively set desired state without checking existence first.

**Trade-offs:**
- Gained: Idempotent configuration pushes (can replay the same policy definition safely)
- Lost: No protection against accidental overwrites; no distinction between "I want to create a new policy" vs "I want to update an existing one"

**Code:** `core.go:CreateOrUpdatePolicy()`

**Revisit if:** Need for optimistic concurrency control or audit trails distinguishing creation from updates.

---

## Non-Obvious Constraints

### Nil-Set-As-Wildcard in Intersection

**Rule:** In `EvaluateGroupsForClient`, if no policies exist for a given rule type (the matched set is nil/empty after query), that dimension is treated as a wildcard and does not constrain the intersection.

**Why:** Without this, adding the first policy of any rule type would break all existing routing -- previously unconstrained dimensions would suddenly produce empty intersections.

**Enforced at:** `core.go:setIntersection()` -- nil input returns the other set unchanged.

**Examples:**
- Only `listening_port` policies exist: hostname/tags/connection_properties sets are all nil, so the result equals the listening_port set alone.
- A `listening_port` policy maps to group A, a `header_client_tags` policy maps to group A: intersection = {A}. If the tags policy mapped to group B instead: intersection = {} (empty, no group matches).

### FallbackGroupId Defaults to Config Value

**Rule:** If `FallbackGroupId` is nil at creation time, it is set to `boot.Config.Gateway.DefaultRoutingGroup`.

**Why:** Ensures every policy has a fallback path so that group evaluation at the Group layer can always resolve a backend even if the primary group is unhealthy.

**Enforced at:** `core.go:CreateOrUpdatePolicy()` -- nil check with config default.

### Auth Delegation Fails Open

**Rule:** If the `EvaluateAuthDelegationForClient` call fails (error from gateway API), the router assumes auth delegation is disabled and proceeds without authentication.

**Why:** Availability over security for the proxy layer -- a gateway API outage should not block all queries. The downstream Trino cluster may still enforce its own auth.

**Enforced at:** `auth.go:isAuthDelegated()` -- logs error, returns false.

### SetRequestSource Applies Only to First Matching Policy

**Rule:** `EvaluateRequestSource` returns the `SetRequestSource` value from `res[0]` -- the first matching policy for the port. If multiple listening_port policies exist for the same port with different source values, only the first one's value is used.

**Why:** Non-deterministic if multiple policies match. This is likely a simplification that works because in practice each port has at most one source label.

**Enforced at:** `core.go:EvaluateRequestSource()` -- `res[0].SetRequestSource`.

### SetRequestSource Sets X-Trino-Source Header

**Rule:** When a policy's `SetRequestSource` is non-empty, the router sets the `X-Trino-Source` HTTP header on the proxied request to that value.

**Why:** Allows Trino to identify the source/origin of queries for resource management, query prioritization, and cost attribution -- without the client needing to set this header.

**Enforced at:** `request.go:prepareReqForRouting()` -- `req.Header.Set("X-Trino-Source", s)`.

### Exempted Users Bypass Auth When Delegation Is Off

**Rule:** When auth delegation is NOT enabled for a port, a hardcoded list of service accounts (e.g., "capital-scorecard", "care", "settlements") bypass the username-match check between BasicAuth and X-Trino-User header.

**Why:** Legacy service accounts that send BasicAuth credentials but may not set the X-Trino-User header consistently. This is technical debt.

**Enforced at:** `auth.go:AuthHandler()` -- hardcoded `exemptedUsers` slice.

---

## Flow Map

### Evaluate Groups for Client (Request-to-Group Matching)

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Happy path (most traffic) | New query POST | `request.go:evaluateRoutingBackend()` -> `server.go:EvaluateGroupsForClient()` -> `core.go:EvaluateGroupsForClient()` | DP1, DP2 | List of group IDs returned |
| No policies match (common) | Request with no matching rules | `core.go:EvaluateGroupsForClient()` | DP2 | Empty list; Group layer falls back to default group |
| Single dimension match (common) | Only port policies configured | `core.go:EvaluateGroupsForClient()` -> `core.go:setIntersection()` | DP1 | Other dimensions are nil (wildcard), result = port-matched groups |

**Decision Points:**
- **DP1: Nil-set wildcard** -- If a rule type has no matching policies, its group set is nil. `setIntersection()` treats nil as "any group." Why: prevents newly-added rule types from breaking existing routing.
- **DP2: Intersection produces empty set** -- All rule types returned non-nil sets but their intersection is empty. Why: misconfigured policies (no group satisfies all constraints simultaneously). Result: empty group list passed to Group layer which will use the default routing group.

### Evaluate Auth Delegation

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Auth delegated (some ports) | Every HTTP request | `auth.go:AuthHandler()` -> `auth.go:isAuthDelegated()` -> `server.go:EvaluateAuthDelegationForClient()` -> `core.go:EvaluateAuthDelegation()` | DP3 | Auth enforced via external provider |
| Auth not delegated (most traffic) | Every HTTP request | `auth.go:AuthHandler()` -> `auth.go:isAuthDelegated()` | DP3 | Pass-through (with exempted-user logic) |
| Gateway API error (rare) | Policy API unreachable | `auth.go:isAuthDelegated()` | DP4 | Fails open, no auth enforced |

**Decision Points:**
- **DP3: Any matching policy has IsAuthDelegated=true** -- Looks up enabled `listening_port` policies for the router's port where `is_auth_delegated=true`. If any exist, returns true. Why: auth is a port-level concern; one policy opting in is sufficient.
- **DP4: Fail-open on error** -- If policy evaluation fails, auth delegation returns false. Why: availability over security at the proxy layer; Trino clusters may enforce their own auth.

### Evaluate Request Source

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Source label configured (some ports) | Request routing preparation | `request.go:prepareReqForRouting()` -> `server.go:EvaluateRequestSourceForClient()` -> `core.go:EvaluateRequestSource()` | DP5 | X-Trino-Source header set |
| No source configured (most traffic) | Request routing preparation | `core.go:EvaluateRequestSource()` | DP5 | No header set, empty string returned |

**Decision Points:**
- **DP5: First-match wins** -- Takes `SetRequestSource` from `res[0]`. Why: simplified assumption that each port has at most one source label. Non-deterministic if multiple policies match the same port with different values.

### Policy CRUD

| Flow | Trigger | Key Functions | Decision Points | Outcome |
|------|---------|---------------|-----------------|---------|
| Create or update | API call | `server.go:CreateOrUpdatePolicy()` -> `core.go:CreateOrUpdatePolicy()` | DP6 | Policy upserted in DB |
| Enable/Disable | API call | `server.go:EnablePolicy()` / `server.go:DisablePolicy()` -> `repo/policy.go:Enable()` / `repo/policy.go:Disable()` | DP7 | Policy toggled |
| Delete | API call | `server.go:DeletePolicy()` -> `repo/policy.go:Delete()` | -- | Policy removed |

**Decision Points:**
- **DP6: Upsert by ID** -- Checks if ID exists; updates if found, creates if not. Why: enables declarative, idempotent configuration.
- **DP7: Idempotency guard** -- Enable fails if already enabled; Disable fails if already disabled. Why: prevents silent no-ops that might mask configuration errors. (Note: error message says "Already active" for both cases -- likely a copy-paste bug in `repo/policy.go:Disable()`.)

---

## Service Contracts

### Who Calls Policy (Consumers)

| Consumer | Calls | Contract | Breaks If |
|----------|-------|----------|-----------|
| **Router** (`router/request.go`) | `EvaluateGroupsForClient` | Sends port, host, connection_properties, client_tags. Expects list of group IDs (may be empty). | Policy API down: routing fails for new queries. |
| **Router** (`router/auth.go`) | `EvaluateAuthDelegationForClient` | Sends incoming port. Expects boolean. | Policy API down: fails open (no auth enforced). |
| **Router** (`router/request.go`) | `EvaluateRequestSourceForClient` | Sends incoming port. Expects string (may be empty). | Policy API down: request source header not set, error propagated. |
| **Admin API** (Twirp clients) | CRUD operations | Full Policy proto with ID, Rule, Group, FallbackGroup, flags. | N/A (management plane). |

### Who Policy Calls (Dependencies)

| Dependency | Relationship | Contract | Breaks If |
|------------|-------------|----------|-----------|
| **Group** entity | `GroupId` and `FallbackGroupId` are string references to Group IDs | No FK enforcement at application layer -- policy stores group ID strings. Group must exist when router evaluates backend. | Group deleted while policy references it: `EvaluateBackendForGroups` at the Group layer fails. |
| **Config** (`boot.Config.Gateway.DefaultRoutingGroup`) | Used as fallback when `FallbackGroupId` is nil at creation | Config value must name a valid Group. | Missing config or invalid group name: policies created without explicit fallback route to non-existent group. |

---

## Gotchas

- **Validation is commented out.** `validation.go` is entirely commented out. There is no runtime validation of policy rule types or values. Invalid rule types will be stored but silently never match during evaluation.
- **Disable error message is wrong.** `repo/policy.go:Disable()` returns "Already active" when the policy is already disabled -- copy-paste from `Enable()`.
- **No referential integrity for GroupId.** Policies reference Groups by string ID with no foreign key or application-level check. Creating a policy pointing to a non-existent group succeeds silently.
- **Auth exempted users list is hardcoded.** The list of service accounts that bypass auth checks in `auth.go:AuthHandler()` is a hardcoded string slice, not configurable. Adding or removing exemptions requires a code change and deploy.
- **EvaluateRequestSource is non-deterministic with multiple matches.** If multiple `listening_port` policies exist for the same port value, `res[0]` is returned -- but the ordering from the DB query is not guaranteed.

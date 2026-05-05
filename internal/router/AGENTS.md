# router — Reverse Proxy Engine

Routes Trino client queries to backend clusters via `httputil.ReverseProxy`. Each gateway port runs an independent proxy instance.

## Request Flow

1. `router.go:Director()` → classifies request → evaluates policy → selects backend
2. Proxy forwards to selected Trino backend
3. `router.go:ModifyResponse()` → extracts query ID → saves audit record async

## Key Rules

- Router calls its own Twirp API via localhost — never bypass with direct DB access.
- Request types: `QueryRequest` (POST /v1/statement), `QueryApiRequest` (DELETE /v1/query), `UiRequest` (GET /ui/*), `ApiRequest` (GET /v1/info — unimplemented).
- Transactions rejected: `X-Trino-Transaction-Id` other than empty/"NONE" returns 400.
- Both `X-Trino-*` and `X-Presto-*` headers accepted (backward compat).
- Query records saved in fire-and-forget goroutine — failures don't block clients.
- Director errors communicated via invalid host trick (`response-error-{code}-{msg}`).
- `auth.go` has hardcoded exempted users list — code change required to update.
- Auth delegation fails open on error (availability over security).

## Files

| File | Purpose |
|------|---------|
| `router.go` | ReverseProxy setup with Director/ModifyResponse/ErrorHandler |
| `request.go` | Request classification and backend resolution |
| `response.go` | Response processing and async query record creation |
| `auth.go` | Auth delegation, exempted users, BasicAuth handling |
| `metric.go` | Prometheus counters/histograms for routing |
| `trinoheaders/` | Trino HTTP header constants |

For full domain docs: `.agents/skills/repo-skill/modules/domain/router.md`

---
paths:
  - "internal/router/**"
---

## Router Domain Rules

- Router calls its own Twirp API via localhost for routing decisions (PolicyApi, GroupApi, BackendApi, QueryApi). This is intentional — do not bypass with direct DB access.
- Auth token for self-calls uses `boot.Config.Auth.Token` — if this is wrong, routing breaks silently.
- Request types: UiRequest (browser), QueryRequest (Trino SQL), QueryApiRequest (kill/status), ApiRequest (v1/info — currently unimplemented, returns nil).
- Transactions are rejected: any `X-Trino-Transaction-Id` other than empty/"NONE" returns 400.
- Both `X-Trino-*` and `X-Presto-*` headers are accepted (backward compatibility).
- Query records are saved asynchronously via goroutine — failures don't affect client responses but create audit gaps.
- Hardcoded exempted users list in `auth.go:AuthHandler()` bypasses external auth validation — requires code changes to update.
- Director errors are communicated to ModifyResponse via invalid host trick (`response-error-{code}-{msg}`) since httputil.ReverseProxy doesn't support error return from Director.

For full context: `.agents/skills/repo-skill/modules/domain/router.md`

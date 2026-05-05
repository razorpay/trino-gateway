---
paths:
  - "internal/gatewayserver/policyApi/**"
  - "internal/gatewayserver/models/policy.go"
  - "internal/gatewayserver/repo/policy.go"
---

## Policy Domain Rules

- Four rule types only: `header_connection_properties`, `header_client_tags`, `header_host`, `listening_port`. No regex or composite rules.
- Policy evaluation uses set intersection semantics: non-nil sets are AND'd, nil sets act as wildcards.
- Auth delegation is per-policy and only works with `listening_port` rule type.
- Auth delegation fails OPEN on error (availability > security) — be aware of this design choice.
- FallbackGroupId defaults to gateway config value when nil — not a hard null.
- SetRequestSource maps to `X-Trino-Source` header for Trino cost attribution. First match wins (non-deterministic with multiple policies per port).
- GroupId references use string IDs with no FK constraint — no referential integrity enforcement.
- Validation in `validation.go` is entirely commented out.

For full context: `.agents/skills/repo-skill/modules/domain/policy.md`

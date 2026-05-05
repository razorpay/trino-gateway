# BUILD_CHECKLIST - Trino Gateway

**Created:** 2026-05-05
**Status:** Complete
**Total Tasks:** 14

## Core
- [x] core/boundaries.md - Service boundaries, ownership, and responsibilities (Trino load balancer/proxy)
- [x] core/quick-ref.md - Common operations, env setup, debugging quick reference, Makefile targets

## Domain Entities
- [x] domain/backend.md - Trino cluster backend: health monitoring, load tracking, enable/disable, routing strategies
- [x] domain/group.md - Routing groups: backend selection strategies (LEAST_LOAD, ROUND_ROBIN, RANDOM), GroupBackendsMapping
- [x] domain/policy.md - Routing policies: rule types (header_connection_properties, header_client_tags, header_host, listening_port), auth delegation, fallback groups
- [x] domain/query.md - Query audit trail: routing decision tracking, backend/group resolution logging
- [x] domain/router.md - Reverse proxy subsystem: request classification (ApiRequest, UiRequest, QueryRequest), backend selection, response modification

## Technical
- [x] technical-patterns.md - Non-obvious infrastructure patterns: spine model framework, Twirp hook chain, auth delegation cache, monitor scheduling, config layering

## Integration
- [x] integration/service-contracts.md - Twirp RPC APIs exposed (BackendApi, GroupApi, PolicyApi, QueryApi, HealthCheckApi), Swagger UI
- [x] integration/external-deps.md - Trino cluster HTTP/SQL monitoring, MySQL/PostgreSQL via GORM, auth validation provider, Prometheus metrics

## Rules Generation
- [x] rules/ - Generate .claude/rules/*.md files from extracted domain knowledge

## Finalization
- [x] SKILL.md - Progressive disclosure rules for repo-skill
- [x] .agent-ready-version - Version tracking
- [x] **AGENTS.md (root)** - Generate with Skills Index (MUST BE LAST)

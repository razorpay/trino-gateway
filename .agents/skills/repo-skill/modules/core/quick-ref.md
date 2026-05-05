# Quick Reference - trino-gateway

Go 1.22 service. Twirp RPC framework. MySQL 8+ backend.

---

## Build & Run

### Makefile Targets

| Target | What it does |
|--------|-------------|
| `make setup` | Installs protoc-gen-go, gopherjs, protoc-gen-openapiv2, protoc-gen-twirp via `scripts/setup.sh` |
| `make build` | Runs protoc compilation (generates Twirp stubs + OpenAPI spec) then vendors deps via `scripts/compile.sh` |
| `make build-frontend` | Compiles Go frontend to JS via gopherjs (slow -- skip if no frontend changes) |
| `make dev-docker-up` | Starts dev environment via `build/docker/dev/docker-compose.yml` |
| `make dev-docker-down` | Tears down dev docker environment |
| `make dev-migration` | Builds and runs DB migration binary (`up` direction) |
| `make test-unit` | Runs `go test` (no tags, root package only) |
| `make test-integration` | Runs `go test -tags=integration ./test/it -v -count=1` |

### Local Dev Setup (non-container)

```bash
# 1. Install Go (match version in go.mod: 1.22.10) + protoc
# 2. Install deps and codegen tools
go mod download && make setup
# 3. Create local config override (empty file is fine, overrides default.toml)
touch config/dev.toml
# 4. Start MySQL 8 instance (or use make dev-docker-up)
# 5. Run migrations
go run ./cmd/migration up
# 6. Run the app (pipe through jq for readable logs -- uses uber/zap JSON logging)
go run ./cmd/gateway | jq
```

### Docker Build (Production)

```bash
docker build -f build/docker/prod/Dockerfile \
  --build-arg GIT_COMMIT_HASH=$(git rev-parse HEAD) \
  -t trino-gateway .
```

The prod container entrypoint (`build/docker/prod/entrypoint.sh`) waits up to 60s for MySQL connectivity, runs migrations automatically, then starts the app.

### CI

GitHub Actions on push to `master` only. Builds and pushes `razorpay/presto_gateway:<sha>` to DockerHub. Razorpay internal image is built separately from datahub repo.

---

## Configuration

### Config Loading (`pkg/config/config.go:Load()`)

1. Loads `config/default.toml` first (base values)
2. Merges environment-specific file on top (e.g., `config/prod.toml`)
3. Environment is set via `app.env` config value

### Config Files

| File | Purpose |
|------|---------|
| `config/default.toml` | Base config with all keys and dev-friendly defaults |
| `config/dev.toml` | Local dev overrides (created empty by `make dev-setup`, gitignored) |
| `config/prod.toml` | Production: `warn` log level, 20s monitor interval, 12 gateway ports (8080-8091) |
| `config/prod-dev.toml` | Prod-dev: same as prod but `info` log level |

### Key Config Sections (`internal/config/config.go:Config`)

| Section | Key Fields | Notes |
|---------|-----------|-------|
| `[app]` | `port` (8000), `metricsPort` (8002), `env`, `gitCommitHash`, `logLevel` | GUI and Twirp must share same port |
| `[db]` | `ConnectionConfig` (dialect, url, port, credentials), `ConnectionPoolConfig` | MySQL only. SSL required by default |
| `[auth]` | `token`, `tokenHeaderKey`, `router.delegatedAuth.*` | Delegated auth has its own validation provider URL + cache TTL |
| `[gateway]` | `ports` (proxy listener ports), `defaultRoutingGroup`, `network` | `network=""` means 0.0.0.0 (for Docker); set to `localhost` outside containers |
| `[monitor]` | `interval`, `healthCheckSql`, `trino.user/password`, `statsValiditySecs` | Prod uses `SHOW SCHEMAS FROM hive` instead of `SELECT 1` |

### Environment Variable Overrides

Viper with prefix `TRINO-GATEWAY` (uppercased). Dots replaced with underscores.

```bash
# Example: override db port
export TRINO-GATEWAY_DB_CONNECTIONCONFIG_PORT=3306
```

Config path can be overridden via `WORKDIR` env var -- loads from `$WORKDIR/config/`.

---

## Database

- **Engine:** MySQL 8+ (full DB access required for app user)
- **Migrations:** Go-based migration files in `internal/gatewayserver/database/migrations/`
- **Run migrations:** `go run ./cmd/migration up` (also: `go run ./cmd/migration` for all options)
- **Auto-migration:** Prod Docker entrypoint runs migrations before starting the app
- **Migration files:** 5 migrations from bootstrap (2021-08) through auth delegation and set-source (2024-05)

---

## Testing

| Type | Command | Notes |
|------|---------|-------|
| Unit | `make test-unit` | Runs `go test` on root package only -- limited scope |
| Integration | `make test-integration` | `go test -tags=integration ./test/it -v -count=1` -- requires running infra |

Test patterns: testify suites, sqlmock for DB mocking.

---

## Debugging & Observability

| Endpoint | Port | Description |
|----------|------|-------------|
| `/admin/swaggerui/` | 8000 (app port) | Swagger UI for all admin/RPC APIs |
| `/commit.txt` | 8000 | Returns git commit hash (set at build time via `GIT_COMMIT_HASH` build arg) |
| HealthCheckAPI (Twirp) | 8000 | `healthApi/server.go:Check()` -- verifies DB connectivity, returns SERVING/NOT_SERVING |
| `/metrics` | 8002 (metrics port) | Prometheus metrics endpoint |

Logging: uber/zap, single-line JSON. Pipe through `jq` for local readability.

---

## Protobuf / Code Generation

- **Proto definition:** `rpc/gateway/service.proto`
- **Generated outputs** (via `make build` / `scripts/compile.sh`):
  - Twirp server stubs (Go)
  - Protobuf Go types
  - OpenAPI v2 spec into `third_party/swaggerui/`
- **Codegen tools installed by `make setup`:** protoc-gen-go, protoc-gen-twirp, protoc-gen-openapiv2, gopherjs
- **Vendor step:** `scripts/compile.sh` runs `go mod vendor` after protoc generation

---

## Gotchas

- **GUI and Twirp share port 8000** -- cannot be separated currently (see comment in `default.toml`)
- **`make test-unit` is minimal** -- only runs root package tests, not `./...`
- **`make dev-migration` has a bug** -- Makefile uses `go build ./cmd/migration/main.go -o migration.go` (flags after source file), but entrypoint and README correctly use `go run ./cmd/migration`
- **Frontend build is slow** (gopherjs) -- skip with `make build` alone if no frontend changes
- **Docker image tag is `presto_gateway`** not `trino-gateway` (historical naming in CI)
- **`network=""` in gateway config** means bind to 0.0.0.0 -- required inside Docker, use `localhost` for local dev
- **Prod monitor healthcheck** uses `SHOW SCHEMAS FROM hive` (validates actual Trino connectivity) vs dev default `SELECT 1`

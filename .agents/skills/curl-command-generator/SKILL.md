---
name: curl-command-generator
description: Generates ready-to-run cURL commands from any codebase (Go, Laravel, Express, Fastify, or other frameworks). Scans route definitions to auto-generate commands, or builds them from user-described endpoints. Includes proper headers, authentication, request bodies with realistic sample data, and environment support (devstack/prod). Use when users ask to "generate curl", "create curl commands", "curl examples for this API", "test this endpoint", or "generate API commands".
disable-model-invocation: true
---

# cURL Command Generator

Generate ready-to-run cURL commands for quick API testing from the command line. Works with any codebase - Go, Laravel/PHP, Express, Fastify, or any HTTP framework.

## Modes of Operation

### Mode 1: Auto-Scan Routes from Codebase

When the user asks to generate cURL commands for a project or service:

1. **Detect the framework** by scanning the project structure. See [framework-patterns.md](references/framework-patterns.md) for detailed detection patterns per framework (Go, Laravel, Express, Fastify).

2. **Extract route metadata** for each endpoint:
   - HTTP method
   - URL path (with path parameters like `:id`, `{id}`)
   - Middleware (auth, validation, rate limiting)
   - Request body structure (from request structs, validation rules, or controller params)
   - Query parameters
   - Response format

3. **Generate cURL commands** grouped by resource/controller

### Mode 2: User-Described Endpoint

When the user describes an endpoint manually (e.g., "generate curl for POST /orders with amount and currency"):

1. Parse the user's description for method, path, and body fields
2. Ask for clarification only if critical info is missing (method or path)
3. Generate the cURL command with realistic sample data
4. Include auth headers based on the project context

## Output Rules

1. **Always use line continuations** (`\`) for readability when command has multiple flags
2. **Use realistic sample data** - not "string" or "test", use domain-appropriate values:
   - Amounts: `50000` (in paise for Razorpay), `100.00` for others
   - Emails: `gaurav@example.com`
   - Names: `Gaurav Kumar`
   - IDs: `acc_GjPLR9Goqd9r2H`, `order_DBJOWzybf0sJbb` (Razorpay-style prefixed IDs when applicable)
   - Currency: `INR`
   - Phone: `9123456789`
   - Timestamps: Unix epoch format where appropriate
3. **Include Content-Type header** for POST, PUT, PATCH requests
4. **Include auth header** based on detected auth mechanism
5. **Replace path params** with placeholder values: `/orders/{order_id}` -> `/orders/order_DBJOWzybf0sJbb`
6. **Add `| jq .`** as a comment suggestion for pretty-printing JSON responses
7. **Show required vs optional fields** in the request body with comments

## Authentication Handling

Razorpay API uses Basic Auth with 9 authentication levels. Detect the correct auth type from route middleware and apply the matching pattern. For non-Razorpay codebases, fall back to generic auth detection (Bearer token, API key header, etc.).

See [razorpay-auth.md](references/razorpay-auth.md) for the full reference with cURL examples for each type.

### Auth Type Quick Reference

| Auth Type | Key | Secret | Use Case |
|-----------|-----|--------|----------|
| **Public** | `rzp_test_KEY` | _(empty)_ | Checkout/SDK routes |
| **Private** | `rzp_test_KEY` | Merchant secret | Direct merchant API calls |
| **Proxy** | `rzp_test_MERCHANTID` | APP_DASHBOARD_SECRET | Dashboard on behalf of merchant |
| **Direct** | _(none)_ | _(none)_ | Gateway callbacks |
| **Internal** | `rzp_test` | From `config/applications.php` | Admin actions, cron, internal apps |
| **Admin** | `rzp_test` | From config + X-Org-Id, X-Admin-Token headers | Admin dashboard operations |
| **Public Callback** | key_id as query param | _(none)_ | Gateway callbacks with key info |
| **Keyless** | _(none)_ | _(none)_ | Payment links, invoices via signed entity IDs |
| **Tally** | Bearer token | _(none)_ | Tally native app integration |

**Partner requests**: Always add `X-Razorpay-Account: SUBMERCHANT_ID` header when a partner makes requests on behalf of a merchant.

### Auth Detection Strategy

1. **Razorpay API monolith (Laravel)**: Check route middleware for `auth:public`, `auth:private`, `auth:proxy`, `auth:direct`, `auth:internal`, `auth:admin`
2. **Go microservices**: Check middleware registrations, interceptor chains, or auth packages. Default to Bearer token or Basic Auth based on code patterns
3. **Other frameworks**: Check for `Authorization` header parsing, passport/JWT middleware, API key validation
4. **If unsure**: Ask the user which auth type applies to their endpoint

## Environment Support

Generate commands for two environments. Default to **devstack** unless the user specifies otherwise.

### Razorpay URL Patterns

**API Monolith:**

| Environment | Base URL |
|-------------|----------|
| Devstack | `https://api-web.dev.razorpay.in` |
| Production | `https://api.razorpay.com` |

**Microservices (with live/test mode):**

| Environment | Base URL Pattern |
|-------------|-----------------|
| Devstack | `https://{service}-{mode}.dev.razorpay.in` |
| Production | `https://{service}-{mode}.razorpay.com` |

Examples:
- `https://partnerships-live.razorpay.com` (prod, live mode)
- `https://partnerships-live.dev.razorpay.in` (devstack, live mode)
- `https://partnerships-test.dev.razorpay.in` (devstack, test mode)

### Devstack Routing Header

For devstack environments, include the `rzpctx-dev-serve-user` header to route requests to a specific pod:

```bash
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X GET "https://api-web.dev.razorpay.in/v1/payments" \
  -H "rzpctx-dev-serve-user: YOUR_DEVSTACK_LABEL"
```

This header is common across all services in devstack. Always include it when generating devstack cURL commands.

### Non-Razorpay Services

For non-Razorpay codebases, default to `http://localhost:8080` and auto-detect base URLs from `.env*` files, `config/` directories, Docker Compose files, Kubernetes manifests, and Makefile targets.

## Advanced cURL Flags

Include these flags only when relevant:

| Flag | When to Include |
|------|-----------------|
| `-i` | When user asks to see response headers |
| `-v` | When user asks for verbose/debug output |
| `-s` | When generating commands for scripts |
| `-o file` | When user wants to save response |
| `-w "\n%{http_code}\n"` | When user wants just the status code |
| `-L` | When the API is known to redirect |
| `-k` | Only for local/stage with self-signed certs (never prod) |
| `--connect-timeout 5` | For APIs known to be slow |
| `-F "file=@path"` | For file upload endpoints |

## Request Body Patterns

### JSON Body (most common)
```bash
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X POST "http://localhost:8080/v1/orders" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000, "currency": "INR", "receipt": "receipt#1", "notes": {"key1": "value1"}}'
```

### Form-Encoded
```bash
curl -X POST "http://localhost:8080/v1/login" \
  -d "email=user@example.com&password=secret123"
```

### Multipart File Upload
```bash
curl -X POST "http://localhost:8080/v1/documents" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@/path/to/document.pdf" \
  -F "type=identification"
```

### Query Parameters
```bash
curl -X GET "http://localhost:8080/v1/payments?from=1609459200&to=1612137600&count=10&skip=0"
```

## Workflow

### Step 1: Determine Mode

- If user says "generate curl for this project/service" -> **Auto-Scan Mode**
- If user says "generate curl for POST /xyz with body {...}" -> **Manual Mode**
- If user says "generate curl" without specifics -> Ask which mode, or default to auto-scan if in a project directory

### Step 2: Gather Context

**Auto-Scan:**
1. Identify the framework and language using [framework-patterns.md](references/framework-patterns.md)
2. Find all route definition files
3. Read route files and extract endpoints
4. Read controllers/handlers to understand request/response shapes
5. Detect auth mechanism from middleware using [razorpay-auth.md](references/razorpay-auth.md)

**Manual:**
1. Parse user's endpoint description
2. Check if we're in a project context for auth defaults
3. Generate with sensible defaults

### Step 3: Generate Commands

For each endpoint, produce a cURL command following the output rules above.

**Grouping order:**
1. Health/Status endpoints (no auth)
2. Authentication endpoints (login, register, token)
3. CRUD operations grouped by resource (list, get, create, update, delete)
4. Action endpoints (submit, approve, reject, etc.)
5. Webhook/callback endpoints

### Step 4: Present Output

1. Show the generated cURL commands in markdown
2. Mention the environment used (default: local)
3. Offer to regenerate for a different environment
4. Offer to add verbose flags or other options

## Example Interactions

### Example 1: Auto-Scan Go Service

**User**: "Generate curl commands for this service"

**Process**: Scan for route files, extract routes (`GET /v1/merchants/:id`, `POST /v1/merchants`), read handlers for request struct fields, detect Private Auth middleware, generate grouped commands.

**Sample output** for a health check endpoint:
```bash
curl -X GET "http://localhost:8080/health"
```

**Sample output** for a CRUD endpoint with Private Auth:
```bash
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X POST "http://localhost:8080/v1/merchants" \
  -H "Content-Type: application/json" \
  -d '{"name": "Acme Corp", "email": "gaurav@example.com", "legal_entity_name": "Acme Corp Pvt Ltd"}'
```

### Example 2: Manual Endpoint (Private Auth)

**User**: "Generate curl for POST /v1/orders with amount, currency, and receipt"

```bash
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X POST "http://localhost:8080/v1/orders" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000, "currency": "INR", "receipt": "receipt#1"}'

# Pretty print response:
# curl ... | jq .
```

### Example 3: Partner Request on Behalf of Merchant

**User**: "Generate curl for a partner fetching payments for a sub-merchant"

```bash
curl -u "rzp_test_partner_CLIENTID:PARTNER_CLIENT_SECRET" \
  -H "X-Razorpay-Account: acc_GjPLR9Goqd9r2H" \
  -X GET "http://localhost:8080/v1/payments?count=10&skip=0"
```

### Example 4: Admin Auth Endpoint

**User**: "Generate curl for admin fetching merchant details"

```bash
curl -u "rzp_test:INTERNAL_APP_SECRET" \
  -X GET "http://localhost:8080/v1/admin/merchants/10000000000000" \
  -H "X-Org-Id: org_100000razorpay" \
  -H "X-Admin-Token: BEARER_TOKEN_CONCAT_PRIMARY_KEY" \
  -H "X-Razorpay-Account: 10000000000000"
```

### Example 5: Devstack Environment

**User**: "Generate curl for the payments list endpoint on devstack"

```bash
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X GET "https://api-web.dev.razorpay.in/v1/payments?from=1609459200&to=1612137600&count=10&skip=0" \
  -H "rzpctx-dev-serve-user: YOUR_DEVSTACK_LABEL"
```

### Example 6: Microservice on Prod

**User**: "Generate curl for partnerships service on prod"

```bash
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X GET "https://partnerships-live.razorpay.com/v1/partners?count=10&skip=0"
```

## Best Practices

1. **Copy-paste ready**: Every command should work as-is after replacing placeholder tokens
2. **Realistic data**: Use domain-appropriate sample values, not generic "test" strings
3. **Grouped logically**: Health -> Auth -> CRUD -> Actions
4. **Auth-aware**: Detect and apply the right auth mechanism
5. **Environment-aware**: Default to devstack, support prod
6. **Minimal flags**: Only include flags that are needed; don't add `-v` or `-i` by default
7. **Comment optional fields**: Mark which body fields are optional vs required
8. **One command per endpoint**: Don't combine multiple requests into shell scripts unless asked

## References

- **[razorpay-auth.md](references/razorpay-auth.md)**: All 9 Razorpay auth types with detailed cURL examples, key/secret formats, and detection strategy
- **[framework-patterns.md](references/framework-patterns.md)**: Framework detection patterns for Go (Echo, Chi, Gin, Gorilla Mux, Twirp, net/http), Laravel, Express, and Fastify

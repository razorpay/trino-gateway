# Razorpay Auth Types Reference

Razorpay API uses Basic Auth with multiple authentication levels. Detect the correct auth type from route middleware and apply the matching pattern.

## 1. Public Auth

Requires only the merchant's `key_id` (no secret). Used for Checkout/Mobile SDK routes that a paying customer will hit.

**Key**: `rzp_test_MERCHANTKEY` (merchant) or `rzp_test_partner_CLIENTID` (partner)
**Secret**: _(empty)_

```bash
# Merchant making the request
curl -u "rzp_test_1DP5mmOlF5G5ag:" \
  -X GET "http://localhost:8080/v1/methods"

# Alternative: key_id as query parameter (for JSONP/browser compatibility)
curl -X GET "http://localhost:8080/v1/methods?key_id=rzp_test_1DP5mmOlF5G5ag"
```

## 2. Private Auth

Requires both `key_id` and `key_secret`. Standard auth for merchants making direct API calls.

**Key**: `rzp_test_MERCHANTKEY` (merchant) or `rzp_test_partner_CLIENTID` (partner)
**Secret**: Merchant's API secret (merchant) or Partner's client secret (partner)

```bash
# Merchant making the request
curl -u "rzp_test_1DP5mmOlF5G5ag:YOUR_KEY_SECRET" \
  -X POST "http://localhost:8080/v1/payments/pay_FHR9UMPCNSqFSz/capture" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000, "currency": "INR"}'

# Partner making request on behalf of a merchant
curl -u "rzp_test_partner_CLIENTID:PARTNER_CLIENT_SECRET" \
  -H "X-Razorpay-Account: acc_GjPLR9Goqd9r2H" \
  -X GET "http://localhost:8080/v1/payments"
```

## 3. Proxy Auth

Used by Dashboard to make requests on behalf of a merchant. The secret is the app-level password (APP_DASHBOARD_SECRET on API side, API_AUTH_PASS on Dashboard side).

**Key**: `rzp_test_MERCHANTID`
**Secret**: `APP_DASHBOARD_SECRET`

```bash
curl -u "rzp_test_10000000000000:APP_DASHBOARD_SECRET" \
  -X GET "http://localhost:8080/v1/webhooks"
```

## 4. Direct Auth

No authentication at all. Used for gateway callbacks where Razorpay has no control over the request format.

```bash
# Gateway callback - no auth headers
curl -X POST "http://localhost:8080/v1/gateway/callback" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "payment_id=pay_FHR9UMPCNSqFSz&status=authorized"
```

## 5. Internal Auth

Application-level auth for internal services. No user or merchant context. Used for admin actions from Dashboard, cron jobs, mock gateways, mailgun, hosted, etc.

**Key**: `rzp_test`
**Secret**: From `config/applications.php` (look up the app-specific secret)

```bash
curl -u "rzp_test:INTERNAL_APP_SECRET" \
  -X POST "http://localhost:8080/v1/settlements/initiate" \
  -H "Content-Type: application/json" \
  -d '{"channel": "icici"}'
```

## 6. Admin Auth

For admin dashboard operations. Requires Internal Auth credentials plus special admin headers.

**Key**: `rzp_test`
**Secret**: From `config/applications.php`
**Required Headers**:
- `X-Org-Id`: `org_100000razorpay`
- `X-Admin-Token`: Concatenation of `bearertoken` + `primary_key` from `admin_tokens` table
- `X-Razorpay-Account`: Merchant ID to validate admin access

```bash
curl -u "rzp_test:INTERNAL_APP_SECRET" \
  -X GET "http://localhost:8080/v1/admin/merchants/10000000000000" \
  -H "X-Org-Id: org_100000razorpay" \
  -H "X-Admin-Token: BEARER_TOKEN_CONCAT_PRIMARY_KEY" \
  -H "X-Razorpay-Account: 10000000000000"
```

## 7. Public Callback Auth

Gateway callbacks where key info comes as a request parameter.

**Key format**: `rzp_test_KEYID` (merchant) or `rzp_test_CLIENTID-ACCOUNTID` (partner)

```bash
# Merchant-initiated callback
curl -X POST "http://localhost:8080/v1/gateway/callback?key_id=rzp_test_1DP5mmOlF5G5ag" \
  -d "status=authorized&payment_id=pay_FHR9UMPCNSqFSz"

# Partner-initiated callback
curl -X POST "http://localhost:8080/v1/gateway/callback?key_id=rzp_test_CLIENTID-acc_GjPLR9Goqd9r2H" \
  -d "status=authorized&payment_id=pay_FHR9UMPCNSqFSz"
```

## 8. Keyless Auth (Public Routes)

No key needed. Uses signed entity IDs (`x_entity_id`) in route params, query params, or headers to identify the merchant. For dashboard use of payment links, invoices, subscriptions without generating API keys.

```bash
# Via route parameter (signed entity ID)
curl -X GET "http://localhost:8080/v1/invoices/inv_FHR9UMPCNSqFSz/status"

# Via query parameter
curl -X GET "http://localhost:8080/v1/invoices/status?x_entity_id=inv_FHR9UMPCNSqFSz"
```

## 9. Tally Auth (Bearer Token)

For Tally native application integration. Token obtained from Razorpay auth-service.

```bash
curl -X GET "http://localhost:8080/v1/tally/invoices" \
  -H "Authorization: Bearer TALLY_AUTH_TOKEN"
```

## Partner Request Pattern

When a partner (aggregator) makes requests on behalf of a merchant, always include the `X-Razorpay-Account` header with the sub-merchant ID:

```bash
curl -u "rzp_test_partner_CLIENTID:PARTNER_CLIENT_SECRET" \
  -H "X-Razorpay-Account: acc_GjPLR9Goqd9r2H" \
  -X GET "http://localhost:8080/v1/payments"
```

## Auth Detection Strategy

1. **Razorpay API monolith (Laravel)**: Check the route middleware in `routes/api.php` or route groups. Look for middleware names like `auth:public`, `auth:private`, `auth:proxy`, `auth:direct`, `auth:internal`, `auth:admin`
2. **Go microservices**: Check middleware registrations, interceptor chains, or auth packages. If no Razorpay-specific auth is found, default to Bearer token or Basic Auth based on code patterns
3. **Other frameworks**: Check for `Authorization` header parsing, passport/JWT middleware, API key validation
4. **If unsure**: Ask the user which auth type applies to their endpoint

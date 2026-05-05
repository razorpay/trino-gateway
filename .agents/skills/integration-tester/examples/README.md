# Payment Server Example

A simple Express-based payment server that demonstrates end-to-end payment workflows including orders, payments, captures, and refunds.

## Quick Start

```bash
# Install dependencies
npm install

# Start the server
npm start
```

The server will run on `http://localhost:3001`

## Authentication

All endpoints require Basic Authentication:
- **Username**: `test`
- **Password**: `test`

## API Endpoints

### Orders

**Create Order**
```bash
POST /api/orders
Content-Type: application/json
Authorization: Basic dGVzdDp0ZXN0

{
  "amount": 50000,
  "currency": "INR",
  "customer_id": "cust_001"
}
```

**Get Order**
```bash
GET /api/orders/:id
Authorization: Basic dGVzdDp0ZXN0
```

### Payments

**Create Payment (Authorize)**
```bash
POST /api/payments
Content-Type: application/json
Authorization: Basic dGVzdDp0ZXN0

{
  "order_id": "ord_xxx",
  "amount": 50000,
  "method": "card"
}
```

**Capture Payment**
```bash
POST /api/payments/:id/capture
Content-Type: application/json
Authorization: Basic dGVzdDp0ZXN0

{}
```

**Get Payment**
```bash
GET /api/payments/:id
Authorization: Basic dGVzdDp0ZXN0
```

### Refunds

**Create Refund**
```bash
POST /api/refunds
Content-Type: application/json
Authorization: Basic dGVzdDp0ZXN0

{
  "payment_id": "pay_xxx",
  "amount": 25000,
  "reason": "Customer request"
}
```

**Get Refund**
```bash
GET /api/refunds/:id
Authorization: Basic dGVzdDp0ZXN0
```

**List Payment Refunds**
```bash
GET /api/payments/:id/refunds
Authorization: Basic dGVzdDp0ZXN0
```

## Testing with HTTP Client

Use the provided HTTP client tool:

```bash
# Create an order
node ../scripts/http-client.js POST "http://localhost:3001/api/orders" \
  '{"amount":50000,"currency":"INR"}' \
  "test:test"

# Create a payment
node ../scripts/http-client.js POST "http://localhost:3001/api/payments" \
  '{"order_id":"ord_xxx","amount":50000,"method":"card"}' \
  "test:test"

# Capture the payment
node ../scripts/http-client.js POST "http://localhost:3001/api/payments/pay_xxx/capture" \
  '{}' \
  "test:test"
```

## Business Rules

The server enforces these business rules:

1. **Amount Validation**: Order amount must be positive
2. **Payment-Order Matching**: Payment amount must match order amount
3. **Double Payment Prevention**: Cannot pay for already-paid orders
4. **Capture Requirement**: Payment must be authorized before capture
5. **Refund Validation**:
   - Can only refund captured payments
   - Total refunds cannot exceed payment amount
   - Partial refunds update status to "partially_refunded"
   - Full refunds update status to "refunded"

## Payment Flow

```
┌─────────┐     ┌──────────┐     ┌──────────┐     ┌─────────┐
│  Order  │────▶│ Payment  │────▶│ Capture  │────▶│ Refund  │
│ created │     │authorized│     │ captured │     │processed│
└─────────┘     └──────────┘     └──────────┘     └─────────┘
                                       │
                                       ▼
                                  ┌─────────┐
                                  │  Order  │
                                  │  paid   │
                                  └─────────┘
```

## Amount Format

All amounts are in **paise** (smallest currency unit):
- ₹1 = 100 paise
- ₹500 = 50000 paise
- ₹1000 = 100000 paise

## Error Handling

The server returns appropriate HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request (validation errors, business rule violations)
- `401` - Unauthorized (missing/invalid auth)
- `404` - Not Found (invalid IDs)

Example error response:
```json
{
  "error": "Order already paid"
}
```

## Testing Scenarios

### Scenario 1: Happy Path
1. Create order → `created`
2. Create payment → `authorized`
3. Capture payment → `captured`, order → `paid`
4. Create partial refund → `partially_refunded`

### Scenario 2: Error Cases
1. Try to pay already-paid order → `400 Order already paid`
2. Try to refund uncaptured payment → `400 Cannot refund uncaptured payment`
3. Try to over-refund → `400 Refund amount exceeds refundable amount`

## Health Check

```bash
GET /health
Authorization: Basic dGVzdDp0ZXN0

Response:
{
  "status": "ok",
  "service": "payment-server"
}
```

## Storage

The server uses in-memory storage (Map objects). All data is lost when the server restarts.

## For Production

This is a demo/testing server. For production use:
- Add persistent database (PostgreSQL, MongoDB)
- Implement proper authentication/authorization
- Add request validation
- Add logging and monitoring
- Add rate limiting
- Use HTTPS
- Add transaction support
- Implement webhooks
- Add idempotency keys

---

**Perfect for demos, integration tests, and understanding payment flows!**

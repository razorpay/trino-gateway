# Payment API Reference

<!--
NOTE: This is a reference document for the payment-api skill.
The payment-api skill will be created as a SEPARATE skill in the future.
This reference is included here for documentation purposes only.
-->

## Overview

This document describes the Payment API endpoints and payloads used by the AI Integration Tester skill.

**IMPORTANT**: A separate `payment-api` skill will be created to provide this information dynamically to Claude Code. For now, this serves as a static reference.

## API Endpoints

### Orders

#### Create Order
```
POST /api/orders
Content-Type: application/json
Authorization: Basic {credentials}

Request Body:
{
  "amount": 50000,           // Required: Amount in paise (₹500)
  "currency": "INR",         // Optional: Default "INR"
  "customer_id": "cust_001"  // Optional: Customer identifier
}

Response (201):
{
  "id": "ord_1234567890_abc123",
  "amount": 50000,
  "currency": "INR",
  "customer_id": "cust_001",
  "status": "created",
  "created_at": "2026-01-09T12:00:00.000Z"
}
```

#### Get Order
```
GET /api/orders/:id
Authorization: Basic {credentials}

Response (200):
{
  "id": "ord_1234567890_abc123",
  "amount": 50000,
  "currency": "INR",
  "customer_id": "cust_001",
  "status": "created|paid",
  "created_at": "2026-01-09T12:00:00.000Z",
  "paid_at": "2026-01-09T12:01:00.000Z"  // Only if status = "paid"
}
```

### Payments

#### Create Payment (Authorize)
```
POST /api/payments
Content-Type: application/json
Authorization: Basic {credentials}

Request Body:
{
  "order_id": "ord_1234567890_abc123",  // Required
  "amount": 50000,                       // Required: Must match order amount
  "method": "card"                       // Optional: Default "card"
}

Response (201):
{
  "id": "pay_1234567890_xyz789",
  "order_id": "ord_1234567890_abc123",
  "amount": 50000,
  "method": "card",
  "status": "authorized",
  "captured": false,
  "refunded_amount": 0,
  "created_at": "2026-01-09T12:00:30.000Z"
}

Error (400 - Order already paid):
{
  "error": "Order already paid"
}

Error (404 - Order not found):
{
  "error": "Order not found"
}
```

#### Capture Payment
```
POST /api/payments/:id/capture
Content-Type: application/json
Authorization: Basic {credentials}

Request Body:
{}  // Empty body

Response (200):
{
  "id": "pay_1234567890_xyz789",
  "order_id": "ord_1234567890_abc123",
  "amount": 50000,
  "method": "card",
  "status": "captured",
  "captured": true,
  "refunded_amount": 0,
  "created_at": "2026-01-09T12:00:30.000Z",
  "captured_at": "2026-01-09T12:01:00.000Z"
}

Error (400 - Already captured):
{
  "error": "Payment already captured"
}

Error (400 - Cannot capture):
{
  "error": "Payment cannot be captured"
}
```

#### Get Payment
```
GET /api/payments/:id
Authorization: Basic {credentials}

Response (200):
{
  "id": "pay_1234567890_xyz789",
  "order_id": "ord_1234567890_abc123",
  "amount": 50000,
  "method": "card",
  "status": "authorized|captured|partially_refunded|refunded",
  "captured": false|true,
  "refunded_amount": 0,
  "created_at": "2026-01-09T12:00:30.000Z",
  "captured_at": "2026-01-09T12:01:00.000Z"  // Only if captured
}
```

### Refunds

#### Create Refund
```
POST /api/refunds
Content-Type: application/json
Authorization: Basic {credentials}

Request Body:
{
  "payment_id": "pay_1234567890_xyz789",  // Required
  "amount": 25000,                         // Required
  "reason": "Customer request"             // Optional
}

Response (201):
{
  "id": "rfnd_1234567890_def456",
  "payment_id": "pay_1234567890_xyz789",
  "amount": 25000,
  "reason": "Customer request",
  "status": "processed",
  "created_at": "2026-01-09T12:02:00.000Z"
}

Error (400 - Cannot refund uncaptured payment):
{
  "error": "Cannot refund uncaptured payment"
}

Error (400 - Over-refund):
{
  "error": "Refund amount exceeds refundable amount",
  "refundable_amount": 20000
}
```

#### Get Refund
```
GET /api/refunds/:id
Authorization: Basic {credentials}

Response (200):
{
  "id": "rfnd_1234567890_def456",
  "payment_id": "pay_1234567890_xyz789",
  "amount": 25000,
  "reason": "Customer request",
  "status": "processed",
  "created_at": "2026-01-09T12:02:00.000Z"
}
```

#### List Payment Refunds
```
GET /api/payments/:id/refunds
Authorization: Basic {credentials}

Response (200):
{
  "count": 2,
  "items": [
    {
      "id": "rfnd_1234567890_def456",
      "payment_id": "pay_1234567890_xyz789",
      "amount": 25000,
      "reason": "Customer request",
      "status": "processed",
      "created_at": "2026-01-09T12:02:00.000Z"
    }
  ]
}
```

## Business Rules

### Orders
- Amount must be positive
- Status transitions: `created` → `paid`

### Payments
- Payment amount must match order amount
- Cannot create payment for already-paid order
- Status transitions: `authorized` → `captured` → `partially_refunded` | `refunded`

### Refunds
- Can only refund captured payments
- Total refunds cannot exceed payment amount
- Partial refund: `refunded_amount < payment.amount` → status = `partially_refunded`
- Full refund: `refunded_amount = payment.amount` → status = `refunded`

## Amount Format

All amounts are in **paise** (smallest currency unit):
- ₹1 = 100 paise
- ₹500 = 50000 paise
- ₹1000 = 100000 paise

## Authentication

All endpoints require Basic Authentication:
- Format: `Authorization: Basic base64(username:password)`
- Default credentials: `test:test`
- Base64 encoded: `dGVzdDp0ZXN0`

## Common HTTP Status Codes

- `200` - Success (GET, POST capture)
- `201` - Created (POST create)
- `400` - Bad Request (validation errors, business rule violations)
- `401` - Unauthorized (missing/invalid auth)
- `404` - Not Found (invalid resource IDs)

## Future: payment-api Skill

**TODO**: Create a separate `payment-api` skill that:
- Provides this API structure dynamically to Claude Code
- Can be queried for endpoint details
- Can be updated independently of the integration tester
- Supports multiple payment gateway formats (Razorpay, Stripe, etc.)

---

**Note**: This is a reference document only. The actual `payment-api` skill will be created separately.

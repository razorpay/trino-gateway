---
name: ai-integration-tester
description: Execute end-to-end business integration tests on payment servers using AI. Orchestrates multi-step payment workflows, validates business logic, and ensures payment systems work correctly before going to production.
---

# AI Integration Tester

Execute end-to-end business integration tests on payment servers using AI. This skill orchestrates multi-step payment workflows, validates business logic, and ensures payment systems work correctly before going to production.

## Overview

This skill provides a complete framework for testing payment integrations through:
- **Automated API Testing**: Execute complex payment workflows with a single command
- **Business Rule Validation**: Verify critical business logic (refund limits, double payment prevention, etc.)
- **Visual Test Reporting**: Get clear, colorful pass/fail results with detailed validation messages
- **Example Payment Server**: Includes a complete payment server implementation for testing

## When to Use

Use this skill when you need to:
- Test payment API integrations end-to-end
- Validate business workflows (order → payment → capture → refund)
- Verify error handling and edge cases
- Demo payment systems with live integration tests
- Record integration testing videos with real-time validation

## Dependencies

<!--
NOTE: This skill references a payment-api skill for API endpoint and payload information.
The payment-api skill will be created as a SEPARATE skill in the future.
For now, API reference documentation is included in references/payment-api-reference.md
-->

**Future Dependency**: `payment-api` skill (to be created separately)
- Will provide Payment API endpoint structures
- Will define request/response payloads
- Will support multiple payment gateways (Razorpay, Stripe, etc.)
- Currently using static reference: `references/payment-api-reference.md`

## Components

### 1. HTTP Client Tool (`scripts/http-client.js`)

A command-line HTTP client for making authenticated API calls:

```bash
node scripts/http-client.js <METHOD> <URL> [BODY] [AUTH]
```

**Parameters:**
- `METHOD`: HTTP method (GET, POST, etc.)
- `URL`: Full URL including protocol, host, port, and path
- `BODY`: JSON request body (use `""` for empty body)
- `AUTH`: Basic auth credentials in format `username:password`

**Output:** Pretty-printed JSON with `{status, headers, body}`

**Example:**
```bash
node scripts/http-client.js POST "http://localhost:3001/api/orders" \
  '{"amount":50000,"currency":"INR"}' \
  "test:test"
```

### 2. Example Payment Server (`examples/payment-server.js`)

A fully functional payment server implementing:
- Order creation and management
- Payment authorization and capture (two-step payment flow)
- Refund processing (partial and full)
- Business rule enforcement

**API Endpoints:**
```
POST   /api/orders              - Create order
GET    /api/orders/:id          - Get order details
POST   /api/payments            - Create payment (authorize)
GET    /api/payments/:id        - Get payment details
POST   /api/payments/:id/capture - Capture authorized payment
POST   /api/refunds             - Create refund
GET    /api/refunds/:id         - Get refund details
GET    /api/payments/:id/refunds - List all refunds for payment
```

**Start the server:**
```bash
npm install express
node examples/payment-server.js
```

Server runs on `http://localhost:3001` with Basic Auth (`test:test`)

## Business Test Flows

The skill executes 6 critical business workflows:

### Flow 1: Happy Path Payment ✅
**Purpose:** Verify complete order-to-payment-to-capture flow

**Steps:**
1. Create order (₹500 / 50000 paise)
2. Create payment for the order
3. Capture the payment
4. Verify order status = "paid"
5. Verify payment status = "captured"

**Validations:**
- ✓ Order.status = "paid"
- ✓ Payment.status = "captured"
- ✓ Payment.captured = true
- ✓ Order.amount = Payment.amount
- ✓ Payment.order_id = Order.id

### Flow 2: Partial Refund 💸
**Purpose:** Verify partial refunds update payment status correctly

**Steps:**
1. Complete Flow 1 to get captured payment
2. Create partial refund (₹250 / 25000 paise)
3. Verify payment status updated

**Validations:**
- ✓ Refund.status = "processed"
- ✓ Refund.amount = 25000
- ✓ Payment.status = "partially_refunded"
- ✓ Payment.refunded_amount = 25000
- ✓ Refund.amount ≤ Payment.amount

### Flow 3: Full Refund 🔄
**Purpose:** Verify full refunds mark payment as fully refunded

**Steps:**
1. Create new order (₹300 / 30000 paise)
2. Create and capture payment
3. Create full refund
4. Verify payment status = "refunded"

**Validations:**
- ✓ Payment.status = "refunded"
- ✓ Payment.refunded_amount = Payment.amount
- ✓ Refund.amount = Payment.amount

### Flow 4: Double Payment Prevention ⚠️
**Purpose:** Verify business rule: cannot pay for already-paid order

**Steps:**
1. Create order (₹100 / 10000 paise)
2. Create and capture first payment
3. Attempt second payment on same order
4. Verify second payment is rejected

**Validations:**
- ✓ Second payment returns HTTP 400
- ✓ Error message: "Order already paid"
- ✓ Order.status remains "paid"
- ✓ Business rule enforced

### Flow 5: Refund Without Capture 🚫
**Purpose:** Verify business rule: cannot refund uncaptured payment

**Steps:**
1. Create order (₹150 / 15000 paise)
2. Create payment (do NOT capture)
3. Attempt to create refund
4. Verify refund is rejected

**Validations:**
- ✓ Refund request returns HTTP 400
- ✓ Error: "Cannot refund uncaptured payment"
- ✓ Payment.status remains "authorized"
- ✓ Payment.captured = false

### Flow 6: Over-refund Prevention 🛡️
**Purpose:** Verify business rule: total refunds cannot exceed payment amount

**Steps:**
1. Create order and capture payment (₹500 / 50000 paise)
2. Create first refund (₹300 / 30000 paise) - succeeds
3. Attempt second refund (₹300 / 30000 paise) - would total ₹600
4. Verify second refund is rejected

**Validations:**
- ✓ First refund succeeds (HTTP 201)
- ✓ Second refund returns HTTP 400
- ✓ Error: "Refund amount exceeds refundable amount"
- ✓ Payment.refunded_amount = 30000 (not 60000)
- ✓ Refundable_amount correctly calculated (20000)

## Usage with Claude Code

### Installation

1. Copy this skill to your Claude Code skills directory:
```bash
cp -r skills/ai-integration-tester ~/.claude/skills/
```

2. The skill will be automatically available in Claude Code

### Running Tests

Simply ask Claude Code:
```
"Run payment integration tests"
"Test the payment server"
"Execute all 6 payment flows"
```

Claude will:
1. Check if payment server is running
2. Execute all 6 test flows in sequence
3. Display color-coded results
4. Provide a final summary with pass/fail counts

### Example Output

```
=== Flow 1: Happy Path Payment ===
✓ Order Created! ord_xxx (Status: created, Amount: ₹500)
✓ Payment Authorized! pay_xxx (Status: authorized)
✓ Payment Captured! (Status: captured, captured: true)
✓ Order Status Updated to 'paid'

Validations:
✓ Order.status = "paid"
✓ Payment.status = "captured"
✓ Payment.captured = true
✓ Amount consistency verified

Flow 1: PASSED (5/5 validations)

───────────────────────────────────────────

FINAL TEST SUMMARY
Total Flows: 6
Passed: 6
Failed: 0
Success Rate: 100%
```

## Recording Demo Videos

This skill is perfect for creating integration testing videos. Here's the setup:

### Terminal Setup (2 terminals side-by-side)

**Terminal 1: Payment Server** (Blue)
```bash
cd examples
node payment-server.js
```
Shows: `✓ Payment Server running on http://localhost:3001`

**Terminal 2: Claude Code** (Green)
```bash
claude
> Run payment integration tests
```
Shows: Test execution with live ✓/✗ results

### Color Legend for Videos

- 🔵 **Blue** = Server running
- 🟢 **Green** = Tests passing (✓)
- 🟠 **Orange** = Order operations
- 🟣 **Purple** = Payment operations
- 🔴 **Red** = Capture operations
- ❌ **Red X** = Expected failures (business rule enforcement)

### What to Highlight

1. **Order Creation** → Status: "created"
2. **Payment Authorization** → Status: "authorized", Money on hold
3. **Payment Capture** → Status: "captured", Money taken, Order becomes "paid"
4. **Refunds** → Partial/Full refunds update payment status
5. **Error Handling** → Double payment blocked, uncaptured refund blocked, over-refund blocked

## Technical Details

### Authentication
All API calls require Basic Authentication:
- Default credentials: `test:test`
- Header format: `Authorization: Basic base64(username:password)`

### Amount Format
- All amounts are in paise (smallest currency unit)
- ₹1 = 100 paise
- ₹500 = 50000 paise

### Payment States
```
Order:  created → paid
Payment: authorized → captured → partially_refunded|refunded
Refund: processed
```

### Error Responses
```json
{
  "error": "Error message",
  "refundable_amount": 20000  // Optional: for over-refund errors
}
```

## Dependencies

- Node.js (v14+)
- Express.js (`npm install express`)

## Architecture

```
┌─────────────────────────────────────────────┐
│           Claude Code (AI Tester)           │
│  - Orchestrates test flows                  │
│  - Validates business rules                 │
│  - Reports results                          │
└────────────────┬────────────────────────────┘
                 │ HTTP calls via http-client.js
                 ▼
┌─────────────────────────────────────────────┐
│         Payment Server (Express)            │
│  - Order management                         │
│  - Payment processing                       │
│  - Refund handling                          │
│  - Business rule enforcement                │
└─────────────────────────────────────────────┘
```

## Benefits

1. **Automated Testing**: Test entire payment workflows with AI assistance
2. **Visual Feedback**: Clear ✓/✗ indicators for each validation
3. **Business-Focused**: Tests actual business rules, not just API contracts
4. **Demo-Ready**: Perfect for presentations and video recordings
5. **Extensible**: Easy to add new flows and validations

## Future Enhancements

### Priority: payment-api Skill (Separate Skill)
**TODO**: Create a separate `payment-api` skill that:
- Provides Payment API structures dynamically to Claude Code
- Supports multiple payment gateways (Razorpay, Stripe, PayPal, etc.)
- Can be queried for endpoint details and request/response formats
- Can be updated independently of the integration tester
- Eliminates need for static reference documentation

### Other Enhancements
- Add webhook testing
- Support for multiple currencies
- Payment failure scenarios
- Settlement simulation
- Load testing capabilities

## Contributing

To add new test flows:
1. Define the business scenario in SKILL.md
2. Add step-by-step execution logic
3. Define validation criteria
4. Update the summary reporting

---

**Made with ❤️ for payment integration testing**

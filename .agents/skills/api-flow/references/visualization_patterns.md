# System Design Visualization Patterns

This reference provides patterns for creating professional flowcharts and system design diagrams for API endpoints using ASCII art.

## Visualization Styles

This skill supports three main visualization styles:

1. **Sequence Diagram** - Shows component interactions over time (recommended for most APIs)
2. **Flowchart** - Shows logic flow with decision points
3. **Layered Architecture** - Shows components across architectural layers with swim lanes

## Style 1: Sequence Diagram (Recommended)

Best for showing API request/response flows with multiple components.

### Example: Simple API Flow

```
POST /api/users/create
════════════════════════════════════════════════════════════════════════════

 Client          API Gateway       Handler           Service         Database
   │                  │                │                │                │
   │  POST /users     │                │                │                │
   ├─────────────────>│                │                │                │
   │                  │                │                │                │
   │                  │ AuthMiddleware │                │                │
   │                  │ (middleware/auth.go:23)         │                │
   │                  ├───────────────>│                │                │
   │                  │    ✓ Valid     │                │                │
   │                  │<───────────────┤                │                │
   │                  │                │                │                │
   │                  │ CreateUserHandler              │                │
   │                  │ (handlers/user.go:45)          │                │
   │                  ├───────────────>│                │                │
   │                  │                │                │                │
   │                  │                │ UserService.Create            │
   │                  │                │ (services/user.go:123)        │
   │                  │                ├───────────────>│                │
   │                  │                │                │                │
   │                  │                │                │ INSERT INTO    │
   │                  │                │                │ users          │
   │                  │                │                │ (repos/user.go:67)
   │                  │                │                ├───────────────>│
   │                  │                │                │                │
   │                  │                │                │   user_id=123  │
   │                  │                │                │<───────────────┤
   │                  │                │                │                │
   │                  │                │  User{id:123}  │                │
   │                  │                │<───────────────┤                │
   │                  │                │                │                │
   │                  │  201 Created   │                │                │
   │                  │<───────────────┤                │                │
   │                  │                │                │                │
   │  201 Created     │                │                │                │
   │  {user object}   │                │                │                │
   │<─────────────────┤                │                │                │
   │                  │                │                │                │

File References:
• middleware/auth.go:23 - JWT token validation
• handlers/user.go:45 - CreateUserHandler entry point
• services/user.go:123 - UserService.Create business logic
• repos/user.go:67 - Database INSERT operation
```

### Example: Complex Flow with Multiple Services

```
POST /api/orders/create
════════════════════════════════════════════════════════════════════════════════════════════

 Client      API       Handler      OrderService   InventoryAPI   PaymentAPI   Database   Queue
   │          │           │               │              │             │           │         │
   │ POST     │           │               │              │             │           │         │
   ├─────────>│           │               │              │             │           │         │
   │          │           │               │              │             │           │         │
   │          │ RateLimit │               │              │             │           │         │
   │          ├──────────>│               │              │             │           │         │
   │          │<──────────┤               │              │             │           │         │
   │          │           │               │              │             │           │         │
   │          │ CreateOrderHandler        │              │             │           │         │
   │          │ (handlers/order.go:234)   │              │             │           │         │
   │          ├──────────>│               │              │             │           │         │
   │          │           │               │              │             │           │         │
   │          │           │ OrderService.Create         │             │           │         │
   │          │           │ (services/order.go:89)      │             │           │         │
   │          │           ├──────────────>│              │             │           │         │
   │          │           │               │              │             │           │         │
   │          │           │               │ GET /check   │             │           │         │
   │          │           │               │ (clients/inventory.go:67)  │           │         │
   │          │           │               ├─────────────>│             │           │         │
   │          │           │               │              │             │           │         │
   │          │           │               │ ✓ In Stock   │             │           │         │
   │          │           │               │<─────────────┤             │           │         │
   │          │           │               │              │             │           │         │
   │          │           │               │ POST /authorize            │           │         │
   │          │           │               │ (clients/payment.go:123)   │           │         │
   │          │           │               ├────────────────────────────>│           │         │
   │          │           │               │              │             │           │         │
   │          │           │               │              │ ✓ Authorized│           │         │
   │          │           │               │<────────────────────────────┤           │         │
   │          │           │               │              │             │           │         │
   │          │           │               │              │             │ BEGIN TXN │         │
   │          │           │               │              │             │ (db/transaction.go:34)
   │          │           │               ├──────────────────────────────────────>│         │
   │          │           │               │              │             │           │         │
   │          │           │               │              │             │ INSERT orders       │
   │          │           │               │              │             │ (repos/order.go:156)│
   │          │           │               ├──────────────────────────────────────>│         │
   │          │           │               │<──────────────────────────────────────┤         │
   │          │           │               │              │             │           │         │
   │          │           │               │              │             │ COMMIT TXN│         │
   │          │           │               ├──────────────────────────────────────>│         │
   │          │           │               │<──────────────────────────────────────┤         │
   │          │           │               │              │             │           │         │
   │          │           │               │              │             │           │ PUBLISH │
   │          │           │               │              │             │           │ order.created
   │          │           │               │              │             │           │ (events/order.go:234)
   │          │           │               ├────────────────────────────────────────────────>│
   │          │           │               │              │             │           │         │
   │          │           │ Order{id:789} │              │             │           │         │
   │          │           │<──────────────┤              │             │           │         │
   │          │           │               │              │             │           │         │
   │          │ 201 Created               │              │             │           │         │
   │          │<──────────┤               │              │             │           │         │
   │          │           │               │              │             │           │         │
   │<─────────┤           │               │              │             │           │         │
   │          │           │               │              │             │           │         │
```

### Example: Flow with Conditional Logic

```
GET /api/account/:id
════════════════════════════════════════════════════════════════════════════

 Client      API      Handler      Cache       Service      Database
   │          │          │            │            │             │
   │  GET     │          │            │            │             │
   ├─────────>│          │            │            │             │
   │          │          │            │            │             │
   │          │ GetAccountHandler     │            │             │
   │          │ (handlers/account.go:78)           │             │
   │          ├─────────>│            │            │             │
   │          │          │            │            │             │
   │          │          │ GET account:123         │             │
   │          │          │ (cache/account.go:23)   │             │
   │          │          ├───────────>│            │             │
   │          │          │            │            │             │
   │          │          │   MISS     │            │             │
   │          │          │<───────────┤            │             │
   │          │          │            │            │             │
   │          │          │ AccountService.GetById  │             │
   │          │          │ (services/account.go:156)             │
   │          │          ├────────────────────────>│             │
   │          │          │            │            │             │
   │          │          │            │            │ SELECT *    │
   │          │          │            │            │ FROM accounts
   │          │          │            │            │ (repos/account.go:45)
   │          │          │            │            ├────────────>│
   │          │          │            │            │             │
   │          │          │            │            │ ✓ Found     │
   │          │          │            │            │<────────────┤
   │          │          │            │            │             │
   │          │          │            │ Account{} │             │
   │          │          │<────────────────────────┤             │
   │          │          │            │            │             │
   │          │          │ SET account:123         │             │
   │          │          ├───────────>│            │             │
   │          │          │<───────────┤            │             │
   │          │          │            │            │             │
   │          │ 200 OK   │            │            │             │
   │          │<─────────┤            │            │             │
   │          │          │            │            │             │
   │<─────────┤          │            │            │             │
   │          │          │            │            │             │

Decision Points:
  ┌─────────────────┐
  │ Cache Hit?      │
  └────┬─────┬──────┘
       │     │
      YES    NO
       │     │
    Return  Query DB
   Cached   & Cache
```

### Example: Parallel/Concurrent Operations

```
POST /api/report/generate
════════════════════════════════════════════════════════════════════════════════════

 Client      API        Handler         Service         Database    Analytics
   │          │            │                │                │            │
   │  POST    │            │                │                │            │
   ├─────────>│            │                │                │            │
   │          │            │                │                │            │
   │          │ GenerateReportHandler       │                │            │
   │          │ (handlers/report.go:123)    │                │            │
   │          ├───────────>│                │                │            │
   │          │            │                │                │            │
   │          │            │ ReportService.Generate          │            │
   │          │            │ (services/report.go:234)        │            │
   │          │            ├───────────────>│                │            │
   │          │            │                │                │            │
   │          │            │                ╔════════════════╗            │
   │          │            │                ║ PARALLEL START ║            │
   │          │            │                ╚════════════════╝            │
   │          │            │                │                │            │
   │          │            │                ├─┐ Goroutine 1: FetchUserData
   │          │            │                │ │ (services/report.go:267)  │
   │          │            │                │ ├──────────────────────────>│
   │          │            │                │ │              │            │
   │          │            │                ├─┤ Goroutine 2: FetchOrderData
   │          │            │                │ │ (services/report.go:289)  │
   │          │            │                │ ├──────────────────────────>│
   │          │            │                │ │              │            │
   │          │            │                ├─┘ Goroutine 3: FetchAnalytics
   │          │            │                │   (services/report.go:312)  │
   │          │            │                ├────────────────────────────────────>│
   │          │            │                │                │            │
   │          │            │                │<──────────────────────────────────────┤
   │          │            │                │<──────────────────────────┤
   │          │            │                │<──────────────────────────┤
   │          │            │                │                │            │
   │          │            │                ╔════════════════╗            │
   │          │            │                ║ WAIT COMPLETE  ║            │
   │          │            │                ╚════════════════╝            │
   │          │            │                │                │            │
   │          │            │                │ AggregateData  │            │
   │          │            │                │ GeneratePDF    │            │
   │          │            │                │                │            │
   │          │            │ Report{url}    │                │            │
   │          │            │<───────────────┤                │            │
   │          │            │                │                │            │
   │          │ 200 OK     │                │                │            │
   │          │<───────────┤                │                │            │
   │          │            │                │                │            │
   │<─────────┤            │                │                │            │
   │          │            │                │                │            │
```

### Example: Background Jobs & Async Processing

```
POST /api/video/upload
════════════════════════════════════════════════════════════════════════════════════

 Client      API       Handler      Service      Database    Queue    [Background Worker]
   │          │           │             │             │         │              │
   │  POST    │           │             │             │         │              │
   ├─────────>│           │             │             │         │              │
   │          │           │             │             │         │              │
   │          │ UploadVideoHandler      │             │         │              │
   │          │ (handlers/video.go:456) │             │         │              │
   │          ├──────────>│             │             │         │              │
   │          │           │             │             │         │              │
   │          │           │ VideoService.Upload       │         │              │
   │          │           │ (services/video.go:234)   │         │              │
   │          │           ├────────────>│             │         │              │
   │          │           │             │             │         │              │
   │          │           │             │ INSERT videos         │              │
   │          │           │             │ status='pending'      │              │
   │          │           │             │ (repos/video.go:123)  │              │
   │          │           │             ├────────────>│         │              │
   │          │           │             │<────────────┤         │              │
   │          │           │             │             │         │              │
   │          │           │             │             │ PUBLISH │              │
   │          │           │             │             │ video.process          │
   │          │           │             │             │ (events/video.go:167)  │
   │          │           │             ├─────────────────────>│              │
   │          │           │             │             │         │              │
   │          │           │ 202 Accepted│             │         │              │
   │          │           │<────────────┤             │         │              │
   │          │           │             │             │         │              │
   │          │ 202 Accepted            │             │         │              │
   │          │ {video_id, status:'processing'}       │         │              │
   │          │<──────────┤             │             │         │              │
   │          │           │             │             │         │              │
   │<─────────┤           │             │             │         │              │
   │          │           │             │             │         │              │
   │          │           │             │             │         │   SUBSCRIBE  │
   │          │           │             │             │         │   video.process
   │          │           │             │             │         ├─────────────>│
   │          │           │             │             │         │              │
   │          │           │             │             │         │ VideoProcessingWorker
   │          │           │             │             │         │ (workers/video_processor.go:234)
   │          │           │             │             │         │              │
   │          │           │             │             │         │ Process video│
   │          │           │             │             │         │ Transcode    │
   │          │           │             │             │         │              │
   │          │           │             │             │<────────────────────────┤
   │          │           │             │             │ UPDATE videos           │
   │          │           │             │             │ status='ready'          │
   │          │           │             │             │ (repos/video.go:156)    │
   │          │           │             │             ├─────────────────────────┤
   │          │           │             │             │         │              │

Timeline:
  ┌────────────────────┐                    ┌──────────────────────────────┐
  │ Sync Request/      │                    │ Async Background Processing   │
  │ Response (Fast)    │                    │ (May take minutes)            │
  └────────────────────┘                    └──────────────────────────────┘
```

## Style 2: Flowchart with Decision Points

Best for showing complex conditional logic and decision trees.

```
POST /api/payment/process
════════════════════════════════════════════════════

                     ┌─────────────────────────┐
                     │ ProcessPaymentHandler   │
                     │ (handlers/payment.go:345)│
                     └────────────┬────────────┘
                                  │
                                  ▼
                     ┌─────────────────────────┐
                     │ PaymentService.Process  │
                     │ (services/payment.go:567)│
                     └────────────┬────────────┘
                                  │
                                  ▼
                     ┌─────────────────────────┐
                     │ SELECT account balance  │
                     │ (repos/account.go:89)   │
                     └────────────┬────────────┘
                                  │
                                  ▼
                          ╱───────────────╲
                         ╱  Sufficient    ╲
                        ╱     Funds?       ╲
                        ╲                  ╱
                         ╲                ╱
                          ╲──────┬───────╱
                                 │
                  ┌──────────────┴──────────────┐
                  │ NO                          │ YES
                  ▼                             ▼
      ┌──────────────────────┐    ┌──────────────────────────┐
      │ PUBLISH payment.failed│   │ POST /gateway/charge     │
      │ (events/payment.go:123)│   │ (clients/payment_gw.go:234)│
      └──────────┬────────────┘    └──────────┬───────────────┘
                 │                            │
                 ▼                            ▼
      ┌──────────────────────┐      ╱────────────────╲
      │ 400 Bad Request      │     ╱  Gateway         ╲
      │ "insufficient_funds" │    ╱   Success?         ╲
      └──────────────────────┘    ╲                   ╱
                                   ╲                  ╱
                                    ╲────────┬───────╱
                                             │
                              ┌──────────────┴────────────┐
                              │ NO                        │ YES
                              ▼                           ▼
                  ┌────────────────────────┐  ┌──────────────────────┐
                  │ INSERT failed_payments │  │ INSERT transaction   │
                  │ (repos/payment.go:234) │  │ (repos/txn.go:145)   │
                  └───────────┬────────────┘  └──────────┬───────────┘
                              │                          │
                              ▼                          ▼
                  ┌────────────────────────┐  ┌──────────────────────┐
                  │ PUBLISH payment.retry  │  │ PUBLISH payment.success│
                  │ (events/payment.go:178)│  │ (events/payment.go:156)│
                  └───────────┬────────────┘  └──────────┬───────────┘
                              │                          │
                              ▼                          ▼
                  ┌────────────────────────┐  ┌──────────────────────┐
                  │ 502 Bad Gateway        │  │ 200 OK               │
                  │ "gateway_error"        │  │ {transaction}        │
                  └────────────────────────┘  └──────────────────────┘

Legend:
  ┌────────┐  = Process/Action
  │        │
  └────────┘

  ╱────────╲  = Decision Point
  ╲        ╱
   ╲──────╱
```

## Style 3: Layered Architecture Diagram

Best for showing how components interact across architectural layers.

```
POST /api/orders/create
════════════════════════════════════════════════════════════════════════════════

┌─ API LAYER ──────────────────────────────────────────────────────────────────┐
│                                                                               │
│  ┌─────────────────┐      ┌─────────────────┐      ┌────────────────────┐   │
│  │ RateLimit MW    │─────>│ Auth MW         │─────>│ CreateOrderHandler │   │
│  │ (middleware/    │      │ (middleware/    │      │ (handlers/         │   │
│  │  ratelimit.go:12)│      │  auth.go:23)    │      │  order.go:234)     │   │
│  └─────────────────┘      └─────────────────┘      └──────────┬─────────┘   │
│                                                                │             │
└────────────────────────────────────────────────────────────────┼─────────────┘
                                                                 │
┌─ SERVICE LAYER ──────────────────────────────────────────────┼─────────────┐
│                                                                │             │
│                              ┌─────────────────────────────────▼──────────┐ │
│                              │ OrderService.Create                        │ │
│                              │ (services/order.go:89)                     │ │
│                              └────┬──────────────┬──────────────┬─────────┘ │
│                                   │              │              │           │
│                    ┌──────────────▼──┐   ┌───────▼────────┐   ┌▼──────────┐│
│                    │ ValidateOrder   │   │ CalculateTotal │   │ ApplyPromo││
│                    │ (services/      │   │ (services/     │   │ (services/││
│                    │  order.go:112)  │   │  order.go:145) │   │  order.go:││
│                    └─────────────────┘   └────────────────┘   └───────────┘│
│                                                                             │
└────────────────────────────────────────────────────────┬────────────────────┘
                                                         │
┌─ INTEGRATION LAYER ─────────────────────────────────┼────────────────────────┐
│                                                        │                      │
│  ┌──────────────────┐        ┌──────────────────┐    │   ┌───────────────┐  │
│  │ InventoryClient  │<───┐   │ PaymentClient    │<───┤   │ NotifClient   │  │
│  │ GET /check       │    │   │ POST /authorize  │    │   │ POST /send    │  │
│  │ (clients/        │    └───┤ (clients/        │    └──>│ (clients/     │  │
│  │  inventory.go:67)│        │  payment.go:123) │        │  notif.go:56) │  │
│  └────────┬─────────┘        └────────┬─────────┘        └───────────────┘  │
│           │                           │                                      │
│           ▼                           ▼                                      │
│  ┌──────────────────┐        ┌──────────────────┐                           │
│  │ Inventory API    │        │ Payment Gateway  │                           │
│  │ (External)       │        │ (Stripe)         │                           │
│  └──────────────────┘        └──────────────────┘                           │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
                                                         │
┌─ DATA LAYER ────────────────────────────────────────┼────────────────────────┐
│                                                        │                      │
│                              ┌─────────────────────────▼──────────┐          │
│                              │ OrderRepository                    │          │
│                              │ (repos/order.go:156)               │          │
│                              └────┬──────────────┬────────────────┘          │
│                                   │              │                           │
│                    ┌──────────────▼──┐   ┌───────▼────────┐                 │
│                    │ BEGIN TRANSACTION│  │ INSERT orders   │                 │
│                    │ (db/txn.go:34)  │   │ INSERT items    │                 │
│                    └──────────────────┘   │ UPDATE inventory│                 │
│                                           │ COMMIT          │                 │
│                                           │ (repos/order.go)│                 │
│                                           └────────┬────────┘                 │
│                                                    │                          │
│                              ┌─────────────────────▼──────────┐              │
│                              │ PostgreSQL Database            │              │
│                              │ • orders table                 │              │
│                              │ • order_items table            │              │
│                              │ • inventory table              │              │
│                              └────────────────────────────────┘              │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
                                                         │
┌─ MESSAGING LAYER ───────────────────────────────────┼────────────────────────┐
│                                                        │                      │
│                              ┌─────────────────────────▼──────────┐          │
│                              │ EventPublisher                     │          │
│                              │ (events/order.go:234)              │          │
│                              └────────────────┬───────────────────┘          │
│                                               │                              │
│                                               ▼                              │
│                              ┌──────────────────────────────────┐            │
│                              │ Kafka Topic: "orders"            │            │
│                              │ Message: order.created           │            │
│                              └──────────────────────────────────┘            │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

## Component Symbols

```
┌────────────────┐
│ Component Box  │  = Function, Service, Handler
└────────────────┘

╱────────────────╲
╱   Decision?    ╲ = Decision Point (Yes/No, True/False)
╲                ╱
 ╲──────────────╱

╔════════════════╗
║ Critical Box   ║ = Critical Operation, Transaction Boundary
╚════════════════╝

│ = Vertical Flow
─ = Horizontal Flow
├ = Branch Point
└ = End Branch
┌ ┐ = Top Corners
└ ┘ = Bottom Corners
─> = Arrow/Direction
├──> = Branch with arrow
└──> = End branch with arrow

[External Service] = External API/Service
{Database Operation} = Database query/command
((Cache Operation)) = Cache interaction
<<Queue Message>> = Message queue operation
```

## File Reference Format

Always include file references in the format:
```
ComponentName (file/path.go:line_number)
```

Example:
```
CreateUserHandler (handlers/user.go:45)
UserService.Create (services/user.go:123)
INSERT INTO users (repos/user.go:67)
```

## Best Practices

1. **Use Sequence Diagrams** for most API flows - they show component interactions clearly
2. **Use Flowcharts** when there's complex branching logic to visualize
3. **Use Layered Architecture** for complex systems with many components across layers
4. **Always include file references** with line numbers for easy code navigation
5. **Show both success and error paths** when relevant
6. **Indicate async operations** clearly (background jobs, goroutines)
7. **Mark external dependencies** distinctly (APIs, databases, queues)
8. **Include actual data** where helpful (query snippets, status codes, response structure)
9. **Keep it readable** - balance detail with clarity
10. **Use consistent spacing** - align components in sequence diagrams for readability

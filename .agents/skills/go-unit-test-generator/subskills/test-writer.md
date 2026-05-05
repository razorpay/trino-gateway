# Phase 3: New Test Writer

Generate production-grade `*_test.go` files for new and uncovered functions identified in Phase 1.

## Pre-Generation Checklist

Before writing any test, confirm:
1. You have read the function's source code completely
2. You have identified ALL interfaces the function depends on
3. You have read the project's existing test conventions (from Phase 0)
4. You know the mock strategy: GoMock, testify/mock, or manual mocks

## Test File Placement

- Test file goes in the **same package** as the source: `order.go` → `order_test.go`
- If the file already exists, **append** new test functions — do not overwrite
- Package declaration matches source: `package core` (not `package core_test` unless the project uses that convention)

## File Existence Check & Append Logic

**CRITICAL: Phase 1 (change-analysis) outputs `test_files` mapping with `exists: true/false` for each source file.**

For each changed source file (e.g., `core.go`):

1. **Read the test file status from Phase 1 output:**
   ```json
   "test_files": {
     "internal/core/order.go": {"test_file": "internal/core/order_test.go", "exists": true},
     "internal/core/payment.go": {"test_file": "internal/core/payment_test.go", "exists": false}
   }
   ```

2. **If `test_files["core.go"].exists == true` (Append mode):**
   - Read the entire existing test file
   - **IDEMPOTENCY CHECK (MANDATORY):** Before appending any new test function, check if it already exists:
     ```bash
     grep -q "func TestCore_CreateOrder(" internal/core/order_test.go
     ```
     - **If found → SKIP** this function. Log: `"Skipping TestCore_CreateOrder — already exists in order_test.go"`
     - **If NOT found → proceed** with append
     - Perform this check for EVERY test function to be written, not just one
   - Find the last test function in the file (search for `func Test`)
   - Identify the insertion point (after the last test function or at EOF)
   - Append only NEW test functions (those that passed the idempotency check) AFTER the last existing test
   - **Preserve all existing test code, imports, and helper functions — do not modify them**
   - Only ADD new `func Test*` functions for newly added/modified source functions
   - Keep the same package declaration and imports

3. **If `test_files["core.go"].exists == false` (Create mode):**
   - Create new `core_test.go` file with:
     - Package declaration: `package core` (must match source file's package)
     - Import block with all required dependencies:
       - Standard: `context`, `testing`, `strings` (if needed)
       - Testing libs: `github.com/golang/mock/gomock`, `github.com/stretchr/testify/{assert,require}`
       - Project imports: any mocks, interfaces, or types from the source package
     - New test functions for all new/modified functions
   - Add to git tracking: `git add core_test.go`

## Append Operation Example

```go
// EXISTING core_test.go
package core

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestCore_GetOrder(t *testing.T) {
    // existing test
}

// NEW TEST APPENDED BELOW
func TestCore_CreateOrder(t *testing.T) {
    tests := []struct{
        // new test structure
    }{}
    // ... test impl
}
```

**Key Rules for Append:**
- Read entire existing file first
- **Never append a test function that already exists** — `grep -q "func TestFunctionName(" <test_file>` before every append
- Find insertion point by searching for last `func Test` function
- Append AFTER it (before EOF)
- Never modify existing test functions
- Add new imports only if needed (check what's already imported)

## Required Test Categories

For EVERY function, generate tests covering:

| Category | Description | Min Cases |
|----------|-------------|-----------|
| Happy path | Normal successful execution | 1-2 |
| Error paths | Every distinct error return | 1 per error |
| Nil/empty input | Nil pointers, empty strings, zero values | 1-2 |
| Boundary values | Max int, empty slice, single element | 1-2 |
| Concurrent access | If function uses shared state, goroutines, channels | 1 (if applicable) |

## Test Structure Template

Use table-driven tests. See [references/production-go-patterns.md](../references/production-go-patterns.md) for complete examples.

```go
func TestCore_CreateOrder(t *testing.T) {
    tests := []struct {
        name      string
        input     CreateOrderRequest
        setupMock func(*mocks.MockOrderRepo, *mocks.MockPaymentSvc)
        want      *CreateOrderResponse
        wantErr   bool
        errClass  errors.Class // Razorpay error class, if applicable
    }{
        {
            name:  "success - valid order",
            input: CreateOrderRequest{Amount: 1000, Currency: "INR", MerchantID: "mid_123"},
            setupMock: func(repo *mocks.MockOrderRepo, pay *mocks.MockPaymentSvc) {
                repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&Order{ID: "order_1"}, nil)
                pay.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
            },
            want:    &CreateOrderResponse{OrderID: "order_1"},
            wantErr: false,
        },
        {
            name:  "error - repository failure",
            input: CreateOrderRequest{Amount: 1000, Currency: "INR", MerchantID: "mid_123"},
            setupMock: func(repo *mocks.MockOrderRepo, pay *mocks.MockPaymentSvc) {
                pay.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
                repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, errorclass.ErrDBError.New("db down"))
            },
            want:     nil,
            wantErr:  true,
            errClass: errorclass.ErrDBError,
        },
        {
            name:    "error - nil input fields",
            input:   CreateOrderRequest{},
            setupMock: func(repo *mocks.MockOrderRepo, pay *mocks.MockPaymentSvc) {
                // No calls expected — validation should fail before reaching repo
            },
            want:     nil,
            wantErr:  true,
            errClass: errorclass.ErrBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockRepo := mocks.NewMockOrderRepo(ctrl)
            mockPay := mocks.NewMockPaymentSvc(ctrl)
            tt.setupMock(mockRepo, mockPay)

            core := &Core{
                orderRepo:  mockRepo,
                paymentSvc: mockPay,
            }

            got, err := core.CreateOrder(context.Background(), tt.input)

            if tt.wantErr {
                require.Error(t, err)
                if tt.errClass != nil {
                    assert.Equal(t, tt.errClass, err.Class())
                }
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Assertion Quality Requirements

Every test case MUST include at least 2 meaningful assertions. A "meaningful" assertion verifies a specific value, not just existence.

**Minimum assertion pattern for success cases:**
```go
require.NoError(t, err)                          // 1. Error check (required)
require.NotNil(t, result)                        // 2. Nil check (required)
assert.Equal(t, expected.ID, result.ID)          // 3. Value check (REQUIRED — this makes it meaningful)
```

**Minimum assertion pattern for error cases:**
```go
require.Error(t, err)                            // 1. Error occurred
assert.Equal(t, expectedErrClass, err.Class())   // 2. Specific error type (not just "any error")
```

**Anti-pattern — do NOT write tests like this:**
```go
// BAD: passes coverage but catches nothing
result, err := core.CreateOrder(ctx, input)
assert.NoError(t, err)
assert.NotNil(t, result)
// Missing: what IS result? what are its fields?
// Would this pass if CreateOrder returned a completely wrong object? YES — that's the problem.
```

---

## Mock Generation

### If GoMock is used (preferred)

Check if mock files exist:
```bash
ls mocks/ 2>/dev/null || ls internal/mocks/ 2>/dev/null
```

If mocks are missing for an interface, generate:
```bash
mockgen -source=internal/core/interfaces.go -destination=internal/mocks/mock_core.go -package=mocks
```

### If testify/mock is used

```go
type MockOrderRepo struct {
    mock.Mock
}

func (m *MockOrderRepo) Create(ctx context.Context, order *Order) (*Order, error) {
    args := m.Called(ctx, order)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Order), args.Error(1)
}
```

### If manual mocks are used

```go
type stubOrderRepo struct {
    createFunc func(ctx context.Context, order *Order) (*Order, error)
}

func (s *stubOrderRepo) Create(ctx context.Context, order *Order) (*Order, error) {
    return s.createFunc(ctx, order)
}
```

**Always match the project's existing mock strategy.** Do not introduce a new mock library.

## Special Patterns

### HTTP Handler Tests

```go
func TestHandleCreateOrder(t *testing.T) {
    tests := []struct {
        name       string
        method     string
        body       string
        setupMock  func(*mocks.MockCore)
        wantCode   int
        wantBody   string
    }{
        {
            name:     "success",
            method:   http.MethodPost,
            body:     `{"amount":1000,"currency":"INR"}`,
            setupMock: func(m *mocks.MockCore) {
                m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(&Response{ID: "ord_1"}, nil)
            },
            wantCode: http.StatusOK,
            wantBody: `"id":"ord_1"`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockCore := mocks.NewMockCore(ctrl)
            tt.setupMock(mockCore)

            handler := NewHandler(mockCore)

            req := httptest.NewRequest(tt.method, "/v1/orders", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            rec := httptest.NewRecorder()

            handler.ServeHTTP(rec, req)

            assert.Equal(t, tt.wantCode, rec.Code)
            if tt.wantBody != "" {
                assert.Contains(t, rec.Body.String(), tt.wantBody)
            }
        })
    }
}
```

### Transaction Context Tests (CRITICAL)

When testing code that uses `db.Transaction`:

```go
func TestCore_TransferFunds(t *testing.T) {
    // CRITICAL: Inside db.Transaction callback, all DB calls MUST use
    // the transaction context (tctx), NOT the parent context (ctx).
    // Verify the mock expectations use gomock.Any() for the context param
    // to avoid false positives — then add a dedicated test that verifies
    // the function passes the transaction context correctly.

    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockDB := mocks.NewMockDB(ctrl)
    mockRepo := mocks.NewMockRepo(ctrl)

    // The transaction callback receives tctx
    mockDB.EXPECT().Transaction(gomock.Any(), gomock.Any()).
        DoAndReturn(func(ctx context.Context, fn func(tctx context.Context) error) error {
            return fn(ctx) // simulate transaction by passing same ctx
        })

    // Repo call inside transaction should receive the transaction context
    mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

    core := &Core{db: mockDB, repo: mockRepo}
    err := core.TransferFunds(context.Background(), transferReq)
    require.NoError(t, err)
}
```

### Async/Channel Tests

```go
func TestWorker_Process(t *testing.T) {
    done := make(chan struct{})
    go func() {
        defer close(done)
        worker.Process(ctx, input)
    }()

    select {
    case <-done:
        // success
    case <-time.After(5 * time.Second):
        t.Fatal("worker.Process timed out after 5s")
    }
}
```

**Never use `time.Sleep` in tests.** Use channels + `select` + `time.After`.

## Test Helper Functions

All test helpers MUST call `t.Helper()`:

```go
func createTestOrder(t *testing.T, amount int64) *Order {
    t.Helper()
    return &Order{
        ID:       "order_test_" + strconv.FormatInt(amount, 10),
        Amount:   amount,
        Currency: "INR",
        Status:   "created",
    }
}
```

## Imports

Standard test import block:

```go
import (
    "context"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/razorpay/<service>/internal/core"
    "github.com/razorpay/<service>/internal/mocks"
)
```

Adjust the import paths to match the project's module path from `go.mod`.

# Production Go Test Patterns

Consolidated reference of production-grade Go unit test patterns used across Razorpay services. Source: Google, Uber, and Razorpay best practices.

## Table of Contents

1. [Table-Driven Tests](#table-driven-tests)
2. [GoMock Patterns](#gomock-patterns)
3. [Testify Patterns](#testify-patterns)
4. [Manual Mock Patterns](#manual-mock-patterns)
5. [HTTP Handler Tests](#http-handler-tests)
6. [Transaction Context Tests](#transaction-context-tests)
7. [Error Class Assertions](#error-class-assertions)
8. [Setup and Teardown](#setup-and-teardown)
9. [Async and Concurrent Tests](#async-and-concurrent-tests)
10. [Test Helper Functions](#test-helper-functions)
11. [Anti-Patterns](#anti-patterns)

---

## Table-Driven Tests

The standard pattern for all Go tests at Razorpay.

### Basic Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "descriptive case name",
            input:   InputType{Field: "value"},
            want:    OutputType{Result: "expected"},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Advanced Structure (with mock setup)

```go
func TestCore_Method(t *testing.T) {
    tests := []struct {
        name      string
        input     RequestType
        setupMock func(*mocks.MockRepo, *mocks.MockService)
        want      *ResponseType
        wantErr   bool
        errClass  errors.Class
    }{
        {
            name:  "success",
            input: RequestType{ID: "123"},
            setupMock: func(repo *mocks.MockRepo, svc *mocks.MockService) {
                repo.EXPECT().FindByID(gomock.Any(), "123").Return(&Entity{ID: "123"}, nil)
                svc.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
            },
            want:    &ResponseType{ID: "123", Status: "active"},
            wantErr: false,
        },
        {
            name:  "error - not found",
            input: RequestType{ID: "999"},
            setupMock: func(repo *mocks.MockRepo, svc *mocks.MockService) {
                repo.EXPECT().FindByID(gomock.Any(), "999").Return(nil, errorclass.ErrNotFound.New("not found"))
            },
            want:     nil,
            wantErr:  true,
            errClass: errorclass.ErrNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockRepo := mocks.NewMockRepo(ctrl)
            mockSvc := mocks.NewMockService(ctrl)
            tt.setupMock(mockRepo, mockSvc)

            core := &Core{repo: mockRepo, svc: mockSvc}
            got, err := core.Method(context.Background(), tt.input)

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

### Naming Conventions

```go
// Test function name: Test<Receiver>_<Method>
func TestCore_CreateOrder(t *testing.T) {}
func TestHandler_ServeHTTP(t *testing.T) {}
func TestValidateInput(t *testing.T) {} // no receiver

// Test case names: <outcome> - <condition>
"success - valid input"
"error - empty merchant ID"
"error - repository failure"
"edge case - zero amount"
"edge case - nil context"
```

---

## GoMock Patterns

### Controller Setup

```go
ctrl := gomock.NewController(t)
defer ctrl.Finish()

mockRepo := mocks.NewMockOrderRepo(ctrl)
```

### Common Matchers

```go
gomock.Any()                          // matches any value
gomock.Eq("expected")                 // exact match
gomock.Nil()                          // matches nil
gomock.Not(gomock.Nil())              // matches non-nil
gomock.InOrder(call1, call2)          // enforce call order
```

### Return Values

```go
mockRepo.EXPECT().Find(gomock.Any(), "id").Return(&Entity{}, nil)
mockRepo.EXPECT().Find(gomock.Any(), "bad").Return(nil, errors.New("not found"))
```

### Call Count

```go
mockRepo.EXPECT().Save(gomock.Any()).Times(1)
mockRepo.EXPECT().Save(gomock.Any()).AnyTimes()
mockRepo.EXPECT().Save(gomock.Any()).MinTimes(1).MaxTimes(3)
```

### DoAndReturn (for complex logic)

```go
mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).
    DoAndReturn(func(ctx context.Context, entity *Entity) error {
        entity.ID = "generated_id"
        return nil
    })
```

### Generate Mocks

```bash
# From interface file
mockgen -source=internal/core/interfaces.go -destination=internal/mocks/mock_core.go -package=mocks

# From specific interface
mockgen -destination=internal/mocks/mock_repo.go -package=mocks github.com/razorpay/svc/internal/core OrderRepository
```

---

## Testify Patterns

### assert vs require

```go
// assert: test continues on failure (soft assertion)
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.True(t, condition)
assert.Contains(t, str, substr)
assert.Nil(t, ptr)
assert.Len(t, slice, 3)

// require: test stops immediately on failure (hard assertion)
require.NoError(t, err)       // Use for setup steps
require.NotNil(t, result)     // Use when subsequent assertions depend on this
```

**Rule**: Use `require` for preconditions and `assert` for the actual test assertions.

### testify/mock

```go
type MockRepo struct {
    mock.Mock
}

func (m *MockRepo) Create(ctx context.Context, entity *Entity) (*Entity, error) {
    args := m.Called(ctx, entity)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Entity), args.Error(1)
}

// In test:
repo := new(MockRepo)
repo.On("Create", mock.Anything, mock.MatchedBy(func(e *Entity) bool {
    return e.Amount > 0
})).Return(&Entity{ID: "1"}, nil)

// Verify all expectations were met
repo.AssertExpectations(t)
```

---

## Manual Mock Patterns

When neither GoMock nor testify is available:

```go
type stubRepo struct {
    findFunc   func(ctx context.Context, id string) (*Entity, error)
    createFunc func(ctx context.Context, entity *Entity) error
}

func (s *stubRepo) Find(ctx context.Context, id string) (*Entity, error) {
    if s.findFunc != nil {
        return s.findFunc(ctx, id)
    }
    return nil, errors.New("findFunc not set")
}

func (s *stubRepo) Create(ctx context.Context, entity *Entity) error {
    if s.createFunc != nil {
        return s.createFunc(ctx, entity)
    }
    return errors.New("createFunc not set")
}

// In test:
repo := &stubRepo{
    findFunc: func(ctx context.Context, id string) (*Entity, error) {
        return &Entity{ID: id, Status: "active"}, nil
    },
}
```

---

## HTTP Handler Tests

```go
func TestHandler(t *testing.T) {
    tests := []struct {
        name       string
        method     string
        path       string
        body       string
        headers    map[string]string
        setupMock  func(*mocks.MockCore)
        wantCode   int
        wantBody   string
    }{
        {
            name:   "POST success",
            method: http.MethodPost,
            path:   "/v1/orders",
            body:   `{"amount":1000}`,
            headers: map[string]string{"Content-Type": "application/json"},
            setupMock: func(m *mocks.MockCore) {
                m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&Order{ID: "ord_1"}, nil)
            },
            wantCode: http.StatusOK,
            wantBody: `"id":"ord_1"`,
        },
        {
            name:     "POST bad request",
            method:   http.MethodPost,
            path:     "/v1/orders",
            body:     `{}`,
            wantCode: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockCore := mocks.NewMockCore(ctrl)
            if tt.setupMock != nil {
                tt.setupMock(mockCore)
            }

            handler := NewHandler(mockCore)
            req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
            for k, v := range tt.headers {
                req.Header.Set(k, v)
            }
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

---

## Transaction Context Tests

**CRITICAL**: Inside `db.Transaction(ctx, fn)`, the callback receives a transaction context (`tctx`). ALL database calls inside the callback MUST use `tctx`, NOT the parent `ctx`.

```go
func TestCore_WithTransaction(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockDB := mocks.NewMockDB(ctrl)
    mockRepo := mocks.NewMockRepo(ctrl)

    mockDB.EXPECT().Transaction(gomock.Any(), gomock.Any()).
        DoAndReturn(func(ctx context.Context, fn func(tctx context.Context) error) error {
            return fn(ctx)
        })

    mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

    core := &Core{db: mockDB, repo: mockRepo}
    err := core.ProcessWithTransaction(context.Background(), input)
    require.NoError(t, err)
}
```

---

## Error Class Assertions

Razorpay services use `errorclass` for typed errors:

```go
// Assert error class
if tt.wantErr {
    require.Error(t, err)
    if tt.errClass != nil {
        assert.Equal(t, tt.errClass, err.Class())
    }
    return
}

// Common error classes
errorclass.ErrNotFound
errorclass.ErrBadRequest
errorclass.ErrDBError
errorclass.ErrUnauthorized
errorclass.ErrConflict
errorclass.ErrInternal
```

---

## Setup and Teardown

### TestMain (global)

```go
func TestMain(m *testing.M) {
    // Global setup
    setup()
    code := m.Run()
    // Global teardown
    teardown()
    os.Exit(code)
}
```

### Per-Test Setup

```go
func setupTest(t *testing.T) (*Core, func()) {
    t.Helper()

    ctrl := gomock.NewController(t)
    mockRepo := mocks.NewMockRepo(ctrl)
    core := &Core{repo: mockRepo}

    cleanup := func() {
        ctrl.Finish()
    }
    return core, cleanup
}

func TestSomething(t *testing.T) {
    core, cleanup := setupTest(t)
    defer cleanup()
    // ... test code
}
```

---

## Async and Concurrent Tests

### Channel-Based (preferred over time.Sleep)

```go
func TestAsync(t *testing.T) {
    done := make(chan struct{})
    go func() {
        defer close(done)
        worker.Process(ctx, input)
    }()

    select {
    case <-done:
        // success — verify results
    case <-time.After(5 * time.Second):
        t.Fatal("timed out waiting for async operation")
    }
}
```

### Race Detection

Run tests with `-race` flag:
```bash
go test -race ./...
```

---

## Test Helper Functions

Every helper MUST call `t.Helper()`:

```go
func assertOrderEquals(t *testing.T, expected, actual *Order) {
    t.Helper()
    assert.Equal(t, expected.ID, actual.ID)
    assert.Equal(t, expected.Amount, actual.Amount)
    assert.Equal(t, expected.Status, actual.Status)
}

func createTestContext(t *testing.T) context.Context {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    t.Cleanup(cancel)
    return ctx
}
```

---

## Anti-Patterns

| Anti-Pattern | Correct Approach |
|---|---|
| `time.Sleep(2 * time.Second)` | Use channels + `select` + `time.After` |
| Testing implementation (which mock was called) | Test behavior (what was returned) |
| `db, _ := setup()` (ignoring errors) | `require.NoError(t, err)` on setup |
| Tests depend on execution order | Each `t.Run` must be independent |
| Hardcoded file paths | Use `testdata/` directory with `os.TempDir()` |
| Testing private functions directly | Test through public API |
| Shared mutable state between tests | Fresh setup per `t.Run` |
| `t.Errorf` without `t.FailNow` for critical failures | Use `require` for hard fails |
| `assert.NotNil(t, result)` as the only value assertion | Assert specific field values: `assert.Equal(t, expected.ID, result.ID)` |
| Only asserting `NoError` + `NotNil` on success path | Add value assertions for key fields — tests should fail if the return value is wrong, not just if it's nil |

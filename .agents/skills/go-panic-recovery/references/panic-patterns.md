# Go Panic/Recover Patterns Reference

Detailed reference for panic scenarios, recovery patterns, and edge cases in Go.

## Table of Contents

1. Runtime Panic Scenarios
2. Recovery Patterns by Context
3. Common Pitfalls
4. Testing Panic Recovery
5. Performance Considerations

## 1. Runtime Panic Scenarios

### Nil Pointer Dereference

```go
type Config struct {
    Cache *CacheClient
}

func (c *Config) GetData() {
    // Panics if c.Cache is nil
    c.Cache.Get("key")
}
```

**Why it matters**: Even with nil checks at initialization, race conditions or lazy initialization can introduce nil pointers.

### Nil Map Assignment

```go
var m map[string]int  // nil map
m["key"] = 42  // PANIC: assignment to entry in nil map

// Reading is OK
val := m["key"]  // Returns zero value, no panic
```

### Out of Bounds Access

```go
slice := []string{"a", "b", "c"}
item := slice[5]  // PANIC: index out of range [5] with length 3

// Safe alternative
if len(slice) > 5 {
    item := slice[5]
}
```

### Type Assertion Without Check

```go
// Unsafe
value := someInterface.(string)  // Panics if not a string

// Safe
value, ok := someInterface.(string)
if !ok {
    return errors.New("not a string")
}
```

### Channel Operations

```go
ch := make(chan int)
close(ch)

// Reading is OK
val, ok := <-ch  // ok = false, val = 0

// Writing panics
ch <- 42  // PANIC: send on closed channel
```

### Division by Zero (integers only)

```go
var x int = 10
var y int = 0
result := x / y  // PANIC: integer divide by zero

// Float division returns Inf
var a float64 = 10.0
var b float64 = 0.0
result := a / b  // +Inf (no panic)
```

## 2. Recovery Patterns by Context

### Basic Goroutine Recovery

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered from panic: %v", r)
            debug.PrintStack()  // Optional: print stack trace
        }
    }()
    // work that might panic
}()
```

### WaitGroup with Error Propagation

```go
var (
    wg  sync.WaitGroup
    err error
    mu  sync.Mutex
)

wg.Add(1)
go func() {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            mu.Lock()
            err = fmt.Errorf("panic: %v", r)
            mu.Unlock()
        }
    }()
    // work
}()

wg.Wait()
if err != nil {
    return err
}
```

### Error Channel Pattern

```go
errChan := make(chan error, numGoroutines)

for i := 0; i < numGoroutines; i++ {
    go func(id int) {
        defer func() {
            if r := recover(); r != nil {
                errChan <- fmt.Errorf("goroutine %d panicked: %v", id, r)
            }
        }()
        // work
        errChan <- nil  // Success
    }(i)
}

// Collect errors
for i := 0; i < numGoroutines; i++ {
    if err := <-errChan; err != nil {
        log.Printf("Error: %v", err)
    }
}
```

### Context-Aware Recovery

```go
go func(ctx context.Context) {
    defer func() {
        if r := recover(); r != nil {
            select {
            case <-ctx.Done():
                // Context cancelled, don't log panic as error
                log.Info("Goroutine cancelled", "panic", r)
            default:
                // Unexpected panic
                log.Error("Unexpected panic", "panic", r)
            }
        }
    }()
    // work
}(ctx)
```

### HTTP Handler Recovery (Middleware)

```go
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Error("Handler panicked", "error", err, "path", r.URL.Path)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

**Important**: Middleware only protects the handler goroutine, not spawned goroutines.

### Server Lifecycle Recovery

```go
func StartServer(ctx context.Context, server *Server) {
    wg := &sync.WaitGroup{}

    // HTTP server
    wg.Add(1)
    go func() {
        defer wg.Done()
        defer func() {
            if r := recover(); r != nil {
                log.Error("HTTP server panicked", "panic", r)
            }
        }()
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Error("HTTP server error", "error", err)
        }
    }()

    // Metrics server
    wg.Add(1)
    go func() {
        defer wg.Done()
        defer func() {
            if r := recover(); r != nil {
                log.Error("Metrics server panicked", "panic", r)
            }
        }()
        if err := metricsServer.ListenAndServe(); err != nil {
            log.Error("Metrics server error", "error", err)
        }
    }()

    wg.Wait()
}
```

## 3. Common Pitfalls

### Pitfall 1: Deferred recover() in Wrong Goroutine

```go
// WRONG - recover() in parent doesn't help child
defer func() {
    if r := recover(); r != nil {
        log.Error("Recovered", "panic", r)
    }
}()

go func() {
    panic("child panic")  // NOT RECOVERED - crashes process
}()
```

```go
// CORRECT - recover() in same goroutine
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Error("Recovered", "panic", r)
        }
    }()
    panic("child panic")  // Recovered successfully
}()
```

### Pitfall 2: Ignoring Panic Return Value

```go
// Less useful
defer func() {
    recover()  // Silently swallows panic
}()

// Better - log it
defer func() {
    if r := recover(); r != nil {
        log.Error("Panic recovered", "panic", r)
    }
}()

// Best - log with stack trace
defer func() {
    if r := recover(); r != nil {
        log.Error("Panic recovered", "panic", r, "stack", string(debug.Stack()))
    }
}()
```

### Pitfall 3: recover() Outside defer

```go
// WRONG - recover() must be called directly by deferred function
func handlePanic() {
    if r := recover(); r != nil {  // Always returns nil
        log.Error("Panic", "panic", r)
    }
}

defer handlePanic()  // Won't work

// CORRECT
defer func() {
    if r := recover(); r != nil {  // Works correctly
        log.Error("Panic", "panic", r)
    }
}()
```

### Pitfall 4: Race Condition in Error Assignment

```go
// WRONG - concurrent writes to err
var err error
for i := 0; i < 10; i++ {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("panic: %v", r)  // Race condition!
            }
        }()
        // work
    }()
}

// CORRECT - use mutex or error channel
var (
    err error
    mu  sync.Mutex
)
for i := 0; i < 10; i++ {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                mu.Lock()
                err = fmt.Errorf("panic: %v", r)
                mu.Unlock()
            }
        }()
        // work
    }()
}
```

## 4. Testing Panic Recovery

### Test That Panic is Recovered

```go
func TestPanicRecovery(t *testing.T) {
    done := make(chan bool)

    go func() {
        defer func() {
            if r := recover(); r != nil {
                t.Log("Panic recovered successfully")
                done <- true
            }
        }()
        panic("test panic")
    }()

    select {
    case <-done:
        // Success
    case <-time.After(1 * time.Second):
        t.Fatal("Panic was not recovered")
    }
}
```

### Test That Function Panics (Expected Panic)

```go
func TestExpectedPanic(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Error("Expected panic but didn't get one")
        }
    }()

    functionThatShouldPanic()
}
```

## 5. Performance Considerations

### Overhead of defer/recover

```go
// Benchmark results (approximate)
// Function call:     1 ns
// defer call:        50 ns  (50x slower)
// panic/recover:     500 ns (500x slower)
```

**Key insights**:
- `defer` has ~50x overhead compared to direct calls
- `panic/recover` has ~500x overhead
- **But**: In goroutines, the alternative is crashing the entire process
- **Verdict**: The overhead is negligible compared to the cost of a service crash

### When to Skip Recovery

Skip panic recovery when:
1. In test files (let tests fail fast)
2. In init() functions (want to fail fast at startup)
3. In main goroutine with proper error handling (middleware handles it)
4. Performance-critical hot paths where goroutine never panics

### Recovery Best Practices

1. **Always recover in spawned goroutines** (default safe choice)
2. **Log with context** (request ID, merchant ID, operation name)
3. **Include stack traces** in development/staging environments
4. **Monitor panic rates** as a production metric
5. **Set up alerts** for panic spike detection

## Edge Cases

### Panic with nil Value

```go
panic(nil)  // Valid but unusual

defer func() {
    r := recover()
    if r != nil {
        // This block won't execute for panic(nil)
    }
}()
```

To catch nil panics:
```go
defer func() {
    r := recover()
    // Check explicitly, don't rely on r != nil
    if r != nil || recover() != nil {
        log.Error("Panic detected")
    }
}()
```

### Nested Panics

```go
defer func() {
    if r := recover(); r != nil {
        log.Error("First panic", "panic", r)
        panic("second panic")  // New panic, not recovered
    }
}()
panic("first panic")
```

If recovery handler panics, the new panic propagates.

### Panic in defer Chain

```go
defer func() { panic("third") }()
defer func() { panic("second") }()
panic("first")

// Only "third" is seen - earlier panics are lost
```

## Structured Logging Examples

### With Standard Logger

```go
defer func() {
    if r := recover(); r != nil {
        log.Error("PANIC_OCCURRED",
            "panic", r,
            "goroutine", "worker-pool",
            "merchant_id", merchantID,
            "request_id", requestID,
            "stack", string(debug.Stack()),
        )
    }
}()
```

### With Context Fields

```go
type contextKey string

const requestIDKey contextKey = "request_id"

func processWithContext(ctx context.Context) {
    defer func() {
        if r := recover(); r != nil {
            requestID := ctx.Value(requestIDKey)
            log.Error("PANIC_PROCESS",
                "panic", r,
                "request_id", requestID,
            )
        }
    }()
    // work
}
```

## Summary Checklist

When adding panic recovery to a goroutine, ensure:

- ✅ `defer func()` is first statement in goroutine
- ✅ `recover()` is called directly in defer (not via helper function)
- ✅ Panic value is logged with relevant context
- ✅ Error is propagated if goroutine is part of error-handling flow
- ✅ WaitGroup.Done() comes before panic recovery defer
- ✅ Mutex/channel used for concurrent error assignment
- ✅ Context values included in log for tracing
- ✅ Production code has recovery; test code may skip

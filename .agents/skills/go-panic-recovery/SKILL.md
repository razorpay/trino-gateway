---
name: go-panic-recovery
description: Analyzes Go codebases to identify unprotected goroutines and systematically adds panic recovery handlers to prevent service crashes. Use when users request panic handling improvements, service resilience enhancements, or goroutine safety analysis. Triggers include "add panic recovery", "find unhandled panics", "protect goroutines", "make service resilient to panics", "scan for unsafe goroutines", or any request to prevent panics from crashing Go services.
---

# Go Panic Recovery

Systematically identifies and protects unprotected goroutines in Go codebases with panic recovery handlers.

## Why Panic Recovery Matters

In Go, **unrecovered panics in goroutines crash the entire process**. Even if functions use proper error handling (error returns, "ok" patterns), runtime panics can still occur:

- Nil pointer dereferences
- Out-of-bounds array/slice access
- Type assertions without "ok" pattern
- Sending to closed channels
- Third-party library panics
- Explicit panic() calls in dependencies

**Critical difference**: Main goroutine panics can be caught by HTTP middleware, but **spawned goroutines bypass all middleware** and crash the entire service.

## Workflow

### Step 0: Fix Concurrent Map Panics in Error Libraries (CRITICAL)

**Before adding panic recovery**, check if the codebase uses `github.com/razorpay/goutils/errors`:

```bash
grep -r "errors\.New(" --include="*.go" | grep -v vendor | head -10
```

If found, **fix concurrent map access bug first** before adding goroutine panic recovery.

**Why:** The goutils/errors library has a fatal concurrent map bug that cannot be caught by panic recovery. Must pre-register all error codes at startup.

**⚠️ CRITICAL VERIFICATION REQUIRED:**
After implementing the fix, you MUST verify the maps are populated:
```bash
# Check map entry counts - must be > 0, not empty!
grep -c '^\s*[A-Z].*:' internal/rzperrors/error_code.go  # Should be 200+
grep -c '^\s*[A-Z].*:' internal/trace/trace.go           # Should be 1000+

# Visual check - should see constant names, not empty {}
tail -10 internal/rzperrors/error_code.go
```

**If maps are empty (count = 0), the fix is NON-FUNCTIONAL. DO NOT PROCEED.**

**See detailed fix instructions:** [references/concurrent-map-fix.md](references/concurrent-map-fix.md)

**Key lessons from implementation:**
- Use Python script for reliable map generation (AWK/sed can fail silently)
- Always verify map population before proceeding
- Use ALL three file discovery patterns to avoid missing files
- Format: `ConstName: ConstName,` NOT `ConstName: "STRING",`

### Step 1: Scan for Unprotected Goroutines

Use the Explore agent to scan the entire codebase:

```
Use Task tool with subagent_type=Explore, thoroughness="very thorough"
Prompt: "Find all goroutines in production Go code (exclude _test.go files) that lack panic recovery handlers. Look for 'go func()' or 'go someFunction()' patterns. Check if they have 'defer func() { recover() }' or similar panic handling."
```

The agent will identify:
- All `go func()` and `go someFunction()` calls
- Which goroutines already have panic recovery
- Which goroutines are unprotected
- Location (file:line) of each unprotected goroutine

### Step 2: Categorize and Prioritize

Focus on **production code only**:
- ✅ Include: All goroutines in non-test files
- ❌ Exclude: Files ending in `_test.go`
- ❌ Exclude: Routes protected by HTTP middleware (user already confirmed middleware exists)

Prioritize by risk level:
1. **Critical**: Fire-and-forget goroutines (no WaitGroup, errors ignored)
2. **High**: Parallel processing with WaitGroups (one panic blocks others)
3. **Medium**: Background workers with error channels

### Step 3: Apply Panic Recovery Pattern

For each unprotected goroutine, add this pattern:

**Fire-and-forget goroutines:**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error(ctx, "PANIC_DESCRIPTION", "panic", r, "context_key", contextValue)
        }
    }()
    // original goroutine code
}()
```

**WaitGroup goroutines with error handling:**
```go
go func() {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            logger.Error(ctx, "PANIC_DESCRIPTION", "panic", r, "context_key", contextValue)
            errVar = fmt.Errorf("panic: %v", r)
            // or: errChan <- fmt.Errorf("panic: %v", r)
        }
    }()
    // original goroutine code
}()
```

**Naming convention for panic logs:**
- Use descriptive ALL_CAPS names: `PANIC_CACHE_INVALIDATION`, `PANIC_SEND_NOTIFICATION`, `PANIC_PROCESS_BATCH`
- Include relevant context fields: merchant_id, request_id, gateway, etc.

### Step 4: Provide Context to User

After each file modification, output a brief summary explaining WHY the change was made:

**Template:**
```
File: internal/services/example.go - 3 goroutines
Why: [Specific reason why these goroutines need protection]
Example: "Parallel API calls to external services. If one response has unexpected format,
type assertion could panic and crash all 3 parallel operations."
```

**Common reasons:**
- External API/service calls that could panic on unexpected responses
- Cache operations where `cache.GetCache()` panics on Redis setup failure
- Fire-and-forget background tasks that shouldn't crash main flow
- Parallel processing where one panic blocks WaitGroup
- Server lifecycle goroutines (metrics, gRPC, HTTP, signal handlers)

### Step 5: Verify Compilation

After all modifications:

```bash
go build ./...
```

Ensure all changes compile without errors.

### Step 6: Create Branch, Commit, Push, and Create PR

After successful compilation:

1. **Verify gh CLI authentication**: `gh auth status`
2. **Create timestamped branch**: `panic-fix/goroutine-recovery-${TIMESTAMP}`
3. **Commit changes** with goroutine count and file list
4. **Push to remote** with `-u origin`
5. **Create PR** with detailed summary
6. **Provide completion summary** with PR link

**See detailed workflow:** [references/git-workflow.md](references/git-workflow.md)

## Example Scenarios

**Scenario 1: Cache invalidation**
```go
// Before
go r.deleteAllDiscrepanciesFromCache(ctx)

// After
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error(ctx, "PANIC_DELETE_CACHE", "panic", r)
        }
    }()
    r.deleteAllDiscrepanciesFromCache(ctx)
}()

// Why: cache.GetCache() panics if Redis setup fails (see cache.go:91)
```

**Scenario 2: Parallel merchant data fetching**
```go
// Before
go func() {
    defer wg.Done()
    merchantInfo, err = apiService.GetMerchantInfo(ctx, merchantId)
}()

// After
go func() {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            logger.Error(ctx, "PANIC_GET_MERCHANT_INFO", "panic", r, "merchant_id", merchantId)
            err = fmt.Errorf("panic: %v", r)
        }
    }()
    merchantInfo, err = apiService.GetMerchantInfo(ctx, merchantId)
}()

// Why: External API call. Network issues, unexpected JSON structure, or nil pointer
// in response parsing could panic. Without recovery, WaitGroup hangs or service crashes.
```

**Scenario 3: Server lifecycle goroutines**
```go
// Before
go func() {
    err := server.Run(ctx)
    if err != nil {
        logger.Error(ctx, "server error", "error", err)
    }
}()

// After
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error(ctx, "PANIC_SERVER_RUN", "panic", r)
        }
    }()
    err := server.Run(ctx)
    if err != nil {
        logger.Error(ctx, "server error", "error", err)
    }
}()

// Why: Server startup panics shouldn't crash the entire application. Metrics server
// failure shouldn't kill main HTTP server, etc.
```

## Key Principles

1. **Defense-in-depth**: Even with proper error handling, runtime panics can occur
2. **Goroutine isolation**: Each goroutine must protect itself; parent context can't help
3. **Context preservation**: Always log relevant context (IDs, keys, values) for debugging
4. **Low cost, high value**: Minimal performance overhead, prevents catastrophic crashes
5. **Production-only**: Skip test files unless explicitly requested

## Common Patterns to Recognize

**Pattern 1: Fire-and-forget** (no WaitGroup)
- Cache invalidation
- Background notifications
- Metrics/logging operations
- Signal handlers

**Pattern 2: Parallel processing** (WaitGroup + error collection)
- Fetching from multiple services
- Processing batches in parallel
- Validating multiple items concurrently

**Pattern 3: Long-running workers**
- Server lifecycle (HTTP, gRPC, metrics)
- Event listeners
- Queue consumers

For detailed Go panic/recover patterns and edge cases, see [references/panic-patterns.md](references/panic-patterns.md).

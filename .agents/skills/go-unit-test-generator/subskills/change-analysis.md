# Phase 1: Change Analysis

Analyze the git diff to identify all Go symbols (functions, methods, types) that were added, modified, or deleted in the current branch.

## Step 1: Get the Diff

```bash
# Find the merge base with the default branch
BASE=$(git merge-base HEAD origin/master 2>/dev/null || git merge-base HEAD origin/main)

# Get only Go files changed
git diff --name-only "$BASE"...HEAD -- '*.go' | grep -v '_test.go' | grep -v 'mock'
```

## Step 2: Claude-Native Symbol Analysis

**Do NOT run `scripts/analyze-diff.sh`.** Claude analyzes changed files directly using Go language understanding. This handles multi-line signatures, generics, complex receivers, and build tags that regex-based parsing cannot.

For each changed file from Step 1:

1. **Read the current version** of the file using the Read tool
2. **Read the base version** for comparison:
   ```bash
   git show "$BASE":<file-path> 2>/dev/null
   ```
3. **Identify symbols** by reading and understanding the Go source:
   - **New functions/methods:** present in current version, absent in base version
   - **Modified functions/methods:** present in both but signature or body changed
   - **Deleted functions/methods:** present in base version, absent in current version
   - For each symbol, extract: name, receiver (if method), file path, line number
   - Correctly handles: multi-line signatures, generic type parameters (`[T any]`), complex receivers (`*Service[Config]`), build-tag-excluded functions

4. **Map source files to test files:**
   ```bash
   TEST_FILE="${FILE%%.go}_test.go"
   test -f "$TEST_FILE" && echo "exists:true" || echo "exists:false"
   ```

5. **Structure findings as JSON** (same schema for backward compatibility):
   ```json
   {
     "new_functions": [
       {"name": "CreateOrder", "file": "internal/core/order.go", "line": 42, "receiver": "Core"}
     ],
     "modified_functions": [
       {"name": "ProcessPayment", "file": "internal/core/payment.go", "line": 88, "receiver": "Core"}
     ],
     "deleted_functions": [
       {"name": "OldHandler", "file": "internal/server/handler.go", "line": 0, "receiver": ""}
     ],
     "test_files": {
       "internal/core/order.go": {"test_file": "internal/core/order_test.go", "exists": false},
       "internal/core/payment.go": {"test_file": "internal/core/payment_test.go", "exists": true},
       "internal/server/handler.go": {"test_file": "internal/server/handler_test.go", "exists": true}
     }
   }
   ```

**Why Claude-native instead of bash regex:**
- Handles multi-line function signatures spanning 2+ lines
- Handles Go generics: `func Process[T constraints.Ordered](items []T) []T`
- Handles complex receivers: `func (s *Service[Config]) Handle(ctx context.Context) error`
- Handles trailing comments, build tags, unconventional formatting
- Understands function body changes (not just signature changes) for modified detection

**CRITICAL: The `test_files` mapping tells Phase 3 where to write tests:**
- `exists: true` → APPEND to existing test file
- `exists: false` → CREATE new test file in the same folder

## Step 3: Map Symbols to Existing Tests

For each symbol in the diff output:

1. Check if a corresponding `*_test.go` exists in the same package:
   ```bash
   # For internal/core/order.go → check internal/core/order_test.go
   ls "${FILE%%.go}_test.go" 2>/dev/null
   # Also check for package-level test files
   ls "$(dirname $FILE)/*_test.go" 2>/dev/null
   ```

2. If a test file exists, grep for test functions that reference the changed symbol:
   ```bash
   grep -n "func Test.*${FUNC_NAME}" "${FILE%%.go}_test.go"
   ```

3. Classify each symbol:

| Symbol State | Test Exists? | Action |
|---|---|---|
| New function | No | Phase 3: Generate new test |
| New function | Yes (partial) | Phase 3: Add test cases |
| Modified function | Yes | Phase 2: Update existing test |
| Modified function | No | Phase 3: Generate new test |
| Deleted function | Yes | Phase 2: Remove/update test |
| Deleted function | No | No action |

## Step 4: Identify Affected Packages

Collect the list of Go packages that contain changes:
```bash
git diff --name-only "$BASE"...HEAD -- '*.go' | xargs -I{} dirname {} | sort -u
```

These packages are the scope for coverage measurement in Phase 4.

## Step 5: Detect Dependencies

For each changed function, identify:
- **Interfaces it depends on** (these need mocks)
- **Structs it constructs** (these need test fixtures)
- **External service calls** (HTTP, gRPC, DB, cache, queue)
- **Error types it returns** (these need assertion patterns)

Read the function body and its receiver struct to extract:
```go
// If function signature includes an interface parameter or struct field:
type Core struct {
    orderRepo   OrderRepository    // → needs mock
    paymentSvc  PaymentService     // → needs mock
    db          *sql.DB            // → needs mock or test DB
}
```

## Output

Present findings to proceed:

```
## Change Analysis Results

### New Functions (need new tests)
- `Core.CreateOrder` in internal/core/order.go:42
- `HandleWebhook` in internal/server/webhook.go:15

### Modified Functions (need test updates)
- `Core.ProcessPayment` in internal/core/payment.go:88
  - Existing test: internal/core/payment_test.go:TestCore_ProcessPayment

### Deleted Functions (tests to clean up)
- `OldHandler` in internal/server/handler.go (removed)
  - Existing test: internal/server/handler_test.go:TestOldHandler → DELETE

### Affected Packages
- internal/core
- internal/server

### Dependencies to Mock
- OrderRepository (interface)
- PaymentService (interface)
```

Proceed to Phase 2 (if existing tests need updates) or Phase 3 (if only new tests needed).

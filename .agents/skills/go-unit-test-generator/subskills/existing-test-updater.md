# Phase 2: Existing Test Updater

Update existing tests that are affected by branch changes. Surgical edits only — never rewrite a working test.

## Principles

1. **Minimal change**: Only modify what the code change broke
2. **Preserve style**: Match the existing test's naming, structure, and assertion patterns
3. **Preserve coverage**: Don't remove test cases unless the code path was deleted
4. **Compile first**: Ensure tests compile before checking semantics

## Step 1: Compile Check

Run a compile-only check on affected test packages:
```bash
go test -run=^$ -count=0 ./internal/core/... 2>&1
```

If compilation fails, the error output tells you exactly what changed:
- `undefined: OldFunction` → function was renamed or deleted
- `too many arguments in call` → function signature changed
- `cannot use X (type A) as type B` → type changed

## Step 2: Fix Compilation Errors

For each compilation error, apply the minimal fix:

### Function Renamed
```go
// Before (broken):
got, err := core.OldName(ctx, input)
// After (fixed):
got, err := core.NewName(ctx, input)
```

### Function Signature Changed
```go
// Before (broken) — function gained a parameter:
got, err := core.Process(ctx, orderID)
// After (fixed):
got, err := core.Process(ctx, orderID, defaultOptions)
```

Read the new function signature from the source to determine the correct fix.

### Return Type Changed
```go
// Before (broken) — function now returns (Result, error) instead of error:
err := core.Process(ctx, input)
// After (fixed):
result, err := core.Process(ctx, input)
// Add assertion for result if meaningful:
assert.NotNil(t, result)
```

### Type Changed
```go
// Before (broken):
input := OldStruct{Field: "value"}
// After (fixed):
input := NewStruct{Field: "value", NewField: "default"}
```

## Step 3: Fix Semantic Mismatches

After compilation passes, check if test expectations are still valid:

### Mock Expectations

If a function now calls a new dependency:
```go
// Add the new mock expectation:
mockNewDep.EXPECT().Validate(gomock.Any()).Return(nil)
```

If a function no longer calls a dependency:
```go
// Remove the stale expectation (it will cause "unexpected call" failures)
// DELETE: mockOldDep.EXPECT().Process(gomock.Any()).Return(nil)
```

### Assertion Values

If the function's behavior changed (e.g., returns different error type):
```go
// Before:
assert.Equal(t, errorclass.ErrBadRequest, err.Class())
// After (if the error class changed):
assert.Equal(t, errorclass.ErrValidation, err.Class())
```

Read the updated function body to determine the correct expected values.

### New Required Fields

If a struct gained a required field:
```go
// Before:
input := CreateOrderRequest{Amount: 100}
// After:
input := CreateOrderRequest{Amount: 100, Currency: "INR"}
```

## Step 4: Handle Deleted Code

For functions/methods that were deleted:

1. **Remove the entire test function** if it only tests deleted code:
   ```go
   // DELETE entire function:
   // func TestOldHandler(t *testing.T) { ... }
   ```

2. **Remove specific table-driven cases** if only some cases tested deleted paths:
   ```go
   // DELETE this case from the tests slice:
   // {
   //     name: "old code path",
   //     ...
   // },
   ```

3. **Update helper functions** that reference deleted code:
   - If the helper is only used by deleted tests → delete the helper too
   - If the helper is shared → update it to remove deleted references

## Step 5: Run Compile Check Again

After all fixes:
```bash
go test -run=^$ -count=0 ./... 2>&1
```

If still failing, repeat Steps 2-4 for remaining errors. Maximum 2 passes — if still broken after 2 passes, flag the test for manual review and proceed to Phase 3.

## Anti-Patterns (DO NOT)

- Do NOT rewrite an entire test function when only one line needs fixing
- Do NOT change test names (breaks CI history and grep-ability)
- Do NOT add new test cases in this phase (that's Phase 3)
- Do NOT change assertion style (e.g., switching from `assert` to `require`)
- Do NOT modify tests for unchanged functions
- Do NOT remove error-path test cases unless the error path was removed from code

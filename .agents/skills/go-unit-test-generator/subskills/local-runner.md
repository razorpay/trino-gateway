# Phase 4: Local Test Execution Loop

Run tests locally, parse coverage, and iterate until all tests pass with >= 95% coverage. Maximum 5 iterations.

## Step 1: Initial Test Run

Execute the coverage script on affected packages:

```bash
scripts/run-coverage.sh
```

This runs:
```bash
go test -coverprofile=coverage.out -covermode=atomic -count=1 -timeout=300s ./...
```

Flags explained:
- `-coverprofile=coverage.out`: Generate coverage data
- `-covermode=atomic`: Accurate coverage for concurrent code
- `-count=1`: Disable test caching (always re-run)
- `-timeout=300s`: 5-minute timeout per package

## Step 1.5: Build Error Detection & Resolution

Before running tests, check if the code compiles. If build errors occur, attempt to fix them automatically.

### Detect Build Errors

```bash
# Try to build the changed packages
go build ./... 2>&1 | tee build.log
```

Look for common Go compilation errors:
- `undefined` — missing import or typo in function/type name
- `no method named` — interface mismatch or missing method
- `cannot use` — type mismatch
- `syntax error` — malformed code in generated tests or modified source

### Common Build Error Patterns & Auto-Fixes

| Error Pattern | Cause | Fix Strategy |
|---|---|---|
| `undefined: <name>` | Missing import or wrong reference | Add correct import or use fully qualified name |
| `cannot use <type> as <type>` | Type mismatch in mock setup or assertions | Check function signature change in source, update mock setup |
| `no method named <method>` | Interface changed or mock incomplete | Regenerate mocks with mockgen or add missing method stub |
| `is not an interface` | Trying to mock a concrete type | Switch to manual stub or check if interface exists |
| `syntax error` | Malformed test code in append operations | Review appended test functions for syntax, fix indentation |

### AI-Powered Build Error Resolution (Max 5 Attempts)

For each build error, use Claude's reasoning to intelligently solve it. This works across any Go service or codebase.

**For each attempt:**

1. **Capture the actual build error** from `go build ./...` output:
   - Extract the full error message as-is: `[FILE]:[LINE]:[COL]: [ERROR_MSG]`
   - Example: `core.go:42:15: undefined: SomeType`
   - Note: Will be different for each repo and each error

2. **Extract error details** dynamically from build output:
   ```bash
   BUILD_ERROR=$(go build ./... 2>&1 | head -1)
   # Parses to: FILE, LINE_NUMBER, ERROR_MESSAGE
   ```

3. **Read context** from the actual error location in this repo:
   ```bash
   # Read the problematic line and surrounding context (works for any file)
   sed -n "$((LINE-5)),$((LINE+5))p" "$FILE"
   
   # Search this repo's codebase for the missing identifier
   grep -r "$(extract_identifier_from_error)" . --include="*.go" | grep -v test | grep -v mock | head -10
   ```

4. **Invoke Claude's AI reasoning** with the ACTUAL error:
   - **The error message**: [paste exact error from build output]
   - **The problematic code**: [lines from FILE at LINE number]
   - **This repo's structure**: [search results showing where the identifier is defined]
   - **Question**: "Here's a Go build error in this codebase. What's the root cause and how should I fix it?"

5. **Claude analyzes intelligently** using actual code understanding:
   - Is this a type/function that was renamed?
   - Was it moved to a different package/location?
   - Is an import missing?
   - Did a function signature change in the source code?
   - Is the mock/test outdated or incomplete?
   - Is the test using an old API that no longer exists?
   - Does the test code have syntax errors?

6. **Claude suggests a contextual fix** based on understanding the actual error:
   - Add missing import with correct path
   - Update mock expectations to match new signature
   - Regenerate mocks if interface changed
   - Fix type references to point to correct package
   - Update test setup to match new function behavior
   - Correct syntax errors in test code
   - (Type of fix depends on actual error, not hardcoded rules)

7. **Apply Claude's suggested fix** to the codebase

8. **Re-run build** to verify:
   ```bash
   go build ./...
   ```
   - If passes → proceed to test execution
   - If still fails → next iteration: Claude analyzes the new error and suggests fix

### Build Failure Escalation (After 5 AI-Powered Attempts)

If Claude's AI reasoning couldn't resolve the error after 5 attempts, present to developer:

```
## Build Error — Could Not Resolve (5/5 attempts with AI analysis)

### Build Error
[exact error message from go build output]

### Problem Location
[file path and line number from error]

### Problematic Code
[the actual code causing the error, with context]

### Claude's Analysis & Fix Attempts
1. Attempt 1: [Claude's analysis] → [Fix attempted] → [Result: still failed]
2. Attempt 2: [Claude's analysis] → [Fix attempted] → [Result: still failed]
3. Attempt 3: [Claude's analysis] → [Fix attempted] → [Result: still failed]
4. Attempt 4: [Claude's analysis] → [Fix attempted] → [Result: still failed]
5. Attempt 5: [Claude's analysis] → [Fix attempted] → [Result: still failed]

### Most Likely Root Causes (per Claude's AI analysis)
1. [Primary diagnosis from final analysis]
2. [Secondary diagnosis]
3. [Tertiary diagnosis]

### Next Steps for Developer
1. Review the build error and problematic code above
2. Check what changed in the source code (the PR/branch changes)
3. Investigate: type rename, package move, signature change, new dependency?
4. Manually apply fix or provide guidance
5. This requires human context and understanding of the change

### How to Resume
Once you fix the issue:
- Run `go build ./...` to verify it compiles
- Then invoke `/go-unit-test-generator` again to continue with Phase 4 test execution
```

Then **stop and present this to the developer**. Do not proceed to test execution.

**Key difference:** After 5 intelligent attempts by Claude's AI reasoning, if it still fails, the problem likely requires human judgment about the actual change made.

---

## Step 2: Parse Results

### Test Results
Parse the `go test` output for pass/fail status:

```
ok      github.com/razorpay/svc/internal/core     0.342s   coverage: 87.3% of statements
FAIL    github.com/razorpay/svc/internal/server    0.156s
```

- Lines starting with `ok` → package passed
- Lines starting with `FAIL` → package failed

### Coverage Results
Parse per-function coverage:

```bash
go tool cover -func=coverage.out
```

Output:
```
github.com/razorpay/svc/internal/core/order.go:42:    CreateOrder     100.0%
github.com/razorpay/svc/internal/core/payment.go:88:  ProcessPayment  72.0%
total:                                                 (statements)    85.6%
```

See [references/coverage-rules.md](../references/coverage-rules.md) for how to interpret and what to exclude.

## Step 3: Evaluate Pass Condition

**ALL conditions must be met:**

1. All test packages report `ok` (no `FAIL`)
2. Total coverage >= 95%
3. No `--- FAIL:` lines in output
4. No race conditions detected (if `-race` flag is used)

If all conditions met → proceed to Step 3.5 (Quality Verification).

## Step 3.5: Test Quality Verification (Post-Pass Gate)

After all tests pass and coverage >= 95%, verify test quality before declaring Phase 4 complete. This step is **advisory** — it flags issues and attempts auto-fix but does not block Phase 5.

### Assertion Density Check

For each test file generated or modified in Phase 3, Claude reads the test code and evaluates:

1. **Assertions per test case:** Count `assert.*` and `require.*` calls within each `t.Run` block
   - Target: **>= 2 meaningful assertions per test case**
   - "Meaningful" means NOT just `NotNil` or `NoError` alone

2. **Shallow test detection:** Flag any test case where the ONLY assertions are:
   - `assert.NotNil(t, result)` / `require.NotNil(t, result)`
   - `assert.NoError(t, err)` / `require.NoError(t, err)`
   - `assert.NotEmpty(t, result)`
   
   These are necessary but insufficient — they confirm the function ran without error but do not verify correctness of the return value.

3. **Error path coverage:** For each function under test, compare:
   - Number of distinct `return ..., err` / `return ..., fmt.Errorf(...)` paths in the source
   - Number of test cases with `wantErr: true` (or equivalent)
   - Target: every distinct error return path has at least one test case

### Quality Remediation

For each flagged shallow test:

1. Read the source function to determine what the return value should contain
2. Add specific value assertions:
   ```go
   // BEFORE (shallow):
   require.NoError(t, err)
   assert.NotNil(t, result)
   
   // AFTER (meaningful):
   require.NoError(t, err)
   require.NotNil(t, result)
   assert.Equal(t, "order_1", result.ID)
   assert.Equal(t, int64(1000), result.Amount)
   assert.Equal(t, "created", result.Status)
   ```

3. Re-run tests after adding assertions to confirm they still pass

### Quality Scorecard

Present a brief scorecard after verification:

```
Test Quality Check:
├── Assertion density:    2.8 per case  ✅ (target: >= 2)
├── Shallow tests:        1 flagged, 1 fixed  ✅
├── Error path coverage:  8/8 paths covered  ✅
└── Overall: PASS
```

If quality issues cannot be resolved (e.g., return type is opaque or function has no meaningful fields to assert), note them but proceed — the quality check is advisory, not a hard gate.

After Step 3.5 passes → proceed to Phase 5.

## Step 4: Handle Failures (Retry Loop)

### On Test Failure

1. **Extract the failing test name and error**:
   ```
   --- FAIL: TestCore_CreateOrder/error_-_nil_input (0.00s)
       order_test.go:85: Expected nil, got &CreateOrderResponse{...}
   ```

2. **Read the failing test** to understand what assertion failed

3. **Read the source function** to understand correct behavior

4. **Apply targeted fix** — one of:
   - Fix assertion value (expected output changed)
   - Fix mock expectation (new dependency call added/removed)
   - Fix test input (function now validates differently)
   - Fix setup (missing initialization)

5. **Track fix attempt count per test**:
   ```
   TestCore_CreateOrder/error_-_nil_input: attempt 1/2
   ```

6. **If 2 fix attempts fail** → add `t.Skip`:
   ```go
   t.Skip("Could not fix after 2 attempts: assertion mismatch on error response - " +
       "expected ErrBadRequest but got ErrValidation. " +
       "Likely needs manual review of validation logic change.")
   ```

### On Coverage Below 95%

1. **Identify uncovered functions** from `go tool cover -func` output
2. **Prioritize by gap**: Functions with 0% coverage first, then lowest coverage
3. **Generate additional test cases** for uncovered branches:
   - Read the source to find untested `if/else` branches
   - Add table-driven cases that exercise those branches
4. **Re-run tests**

### Iteration Tracking

```
=== ITERATION 1/5 ===
Tests: 42 passed, 3 failed
Coverage: 78.2%
Action: Fix 3 failing tests

=== ITERATION 2/5 ===
Tests: 45 passed, 0 failed
Coverage: 82.1%
Action: Add tests for uncovered functions (ProcessRefund, ValidateWebhook)

=== ITERATION 3/5 ===
Tests: 52 passed, 1 failed
Coverage: 91.4%
Action: Fix TestValidateWebhook/empty_signature, add edge case tests

=== ITERATION 4/5 ===
Tests: 54 passed, 0 failed
Coverage: 96.2%
Action: PASS — all conditions met
```

## Step 5: Escalation (After 5 Iterations)

If 5 iterations are exhausted without meeting the pass condition, present to user:

```
## Local Test Loop Exhausted (5/5 iterations)

### Current State
- Tests: X passed, Y failed, Z skipped
- Coverage: XX.X%
- Target: 95%

### Remaining Failures
1. TestFoo/case_name — <error description>
   - Attempted: <fix 1>, <fix 2>
   - Root cause: <diagnosis>

2. TestBar/case_name — <error description>
   - Attempted: <fix 1>, <fix 2>
   - Root cause: <diagnosis>

### Uncovered Code (top gaps)
1. internal/core/refund.go:ProcessRefund — 0% (complex branching)
2. internal/server/middleware.go:AuthMiddleware — 45% (external dependency)

### Options
1. Lower coverage target to XX% and push
2. Skip remaining failures and push
3. Abort — manual intervention needed
```

## Anti-Patterns

- **No blind retries**: Each iteration must produce new diagnostic information
- **No identical fixes**: If the same fix was tried and failed, try a different approach or skip
- **No test deletion to raise coverage**: Never delete a valid test to improve coverage numbers
- **No `time.Sleep` for flaky tests**: Fix the root cause or add proper synchronization

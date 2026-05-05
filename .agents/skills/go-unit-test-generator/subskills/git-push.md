# Phase 5: Commit & Push

Stage, commit, and push test files after local tests pass. Requires user confirmation before push.

## Pre-Push Verification (CRITICAL GATE)

**ABORT PHASE 5 if ANY of these checks fail. DO NOT PUSH.**

Before staging, verify Phase 4 completed successfully:

1. ✅ **Phase 4 was executed:**
   - Test execution logs exist (coverage.out file present)
   - Phase 4 reported final status: `PASS` or `FAIL`
   - If Phase 4 was skipped or did not run: **STOP and ask user**

2. ✅ **All local tests passed:**
   - `go test ./...` exited with code 0
   - No `FAIL` status in test output
   - No syntax errors in test code
   - Build succeeded (Step 1.5 completed with no unresolved build errors)

3. ✅ **Coverage >= 95%:**
   - Verify from `coverage.out` file: `go tool cover -func=coverage.out | tail -1`
   - Total coverage percentage must be >= 95%
   - If coverage < 95%: **STOP and report gap to user**

4. ✅ **No unrelated source code changes:**
   - `git diff --name-only` shows ONLY modified test files
   - Source code files (`core.go`, `payment.go`, etc.) should NOT be in staged changes
   - If source files are modified: **STOP and clarify which files are intentional**

**If all 4 checks pass:** Proceed to Step 1 (Stage Test Files).  
**If any check fails:** Report the failure and ask user for guidance. Do NOT commit or push.

## Step 1: Stage Test Files Only

```bash
# Stage only test files and mock files
git add $(git diff --name-only --diff-filter=AM | grep -E '(_test\.go|mocks/)')

# Also stage any new test files
git add $(git ls-files --others --exclude-standard | grep -E '(_test\.go|mocks/)')
```

**Verify staged files** — only `*_test.go` and `mocks/*.go` should be staged:
```bash
git diff --cached --name-only
```

If non-test files are staged, unstage them:
```bash
git reset HEAD <non-test-file>
```

## Step 2: Create Commit

Determine the commit message based on what was done:

### If only new tests were added:
```
test: add unit tests for <package-name> (coverage: X%)
```

### If existing tests were updated:
```
test: update unit tests for <package-name> after refactor (coverage: X%)
```

### If both:
```
test: add and update unit tests for <package-name> (coverage: X%)
```

Commit:
```bash
git commit -m "test: add/update unit tests for <package> (coverage: X%)"
```

## Step 3: Show Diff and Request Confirmation

**MANDATORY: Show the user what will be pushed.**

Present:
```
## Ready to Push

**Branch**: <current-branch>
**Commit**: test: add/update unit tests for <package> (coverage: X%)

### Files Changed
- internal/core/order_test.go (new, +142 lines)
- internal/core/payment_test.go (modified, +23 -8 lines)
- internal/mocks/mock_order_repo.go (new, +45 lines)

### Test Results
- Passed: 54
- Failed: 0
- Skipped: 1
- Coverage: 96.2%

Push to origin/<branch>?
```

**Wait for explicit user confirmation before proceeding.**

## Step 4: Push

```bash
git push origin $(git branch --show-current)
```

If push fails due to remote changes:
```bash
git pull --rebase origin $(git branch --show-current)
# Re-run tests after rebase to ensure nothing broke
go test -count=1 ./...
# If tests still pass, push again
git push origin $(git branch --show-current)
```

If push fails for other reasons (auth, permissions), report the error and ask user for guidance.

## Output

After successful push:
```
Pushed to origin/<branch> successfully.
Proceeding to Phase 6: CI Monitor.
```

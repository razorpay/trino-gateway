# Coverage Rules

How to measure, interpret, and enforce Go code coverage.

## Commands

### Generate Coverage Profile

```bash
# All packages
go test -coverprofile=coverage.out -covermode=atomic ./...

# Specific packages only (recommended — scope to changed packages)
go test -coverprofile=coverage.out -covermode=atomic ./internal/core/... ./internal/server/...
```

### View Coverage Summary

```bash
# Per-function breakdown
go tool cover -func=coverage.out

# Output:
# github.com/razorpay/svc/internal/core/order.go:42:    CreateOrder     100.0%
# github.com/razorpay/svc/internal/core/order.go:88:    ValidateOrder   85.7%
# total:                                                 (statements)    92.3%
```

### View Coverage in Browser

```bash
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Coverage Target

**Target: >= 95% statement coverage** for packages containing changed code.

### Scoping

Coverage is measured ONLY on packages that contain files changed in the branch:
```bash
# Get changed packages
CHANGED_PKGS=$(git diff --name-only $(git merge-base HEAD origin/master)...HEAD -- '*.go' | \
    grep -v '_test.go' | xargs -I{} dirname {} | sort -u | sed 's|^|./|')

# Run coverage on those packages only
go test -coverprofile=coverage.out -covermode=atomic $CHANGED_PKGS
```

This prevents unrelated low-coverage packages from failing the threshold.

## What Counts Toward Coverage

| Counts | Doesn't Count |
|--------|--------------|
| Executed `if` branches | Unreachable `default` in exhaustive switches |
| Function calls that return | `panic` paths (unless explicitly tested) |
| Loop bodies that execute | Generated code (`*.pb.go`, `mock_*.go`) |
| Error handling paths | Build-tag-excluded code |

## What to Exclude

### Generated Files

Exclude from coverage measurement:
```bash
# Filter out generated files from coverage
grep -v 'pb.go\|mock_\|_gen.go\|_string.go' coverage.out > coverage-filtered.out
go tool cover -func=coverage-filtered.out
```

### Integration-Only Code

Code that requires external services (DB, Redis, Kafka) and cannot be unit-tested:
```go
//go:build integration

func TestWithRealDB(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
}
```

## Interpreting Low Coverage

When a function shows low coverage, check:

1. **Untested error paths**: Add test cases for each `if err != nil` branch
2. **Untested switch cases**: Add test cases for each `case` value
3. **Early returns**: Add test cases that exercise the early return vs full execution
4. **Nil checks**: Add test cases with nil inputs
5. **Default values**: Add test cases where defaults are used vs overridden

## Coverage Improvement Strategy

Priority order for improving coverage:

1. **0% functions** (completely untested) — highest priority
2. **Functions below 50%** — missing major branches
3. **Functions below 80%** — missing edge cases
4. **Functions below 95%** — minor branches, often error handling

For each uncovered line, read the source to understand WHAT branch is uncovered, then write a test case that exercises that specific branch.

## Parsing Coverage Output Programmatically

```bash
# Get total coverage percentage
TOTAL=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')

# Check threshold
if (( $(echo "$TOTAL < 95.0" | bc -l) )); then
    echo "FAIL: Coverage $TOTAL% is below 95% threshold"
    exit 1
fi
echo "PASS: Coverage $TOTAL%"
```

## Per-Function Coverage Check

```bash
# Find functions below threshold
go tool cover -func=coverage.out | awk '$3 != "100.0%" {print}' | grep -v total
```

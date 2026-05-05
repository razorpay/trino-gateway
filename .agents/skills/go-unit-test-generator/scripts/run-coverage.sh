#!/usr/bin/env bash
# run-coverage.sh — Run Go tests with coverage and parse results
# Usage: scripts/run-coverage.sh [packages...]
# If no packages specified, detects changed packages from git diff
# Output: Test results + coverage summary + pass/fail status

set -uo pipefail

COVERAGE_FILE="coverage.out"
THRESHOLD="${COVERAGE_THRESHOLD:-95.0}"
TIMEOUT="${TEST_TIMEOUT:-300s}"

# Determine packages to test
if [ $# -gt 0 ]; then
    PACKAGES="$*"
else
    BASE=$(git merge-base HEAD origin/master 2>/dev/null || git merge-base HEAD origin/main 2>/dev/null || echo "HEAD~1")
    PACKAGES=$(git diff --name-only "$BASE"...HEAD -- '*.go' | \
        grep -v '_test.go' | \
        grep -v '/mocks/' | \
        grep -v '.pb.go' | \
        xargs -I{} dirname {} 2>/dev/null | \
        sort -u | \
        sed 's|^|./|' || echo "./...")
fi

if [ -z "$PACKAGES" ]; then
    echo "No changed Go packages found. Running all tests."
    PACKAGES="./..."
fi

echo "=== Running Go Tests ==="
echo "Packages: $PACKAGES"
echo "Timeout: $TIMEOUT"
echo "Coverage threshold: $THRESHOLD%"
echo ""

# Run tests with coverage
# shellcheck disable=SC2086
go test \
    -coverprofile="$COVERAGE_FILE" \
    -covermode=atomic \
    -count=1 \
    -timeout="$TIMEOUT" \
    -v \
    $PACKAGES 2>&1 | tee /tmp/go-test-output.txt

TEST_EXIT=$?

echo ""
echo "=== Coverage Summary ==="

if [ -f "$COVERAGE_FILE" ]; then
    # Filter out generated files
    grep -v 'pb.go\|mock_\|_gen.go\|_string.go' "$COVERAGE_FILE" > "${COVERAGE_FILE}.filtered" 2>/dev/null || cp "$COVERAGE_FILE" "${COVERAGE_FILE}.filtered"

    # Show per-function coverage
    go tool cover -func="${COVERAGE_FILE}.filtered" 2>/dev/null || go tool cover -func="$COVERAGE_FILE"

    # Extract total
    TOTAL=$(go tool cover -func="${COVERAGE_FILE}.filtered" 2>/dev/null | grep total | awk '{print $3}' | tr -d '%' || echo "0")

    echo ""
    echo "Total Coverage: ${TOTAL}%"
    echo "Threshold: ${THRESHOLD}%"

    # Check threshold
    PASS_COVERAGE=$(echo "$TOTAL >= $THRESHOLD" | bc -l 2>/dev/null || python3 -c "print(1 if $TOTAL >= $THRESHOLD else 0)")

    if [ "$PASS_COVERAGE" = "1" ]; then
        echo "Coverage: PASS"
    else
        echo "Coverage: FAIL (need ${THRESHOLD}%, got ${TOTAL}%)"
    fi
else
    echo "No coverage file generated."
    TOTAL="0"
    PASS_COVERAGE="0"
fi

# Count pass/fail
PASSED=$(grep -c "^--- PASS:" /tmp/go-test-output.txt 2>/dev/null || echo "0")
FAILED=$(grep -c "^--- FAIL:" /tmp/go-test-output.txt 2>/dev/null || echo "0")
SKIPPED=$(grep -c "^--- SKIP:" /tmp/go-test-output.txt 2>/dev/null || echo "0")

echo ""
echo "=== Test Results ==="
echo "Passed:  $PASSED"
echo "Failed:  $FAILED"
echo "Skipped: $SKIPPED"

# Overall result
if [ "$TEST_EXIT" -eq 0 ] && [ "$PASS_COVERAGE" = "1" ]; then
    echo ""
    echo "=== RESULT: PASS ==="
    exit 0
else
    echo ""
    echo "=== RESULT: FAIL ==="
    if [ "$TEST_EXIT" -ne 0 ]; then
        echo "Reason: Test failures detected"
        echo ""
        echo "=== Failing Tests ==="
        grep "^--- FAIL:" /tmp/go-test-output.txt 2>/dev/null || echo "See output above"
    fi
    if [ "$PASS_COVERAGE" != "1" ]; then
        echo "Reason: Coverage below threshold (${TOTAL}% < ${THRESHOLD}%)"
        echo ""
        echo "=== Uncovered Functions ==="
        go tool cover -func="${COVERAGE_FILE}.filtered" 2>/dev/null | awk '$3 != "100.0%" && !/^total/' | head -20
    fi
    exit 1
fi

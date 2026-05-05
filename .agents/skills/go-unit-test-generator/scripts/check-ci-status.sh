#!/usr/bin/env bash
# check-ci-status.sh — Check GitHub Actions CI status for a branch
# Usage: scripts/check-ci-status.sh [branch-name]
# Output: CI run status, conclusion, and URL

# NOTE: This script is no longer used by the main go-unit-test-generator pipeline.
# Phase 6 now uses a fire-and-forget approach (see subskills/ci-monitor.md).
# This script is retained for standalone use and for the ci-fix companion skill.
# The timeout has been updated from 30 min to 90 min to match real CI durations.

set -uo pipefail

BRANCH="${1:-$(git branch --show-current)}"
MAX_WAIT="${CI_MAX_WAIT:-300}"  # 5 minutes max wait for run to appear
POLL_INTERVAL="${CI_POLL_INTERVAL:-60}"  # 60 seconds between polls

echo "=== CI Status Check ==="
echo "Branch: $BRANCH"
echo ""

# Step 1: Find the latest run for this branch
find_run() {
    gh run list --branch "$BRANCH" --limit 1 --json databaseId,status,conclusion,event,headSha,createdAt,url \
        --jq '.[0]' 2>/dev/null
}

# Wait for a run to appear (CI may take a few seconds to trigger)
ELAPSED=0
RUN_JSON=""
while [ $ELAPSED -lt $MAX_WAIT ]; do
    RUN_JSON=$(find_run)
    if [ -n "$RUN_JSON" ] && [ "$RUN_JSON" != "null" ]; then
        break
    fi
    echo "Waiting for CI run to appear... ($ELAPSED/${MAX_WAIT}s)"
    sleep 10
    ELAPSED=$((ELAPSED + 10))
done

if [ -z "$RUN_JSON" ] || [ "$RUN_JSON" = "null" ]; then
    echo "ERROR: No CI run found for branch '$BRANCH' after ${MAX_WAIT}s"
    echo ""
    echo "Possible causes:"
    echo "  1. Merge conflicts blocking CI — check: gh pr view --json mergeable"
    echo "  2. Workflow not configured for this branch — check .github/workflows/"
    echo "  3. Concurrency group blocking — previous run may still be active"
    exit 1
fi

RUN_ID=$(echo "$RUN_JSON" | jq -r '.databaseId')
STATUS=$(echo "$RUN_JSON" | jq -r '.status')
CONCLUSION=$(echo "$RUN_JSON" | jq -r '.conclusion // "pending"')
URL=$(echo "$RUN_JSON" | jq -r '.url // ""')
SHA=$(echo "$RUN_JSON" | jq -r '.headSha // ""')

echo "Run ID: $RUN_ID"
echo "SHA: ${SHA:0:7}"
echo "Status: $STATUS"
echo "URL: $URL"

# Step 2: If still in progress, poll until complete
if [ "$STATUS" != "completed" ]; then
    echo ""
    echo "=== Monitoring CI Run ==="
    while true; do
        sleep "$POLL_INTERVAL"
        ELAPSED=$((ELAPSED + POLL_INTERVAL))

        RUN_JSON=$(gh run view "$RUN_ID" --json status,conclusion 2>/dev/null)
        STATUS=$(echo "$RUN_JSON" | jq -r '.status')
        CONCLUSION=$(echo "$RUN_JSON" | jq -r '.conclusion // "pending"')

        echo "[$ELAPSED s] Status: $STATUS | Conclusion: $CONCLUSION"

        if [ "$STATUS" = "completed" ]; then
            break
        fi

        if [ $ELAPSED -gt 5400 ]; then
            echo "TIMEOUT: CI run exceeded 90 minutes. Check manually: $URL"
            exit 2
        fi
    done
fi

echo ""
echo "=== CI Result ==="
echo "Conclusion: $CONCLUSION"
echo "URL: $URL"

# Step 3: Output result
if [ "$CONCLUSION" = "success" ]; then
    echo ""
    echo "CI: PASSED"
    exit 0
else
    echo ""
    echo "CI: FAILED"
    echo ""
    echo "=== Failed Job Logs ==="
    gh run view "$RUN_ID" --log-failed 2>/dev/null | tail -100 || echo "Could not fetch logs"
    exit 1
fi

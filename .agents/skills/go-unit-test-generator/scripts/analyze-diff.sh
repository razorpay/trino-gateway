#!/usr/bin/env bash
# DEPRECATED: This script is retained for reference only.
# Phase 1 now uses Claude-native Go analysis instead of regex-based parsing.
# See subskills/change-analysis.md Step 2 for the current approach.
#
# Original purpose: Parse git diff to identify changed Go symbols
# Usage: scripts/analyze-diff.sh [base-branch]
# Output: JSON with new_functions, modified_functions, deleted_functions

set -euo pipefail

BASE_BRANCH="${1:-origin/master}"
MERGE_BASE=$(git merge-base HEAD "$BASE_BRANCH" 2>/dev/null || git merge-base HEAD origin/main 2>/dev/null || echo "HEAD~1")

# Get changed Go files (excluding tests and mocks)
CHANGED_FILES=$(git diff --name-only "$MERGE_BASE"...HEAD -- '*.go' | grep -v '_test.go' | grep -v '/mocks/' | grep -v '.pb.go' || true)

if [ -z "$CHANGED_FILES" ]; then
    echo '{"new_functions":[],"modified_functions":[],"deleted_functions":[],"changed_packages":[]}'
    exit 0
fi

NEW_FUNCS="[]"
MOD_FUNCS="[]"
DEL_FUNCS="[]"

for FILE in $CHANGED_FILES; do
    # Skip if file doesn't exist (deleted file)
    if [ ! -f "$FILE" ]; then
        # Extract function names from the deleted file in the base
        DELETED=$(git show "$MERGE_BASE:$FILE" 2>/dev/null | grep -E '^func ' | sed 's/func //' | sed 's/(.*//' | sed 's/^(//' || true)
        for FUNC in $DELETED; do
            RECEIVER=""
            FUNC_NAME="$FUNC"
            # Parse receiver if present: (r *Receiver) MethodName
            if echo "$FUNC" | grep -q '^\*\?\w\+)'; then
                RECEIVER=$(echo "$FUNC" | sed 's/.*\*\?\(\w\+\)).*/\1/')
                FUNC_NAME=$(echo "$FUNC" | sed 's/^[^)]*) //')
            fi
            DEL_FUNCS=$(echo "$DEL_FUNCS" | jq --arg name "$FUNC_NAME" --arg file "$FILE" --arg recv "$RECEIVER" \
                '. + [{"name": $name, "file": $file, "line": 0, "receiver": $recv}]')
        done
        continue
    fi

    # Get diff for this file — look for added/removed function signatures
    DIFF=$(git diff "$MERGE_BASE"...HEAD -- "$FILE" || true)

    # New functions (lines starting with + func)
    ADDED_FUNCS=$(echo "$DIFF" | grep '^+func ' | sed 's/^+//' || true)
    # Removed functions (lines starting with - func)
    REMOVED_FUNCS=$(echo "$DIFF" | grep '^-func ' | sed 's/^-//' || true)

    # Process added functions
    while IFS= read -r LINE; do
        [ -z "$LINE" ] && continue
        # Extract function name and receiver
        FUNC_NAME=$(echo "$LINE" | sed 's/^func //' | sed 's/(.*$//' | sed 's/ .*//')
        RECEIVER=""
        if echo "$LINE" | grep -q '^func ('; then
            RECEIVER=$(echo "$LINE" | sed 's/^func (\*\?\([^ ]*\).*/\1/' | tr -d ')')
            FUNC_NAME=$(echo "$LINE" | sed 's/^func ([^)]*) //' | sed 's/(.*$//')
        fi
        # Get line number in current file
        LINE_NUM=$(grep -n "^func.*${FUNC_NAME}" "$FILE" 2>/dev/null | head -1 | cut -d: -f1 || echo "0")

        # Check if this function existed before (modified) or is truly new
        if echo "$REMOVED_FUNCS" | grep -q "$FUNC_NAME"; then
            MOD_FUNCS=$(echo "$MOD_FUNCS" | jq --arg name "$FUNC_NAME" --arg file "$FILE" --arg line "$LINE_NUM" --arg recv "$RECEIVER" \
                '. + [{"name": $name, "file": $file, "line": ($line | tonumber), "receiver": $recv}]')
        else
            NEW_FUNCS=$(echo "$NEW_FUNCS" | jq --arg name "$FUNC_NAME" --arg file "$FILE" --arg line "$LINE_NUM" --arg recv "$RECEIVER" \
                '. + [{"name": $name, "file": $file, "line": ($line | tonumber), "receiver": $recv}]')
        fi
    done <<< "$ADDED_FUNCS"

    # Process removed functions (that weren't also added — i.e., truly deleted)
    while IFS= read -r LINE; do
        [ -z "$LINE" ] && continue
        FUNC_NAME=$(echo "$LINE" | sed 's/^func //' | sed 's/(.*$//' | sed 's/ .*//')
        RECEIVER=""
        if echo "$LINE" | grep -q '^func ('; then
            RECEIVER=$(echo "$LINE" | sed 's/^func (\*\?\([^ ]*\).*/\1/' | tr -d ')')
            FUNC_NAME=$(echo "$LINE" | sed 's/^func ([^)]*) //' | sed 's/(.*$//')
        fi
        if ! echo "$ADDED_FUNCS" | grep -q "$FUNC_NAME"; then
            DEL_FUNCS=$(echo "$DEL_FUNCS" | jq --arg name "$FUNC_NAME" --arg file "$FILE" --arg recv "$RECEIVER" \
                '. + [{"name": $name, "file": $file, "line": 0, "receiver": $recv}]')
        fi
    done <<< "$REMOVED_FUNCS"
done

# Get changed packages
CHANGED_PKGS=$(echo "$CHANGED_FILES" | xargs -I{} dirname {} | sort -u | jq -R -s 'split("\n") | map(select(. != ""))')

# Map source files to test files (check which test files exist)
TEST_FILE_MAP="{}"
for FILE in $CHANGED_FILES; do
    TEST_FILE="${FILE%%.go}_test.go"
    TEST_EXISTS="false"
    if [ -f "$TEST_FILE" ]; then
        TEST_EXISTS="true"
    fi
    TEST_FILE_MAP=$(echo "$TEST_FILE_MAP" | jq --arg src "$FILE" --arg test "$TEST_FILE" --arg exists "$TEST_EXISTS" \
        '. + {($src): {"test_file": $test, "exists": ($exists == "true")}}')
done

# Output
jq -n \
    --argjson new "$NEW_FUNCS" \
    --argjson modified "$MOD_FUNCS" \
    --argjson deleted "$DEL_FUNCS" \
    --argjson packages "$CHANGED_PKGS" \
    --argjson test_files "$TEST_FILE_MAP" \
    '{new_functions: $new, modified_functions: $modified, deleted_functions: $deleted, changed_packages: $packages, test_files: $test_files}'

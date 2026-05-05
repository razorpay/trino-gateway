# Git Workflow for Panic Recovery Changes

Complete guide for creating branches, committing, and creating PRs for panic recovery changes.

## 6.1: Verify GitHub CLI Authentication

First, check if gh CLI is authenticated:

```bash
gh auth status
```

**If not authenticated or gh is not installed:**

Output to user:
```
GitHub CLI (gh) is not authenticated. Please authenticate by running:

gh auth login

Follow the prompts to:
1. Select "GitHub.com"
2. Choose your preferred authentication method (HTTPS or SSH)
3. Authenticate via web browser or paste an authentication token

After authentication, run the skill again.
```

**If authentication fails or gh is not installed:**
```
GitHub CLI (gh) is not installed or not in PATH.

Install gh CLI:
- macOS: brew install gh
- Linux: https://github.com/cli/cli/blob/trunk/docs/install_linux.md
- Windows: https://github.com/cli/cli/releases

After installation, run: gh auth login
```

Stop execution and wait for user to complete authentication.

## 6.2: Get Current Branch and Repository Info

```bash
# Get current branch name
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Get repository info
REPO_INFO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
```

Output to user:
```
Current branch: ${CURRENT_BRANCH}
Repository: ${REPO_INFO}
```

## 6.3: Create panic-fix Branch

Generate branch name with timestamp:

```bash
# Generate branch name
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BRANCH_NAME="panic-fix/goroutine-recovery-${TIMESTAMP}"

# Create and checkout new branch
git checkout -b ${BRANCH_NAME}
```

Output to user:
```
✓ Created new branch: ${BRANCH_NAME}
```

## 6.4: Stage and Commit Changes

Get list of modified files:

```bash
# Show what files were changed
git status --short
```

Commit all changes with descriptive message:

```bash
git add -A

# Generate commit message
git commit -m "Add panic recovery to unprotected goroutines

- Protected X goroutines across Y files
- Added defer/recover handlers to prevent service crashes
- Includes proper error logging with context
- Files modified:
  $(git diff --cached --name-only | sed 's/^/  - /')

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

**Commit message format:**
- First line: Clear, imperative summary
- Body: Bulleted list of changes
- Include file count and goroutine count
- List all modified files
- Add co-authored-by tag

Output to user:
```
✓ Committed changes: "Add panic recovery to unprotected goroutines"
  Files: ${FILE_COUNT}
  Goroutines protected: ${GOROUTINE_COUNT}
```

## 6.5: Push Branch to Remote

Push the new branch:

```bash
git push -u origin ${BRANCH_NAME}
```

Output to user:
```
✓ Pushed branch to origin: ${BRANCH_NAME}
```

## 6.6: Create Pull Request

After pushing the branch, automatically create a PR:

```bash
# Get list of modified files and count
MODIFIED_FILES=$(git diff ${CURRENT_BRANCH}..${BRANCH_NAME} --name-only)
FILE_COUNT=$(echo "$MODIFIED_FILES" | wc -l | xargs)
GOROUTINE_COUNT=$(grep -r "defer func()" $MODIFIED_FILES 2>/dev/null | wc -l | xargs)

# Create PR with detailed description
gh pr create \
  --title "Add panic recovery to unprotected goroutines" \
  --body "$(cat <<'EOF'
## Summary
Systematically added panic recovery handlers to protect goroutines from unrecovered panics that could crash the entire service.

## Changes
- Protected ${GOROUTINE_COUNT} goroutines across ${FILE_COUNT} files
- Added `defer func() { recover() }` handlers with proper error logging
- Includes relevant context in panic logs for debugging
- Production code only (excluded test files)

## Why This Matters
In Go, unrecovered panics in goroutines crash the entire process. Even with proper error handling, runtime panics can occur from:
- Nil pointer dereferences
- Type assertions without "ok" pattern
- Sending to closed channels
- Third-party library panics
- Unexpected data formats from external APIs

## Testing
- ✅ Code compiles: \`go build ./...\` passed
- Files modified:
$(echo "$MODIFIED_FILES" | sed 's/^/  - /')

## Risk Assessment
Low risk - defensive changes only. Added panic recovery without modifying business logic.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

Output to user:
```
✓ Pull request created successfully!

View PR: $(gh pr view --json url -q .url)
```

## 6.7: Provide Completion Summary

Output to user:
```
🎉 Panic recovery implementation complete!

Summary:
- Branch: ${BRANCH_NAME}
- Repository: ${REPO_INFO}
- Files modified: ${FILE_COUNT}
- Goroutines protected: ${GOROUTINE_COUNT}
- Pull request: $(gh pr view --json url -q .url)

Next steps:
1. Review the PR and request reviews from your team
2. Address any review comments
3. Merge when approved
4. Switch back to your original branch: git checkout ${CURRENT_BRANCH}
```

## Error Handling

### Git Operation Failures

If any git operation fails:
- Check if working directory is clean before creating branch
- If there are uncommitted changes on current branch, ask user what to do:
  ```
  You have uncommitted changes on ${CURRENT_BRANCH}:

  $(git status --short)

  Options:
  1. Stash changes: git stash
  2. Commit them first
  3. Discard them: git reset --hard

  What would you like to do?
  ```

### PR Creation Failures

If PR creation fails:
- Check if there's already a PR for this branch: `gh pr list --head ${BRANCH_NAME}`
- If gh is not authenticated, show authentication instructions (see 6.1)
- If PR already exists, output:
  ```
  ⚠️  A pull request already exists for this branch.

  View existing PR: $(gh pr view ${BRANCH_NAME} --json url -q .url)
  ```

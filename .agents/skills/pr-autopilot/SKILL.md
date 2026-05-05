---
name: pr-autopilot
description: AI-powered PR co-pilot that autonomously handles CI failures, flaky tests, coverage issues, and AI review comments. Detects failing tests, runs them locally to identify flaky vs genuine failures, auto-generates unit tests following repo patterns, and intelligently processes AI PR review comments. Use when CI checks fail, tests are failing, coverage is low, AI review comments need addressing, or when preparing a PR for merge.
---

# PR Autopilot

## Overview

Your AI-powered PR co-pilot that autonomously handles CI failures, test issues, coverage gaps, and AI review comments. Detects and fixes failing tests, identifies flaky tests, auto-generates unit tests following repo-level patterns, and intelligently processes AI PR review comments with reasoned decisions. Works seamlessly with pr-creator skill to push fixes.

## Workflow

Follow these steps when working on a PR with issues:

### Step 0: Verify GitHub CLI Setup

**CRITICAL:** Check gh CLI is installed and authenticated.

```bash
gh --version && gh auth status
```

If not authenticated, guide user to run: `gh auth login`

### Step 0.5: Verify Service and Local Repo Access

**CRITICAL:** Determine service name and local repository path to access code.

**Check if service/repo is obvious from context:**

```bash
# Get repo name from PR URL or current directory
gh pr view $PR_NUMBER --json repository --jq '.repository.name'
```

**If service name not provided or unclear:**

Ask user:
```
Which service/repository is this PR for?
Examples: terminals, payments-core, api, etc.
```

**Ask for local repository path:**

```
What is the local path to the repository?
Example: /Users/username/Desktop/Repo/platform/terminals
```

**Verify access to local repo:**

```bash
# Check if path exists and is a git repo
cd <local-repo-path> && git status

# Verify we're on the correct branch or can check it out
git fetch origin <pr-branch-name>
git checkout <pr-branch-name>
git pull
```

**Check for repo-level documentation:**

```bash
# Look for CLAUDE.md or similar repo documentation
ls -la | grep -i "claude\|readme\|contributing"

# If CLAUDE.md exists, read it for:
# - Testing conventions
# - Code patterns
# - CI/CD specifics
# - Repository structure
cat CLAUDE.md 2>/dev/null || echo "No CLAUDE.md found"
```

**Why this matters:**
- AI comment implementation requires reading actual source code
- Test generation needs to follow repo-level patterns
- Coverage analysis requires access to test files
- Without local access, we can only provide feedback, not fixes

**If local access unavailable:**
- Inform user we can only analyze and suggest, not implement
- Recommend they provide repo access for full automation

### Step 1: Get PR Details

**Identify the PR:**

```bash
# Option 1: User provides PR number
PR_NUMBER=<user-provided>

# Option 2: Get PR for current branch
PR_NUMBER=$(gh pr view --json number --jq .number)
```

**Fetch PR info:**
```bash
gh pr view $PR_NUMBER --json number,title,headRefName,url
```

### Step 2: Check CI Status

**Get CI check status:**

```bash
gh pr checks $PR_NUMBER
```

**Analyze:**
- ✅ All passing → Inform user, no action needed
- ❌ Some failing → Proceed to Step 3
- ⏳ Running → Wait or check back later
- ⏸️ `quality-gate-utExpected` waiting/pending → Coverage check incomplete
- ❌ `rCoRe / comment_resolution_validator` failing → AI comments need addressing (Step 6d)

**Focus on required checks:** builds, unit tests, coverage, AI comment resolution (not optional linters)

### Step 3: Identify Failed Workflows and Tests

**List failed runs:**

```bash
gh run list --branch <branch-name> --status failure --limit 5
```

**Get latest failed run:**

```bash
RUN_ID=$(gh run list --branch <branch-name> --status failure --limit 1 --json databaseId --jq '.[0].databaseId')
```

**Download logs:**

```bash
gh run view $RUN_ID --log > ci_logs.txt
```

**Parse logs for failed tests:**

Patterns to look for:
- **Go**: `FAIL: TestName`, `--- FAIL: TestName`
- **Python**: `FAILED test_file.py::test_function`
- **JavaScript**: `FAIL src/test.js`
- **Java**: `Tests run: X, Failures: Y`

Extract test names and failure messages.

### Step 4: Run Failed Tests Locally

**For Go projects** (if `go.mod` exists):

```bash
# Run specific test
go test -v -run ^TestName$ ./path/to/package

# With race detector (important for concurrency)
go test -race -v -run ^TestName$ ./path/to/package

# All tests in package
go test -v ./path/to/package/...
```

**For Python projects:**

```bash
pytest tests/test_file.py::test_function -v
```

**For JavaScript/TypeScript:**

```bash
npm test -- --testNamePattern="test name"
```

**Capture results:** Note which tests pass/fail locally

### Step 5: Compare Results - Flaky vs Genuine

| CI | Local | Conclusion |
|----|-------|-----------|
| ❌ | ✅ | **Flaky** - Retrigger CI |
| ❌ | ❌ | **Genuine** - Fix code |
| ❌ | ❌ (different) | **Environment issue** |

### Step 6a: Handle Flaky Tests

**If tests pass locally but fail in CI:**

1. **Inform user:**
   ```
   🔄 Flaky test detected: TestName
   - Passes locally ✅
   - Fails in CI ❌
   ```

2. **Retrigger failed jobs:**
   ```bash
   gh run rerun $RUN_ID --failed
   ```

3. **Monitor:**
   ```bash
   gh run watch $RUN_ID
   ```

4. **If still failing:**
   - Investigate race conditions (use `-race` flag)
   - Check timing dependencies
   - Compare CI vs local environment

### Step 6b: Handle Genuine Failures

**If tests fail both locally and in CI:**

1. **Analyze failure:**
   - Read error from local run
   - Read error from CI logs
   - Identify root cause

2. **Common issues:**
   - Compilation errors → Fix syntax/types
   - Assertion failures → Fix logic
   - Missing deps → Update imports
   - Config issues → Check env vars

3. **For terminals/Go-specific:**
   - Transaction context (`ctx` vs `tctx`)
   - Race conditions
   - Mock setup
   - Test data

4. **Fix the code:**
   - Make necessary changes
   - Run tests locally to verify
   - Ensure `go fmt ./...` is run for Go projects

5. **Push fixes using pr-creator skill:**

   **IMPORTANT:** Use the pr-creator skill to push fixes properly.

   Tell user:
   ```
   ✅ Fix applied. Ready to push?
   ```

   After user confirms, invoke pr-creator workflow:
   - Show files changed
   - Stage fixes
   - Commit with message like "Fix failing tests in TestName"
   - Push to PR branch

   Alternatively, if pr-creator skill is available, say:
   ```
   "Let me push these fixes to your PR"
   ```
   And follow pr-creator workflow (Step 6-11 from that skill).

### Step 6c: Handle UT Coverage Failures

**If `quality-gate-utExpected` check fails or is waiting:**

This check monitors unit test coverage for new/changed code. When it fails, you need to add tests for uncovered lines.

1. **Identify the coverage check:**
   ```bash
   # Check for quality-gate-utExpected status
   gh pr checks $PR_NUMBER | grep -i "quality-gate\|coverage"
   ```

2. **Get failed run and download logs:**
   ```bash
   # Get the coverage workflow run
   RUN_ID=$(gh run list --branch <branch-name> --workflow "quality-gate-utExpected" --limit 1 --json databaseId --jq '.[0].databaseId')

   # Download logs
   gh run view $RUN_ID --log > coverage_logs.txt
   ```

3. **Parse coverage report for uncovered lines:**

   Look for patterns like:
   - **Go**: `coverage: X% of statements` with line-by-line coverage
   - **Python**: `TOTAL ... X%` with missing lines shown
   - **JavaScript**: `% Lines ... X/Y` with uncovered line numbers

   Common formats:
   ```
   file.go:45-52: not covered
   src/service.py: Lines 34, 67-72 not covered
   utils.js: Line 23 uncovered
   ```

4. **Analyze uncovered code:**

   For each file with uncovered lines:
   - Read the source file
   - Identify the uncovered functions/methods
   - Understand the logic and edge cases
   - Check existing test patterns in the repo

   ```bash
   # Read the source file with uncovered lines
   # Use Read tool to view file:line_number

   # Find existing test files for patterns
   find . -name "*test*" -type f | head -5
   ```

5. **Analyze repo-level test architecture (CRITICAL STEP):**

   **Before writing any tests, thoroughly analyze existing test patterns:**

   a. **Check CLAUDE.md for testing guidelines:**
   ```bash
   # Look for testing sections in CLAUDE.md
   grep -A 20 -i "test\|testing" CLAUDE.md
   ```

   b. **Identify test framework and conventions:**
   ```bash
   # For Go
   ls -la | grep _test.go
   grep -r "github.com/stretchr/testify" go.mod

   # For Python
   find . -name "test_*.py" -o -name "*_test.py" | head -5
   grep -r "pytest\|unittest" requirements*.txt

   # For JavaScript/TypeScript
   find . -name "*.test.js" -o -name "*.test.ts" | head -5
   cat package.json | grep -A 5 '"test"'
   ```

   c. **Study existing test patterns in the same module:**
   - Read 2-3 existing test files in the same directory/package
   - Identify patterns:
     - How are mocks created? (mockery, jest.mock, unittest.mock)
     - How is test data set up? (fixtures, factories, builders)
     - How are assertions written? (assert.Equal, expect, assertEqual)
     - How are tests organized? (table-driven, describe/it blocks, test classes)

   d. **Razorpay-specific patterns to check:**
   - **Transaction contexts:** Check if tests use `tctx` vs `ctx`
   - **Database mocking:** Look for `sqlmock`, `testcontainers`, or in-memory DBs
   - **Service dependencies:** Check how external services are mocked
   - **Configuration:** Look for test config files or env var patterns
   - **Common test utilities:** Check for helper functions in `testutil/` or similar

   ```bash
   # Find test utilities
   find . -type d -name "testutil" -o -name "testhelpers" -o -name "test_helpers"

   # Look for common test patterns
   grep -r "tctx\|transaction context" **/*_test.go
   grep -r "mock\|Mock" **/*_test* | head -10
   ```

   e. **Check for shared test fixtures or factories:**
   ```bash
   # Common fixture patterns
   grep -r "@pytest.fixture\|setupTest\|SetUp\|beforeEach" tests/
   ```

6. **Generate unit tests following repo patterns:**

   **Critical:** Match the exact patterns found in step 5:

   - **Naming:** Use same convention as existing tests
   - **Structure:** Follow same test organization (table-driven, subtests, etc.)
   - **Mocking:** Use same mocking library and patterns
   - **Assertions:** Use same assertion style
   - **Setup/Teardown:** Follow same lifecycle patterns
   - **Test data:** Use same approach (inline, fixtures, factories)

   **Razorpay/Go-specific best practices:**

   - Use table-driven tests for multiple scenarios:
   ```go
   func TestFeature(t *testing.T) {
       tests := []struct {
           name     string
           input    Input
           expected Output
           wantErr  bool
       }{
           {name: "happy path", input: validInput, expected: validOutput},
           {name: "edge case", input: edgeInput, expected: edgeOutput},
           {name: "error case", input: invalidInput, wantErr: true},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Test implementation
           })
       }
   }
   ```

   - Use `tctx` for transaction contexts in Razorpay services
   - Follow existing mock patterns (check if using testify/mock, mockery, etc.)
   - Include error scenarios and edge cases
   - Test both success and failure paths

   **Python best practices:**
   - Use pytest fixtures for common setup
   - Follow AAA pattern (Arrange, Act, Assert)
   - Use parametrize for multiple test cases
   - Mock external dependencies properly

   **JavaScript/TypeScript best practices:**
   - Use describe/it or test blocks consistently
   - Mock modules at the top of test file
   - Clear mocks between tests
   - Test async code properly (async/await, done callbacks)

7. **Write tests focusing on uncovered lines:**

   - Create or update test file for the source file
   - Write test cases that exercise uncovered code paths
   - Include:
     - Happy path scenarios
     - Edge cases
     - Error conditions
     - Boundary values

   Example structure:
   ```go
   // For Go
   func TestNewFeature_EdgeCase(t *testing.T) {
       // Arrange
       input := setupTestData()

       // Act
       result := NewFeature(input)

       // Assert
       assert.Equal(t, expected, result)
   }
   ```

8. **Run tests locally to verify:**

   **For Go:**
   ```bash
   # Run new tests
   go test -v -run ^TestNewFeature ./path/to/package

   # Check coverage locally
   go test -cover ./path/to/package
   go test -coverprofile=coverage.out ./path/to/package
   go tool cover -func=coverage.out
   ```

   **For Python:**
   ```bash
   # Run with coverage
   pytest tests/test_file.py --cov=src/module --cov-report=term-missing
   ```

   **For JavaScript:**
   ```bash
   # Run with coverage
   npm test -- --coverage
   ```

9. **Verify coverage improvement:**

   - Check that new tests cover previously uncovered lines
   - Ensure coverage percentage increases
   - Verify all tests pass

   ```
   Before: 75% coverage (lines 45-52 uncovered)
   After: 92% coverage (all lines covered)
   ```

10. **Push tests using pr-creator:**

   ```
   ✅ Tests added for uncovered lines. Ready to push?
   ```

   After confirmation:
   - Stage new/updated test files
   - Commit with message like "Add unit tests for <module> to improve coverage"
   - Push to PR branch
   - Monitor coverage check

### Step 6d: Handle AI PR Review Comments

**If `rCoRe / comment_resolution_validator` check fails:**

This check ensures AI PR reviewer comments are addressed. Review each comment and take appropriate action.

1. **Fetch AI PR review comments:**
   ```bash
   # Get all review comments on the PR
   gh pr view $PR_NUMBER --json reviews,comments,reviewThreads

   # Alternative: Use GitHub API for detailed comment threads
   gh api repos/{owner}/{repo}/pulls/$PR_NUMBER/comments
   ```

2. **Identify AI-generated comments:**

   Look for comments from bots or AI reviewers:
   - Comment author: bot accounts, AI reviewers
   - Comment patterns: structured feedback, code suggestions
   - Tags: `AI-PR-Review` label on PR

3. **Analyze each AI comment (CRITICAL EVALUATION):**

   **PREREQUISITE:** Must have local repo access from Step 0.5. Cannot implement suggestions without reading actual code.

   **For each comment, evaluate:**

   a. **Is the suggestion useful?**
      - Does it improve code quality?
      - Does it fix a real issue (bug, security, performance)?
      - Does it follow repo conventions?
      - Is it aligned with the PR's purpose?

   b. **Will it impact code behavior?**
      - Does it change logic or functionality?
      - Could it introduce bugs or break existing code?
      - Does it affect edge cases or error handling?
      - Is it just style/formatting vs functional change?

   c. **Context check (use local repo from Step 0.5):**
      - Read the actual source file being commented on using Read tool
      - Understand the surrounding context (read 20-30 lines around the change)
      - Check CLAUDE.md for repo-specific patterns
      - Verify against existing patterns in the codebase:
        ```bash
        # Navigate to local repo
        cd <local-repo-path>

        # Read the file with context
        # Use Read tool with file path and line numbers from comment

        # Check for similar patterns in the codebase
        grep -r "similar_pattern" .
        ```
      - Ensure suggestion aligns with repo architecture and conventions

4. **Decision matrix for AI comments:**

   | Condition | Action | Emoji | Comment | Resolve |
   |-----------|--------|-------|---------|---------|
   | **Useful & Safe** | Implement change | 👍 | "Implemented: [describe action]" | ✅ Yes |
   | **Not Useful** | Reject | 👎 | "Not applicable because [reason]" | ✅ Yes |
   | **Doubtful/Unclear** | Tag author | ❓ | "@author Please review this suggestion" | ❌ No |

5. **Process useful comments (👍 Implement):**

   **When a comment is useful and won't negatively impact behavior:**

   a. **Read the code file (CRITICAL - use local repo path from Step 0.5):**
   ```bash
   # Navigate to local repo
   cd <local-repo-path-from-step-0.5>

   # Use Read tool to view the file and specific lines mentioned in the AI comment
   # Example: Read tool with path: /Users/user/Repo/platform/terminals/internal/controllers/v1/api_test.go
   # Include context (20-30 lines around the commented line)
   ```

   b. **Implement the suggestion:**
   - Navigate to local repo: `cd <local-repo-path>`
   - Make the code change as suggested using Edit tool
   - Ensure it follows repo patterns (check CLAUDE.md)
   - Test locally if needed (run relevant tests)
   - Run formatters: `go fmt ./...` or `npm run format` etc.

   c. **Add reaction and reply IN-THREAD (CRITICAL):**
   ```bash
   # Add thumbs up reaction to the AI comment
   gh api -X POST /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions \
     -f content='+1'

   # Reply IN-THREAD (not as new PR comment!) using in_reply_to parameter
   gh api -X POST /repos/{owner}/{repo}/pulls/3830/comments \
     -f body="👍 Implemented this suggestion.

   **Next action:** [Describe what you did]
   - Changed X to Y
   - Updated Z for better clarity

   Co-authored-by: Claude Sonnet 4.5 <noreply@anthropic.com>" \
     -F in_reply_to={comment_id}
   ```

   **IMPORTANT:** Must use `in_reply_to` parameter to reply in the same thread, not create a new PR comment!

   d. **Mark conversation as resolved:**
   ```bash
   # Resolve the comment thread (using GraphQL)
   gh api graphql -f query='
     mutation {
       resolveReviewThread(input: {threadId: "'"$THREAD_ID"'"}) {
         thread {
           isResolved
         }
       }
     }'
   ```

   **Note:** To get thread IDs, use:
   ```bash
   gh api /repos/{owner}/{repo}/pulls/$PR_NUMBER/comments --jq '.[] | {id, body, path, line}'
   ```

6. **Process not useful comments (👎 Reject):**

   **When a comment is not applicable or incorrect:**

   a. **Analyze why it's not useful:**
   - Doesn't apply to this context
   - Would introduce bugs or break functionality
   - Conflicts with repo conventions
   - Out of scope for this PR

   b. **Add reaction and reply IN-THREAD:**
   ```bash
   # Add thumbs down reaction
   gh api -X POST /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions \
     -f content='-1'

   # Reply IN-THREAD with reasoning
   gh api -X POST /repos/{owner}/{repo}/pulls/$PR_NUMBER/comments \
     -f body="👎 Not implementing this suggestion.

   **Reason:** [Explain clearly why]
   - This would break X functionality
   - Not aligned with repo pattern Y
   - Out of scope for this PR (focus is Z)

   Co-authored-by: Claude Sonnet 4.5 <noreply@anthropic.com>" \
     -F in_reply_to={comment_id}
   ```

   c. **Mark conversation as resolved:**
   ```bash
   # Same as above - resolve the thread
   ```

7. **Process doubtful comments (❓ Tag author):**

   **When you're uncertain about a suggestion:**

   a. **Add reaction and tag PR author IN-THREAD:**
   ```bash
   # Add confused/eyes reaction
   gh api -X POST /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions \
     -f content='eyes'

   # Get PR author
   PR_AUTHOR=$(gh pr view $PR_NUMBER --json author --jq '.author.login')

   # Tag author IN-THREAD for input
   gh api -X POST /repos/{owner}/{repo}/pulls/$PR_NUMBER/comments \
     -f body="@$PR_AUTHOR Please review this AI suggestion.

   **My analysis:** [Your thoughts]
   - Potential benefit: X
   - Potential concern: Y

   Could you confirm the right approach here?

   Co-authored-by: Claude Sonnet 4.5 <noreply@anthropic.com>" \
     -F in_reply_to={comment_id}
   ```

   b. **DO NOT resolve** - leave for author to decide

8. **Batch changes together (use local repo from Step 0.5):**

   After processing all useful AI comments:
   ```bash
   # Navigate to local repo
   cd <local-repo-path>

   # Check what changed
   git status

   # Run tests locally to verify changes
   # For Go: go test -v ./...
   # For Python: pytest
   # For JavaScript: npm test

   # Run formatters
   # For Go: go fmt ./...
   # For JavaScript: npm run format
   ```

   - Group all implemented changes
   - Verify tests pass locally
   - Commit with clear message describing AI comment fixes
   - Push to PR branch using pr-creator workflow

9. **Add AI-PR-Review label:**

   ```bash
   # After pushing changes, ensure label is added
   gh pr edit $PR_NUMBER --add-label "AI-PR-Review"
   ```

10. **Summary report:**

    ```
    🤖 AI PR Review Comments Processed

    - 👍 Implemented: X comments
    - 👎 Rejected: Y comments
    - ❓ Flagged for author: Z comments
    - ✅ Resolved threads: N

    Changes pushed: [commit hash]
    Label added: AI-PR-Review
    ```

**Best practices for AI comment evaluation:**

- **Read the code context** - Always view the actual code before deciding
- **Test if uncertain** - Run the code/tests locally to verify impact
- **Be conservative** - If doubtful, tag the author rather than making risky changes
- **Document reasoning** - Always explain why you accepted/rejected a suggestion
- **Batch related changes** - Group similar fixes in one commit
- **Respect PR scope** - Don't implement suggestions outside the PR's purpose
- **Check for side effects** - Consider impact on other parts of codebase

**Common AI comment patterns to watch for:**

✅ **Usually useful:**
- Removing unused variables
- Adding error handling for unchecked errors
- Fixing typos in comments/strings
- Improving log messages for clarity
- Adding null/nil checks for safety

⚠️ **Evaluate carefully:**
- Changing variable names (may break references)
- Refactoring logic (may introduce bugs)
- Adding new dependencies
- Changing function signatures
- Performance optimizations (verify impact)

❌ **Usually reject:**
- Suggestions outside PR scope
- Style changes conflicting with repo patterns
- Overly complex refactoring
- Changes to unrelated code
- Breaking changes without clear benefit

### Step 7: Monitor CI After Fix/Retrigger

**Wait for CI to run:**

```bash
# Wait a bit
sleep 15

# Check status
gh pr checks $PR_NUMBER
```

**Report to user:**

```
✅ CI Status: All checks passing
🔗 PR: <PR URL>

Summary:
- Flaky tests retriggered: X
- Genuine failures fixed: Y
- Coverage tests added: Z
- AI comments implemented: A
- AI comments rejected: B
- AI comments flagged for review: C
- Total fixes pushed: N
```

**If still failing:**
- Report remaining failures
- Ask user if they want to investigate further
- Repeat workflow for new failures

## Error Handling

| Issue | Solution |
|-------|----------|
| gh CLI not installed | `brew install gh` |
| gh not authenticated | `gh auth login` |
| PR not found | Verify PR number/branch |
| Service name unknown | Ask user for service/repo name and local path |
| Local repo not accessible | Request local path or inform can only analyze, not implement |
| Can't download logs | Check permissions, `gh auth refresh` |
| Can't parse logs | Ask user for failed test names |
| Can't rerun workflow | Check write permissions |
| Tests still fail after fix | Re-analyze, might be different issue |
| Can't find coverage report | Check workflow logs, may need different parsing |
| Coverage still low after adding tests | Verify tests actually execute uncovered lines |
| Can't fetch PR comments | Check `gh api` permissions, try `gh auth refresh` |
| Can't add reactions | Requires write access, verify token scopes |
| Can't resolve threads | Need GraphQL API access, verify permissions |
| Can't add labels | Requires triage permission on repo |

## Best Practices

1. **Focus on required checks** - Ignore optional checks
2. **Run with same config as CI** - Same Go/Node/Python version
3. **Use race detector for Go** - `go test -race`
4. **Verify fix locally first** - Always test before pushing
5. **Limit retrigger attempts** - Max 2, then investigate
6. **Use pr-creator for fixes** - Proper workflow for commits/push
7. **Check repo context** - CLAUDE.md, skills for patterns
8. **Analyze test patterns before writing** - Always review existing tests
9. **Match repo conventions** - Use same testing frameworks, patterns, mocks
10. **Test quality over quantity** - Write meaningful tests, not just for coverage
11. **Include edge cases** - Test error paths, boundaries, race conditions
12. **Read code before implementing AI suggestions** - Always understand context
13. **Document AI comment decisions** - Explain why you accepted/rejected
14. **Be conservative with AI suggestions** - When doubtful, tag author
15. **Batch AI comment fixes** - Group related changes in one commit

## Integration with pr-creator

**Seamless workflow:**

1. User creates PR with `/pr-creator`
2. CI fails or AI review comments need addressing
3. User: "Fix CI and handle AI comments" → `/pr-autopilot`
4. Skill detects failures, runs tests, processes AI comments
5. If genuine failure → Fix code
6. If useful AI suggestions → Implement them
7. Use pr-creator to push fixes to PR
8. CI passes → PR ready to merge

**Invoking pr-creator from pr-autopilot:**

When pushing fixes, follow pr-creator workflow:
- Navigate to local repo (from Step 0.5)
- Run formatters (`go fmt`, etc.)
- Stage files
- Commit with descriptive message
- Get push confirmation from user
- Push to PR branch
- Verify CI status

## Usage Examples

**Basic usage:**
```
"/pr-autopilot"
"Fix CI failures on this PR"
"Handle AI review comments for terminals PR #3830"
"Check why tests are failing on PR #123"
"Get my PR merge-ready"
```

**With service specification:**
```
"/pr-autopilot --repo razorpay/terminals --pr 3830"
"Fix CI for terminals PR 3830 (local path: ~/Repo/platform/terminals)"
"Process AI comments for payments-core PR 456"
```

**Specific actions:**
```
"Retrigger flaky tests"
"The build failed, can you fix it?"
"Tests pass locally but fail in CI"
"Add tests for coverage failures"
"Coverage check is failing, can you fix it?"
"quality-gate-utExpected is failing"
"Improve unit test coverage"
"Review and address AI PR comments"
"Implement useful AI comments and add AI-PR-Review label"
```

## Flaky Test Patterns

Common indicators:
- Time-dependent tests (timeouts, time.Sleep)
- Concurrent operations (goroutines, channels)
- External services (HTTP, database)
- Random data
- File system ops

## Coverage Failure Patterns

**Common uncovered code patterns:**

1. **Error handling paths**
   - Error returns not tested
   - Panic recovery not covered
   - Validation failures not tested

2. **Edge cases**
   - Nil/empty input handling
   - Boundary conditions
   - Default/fallback values

3. **Conditional branches**
   - If/else paths not fully covered
   - Switch cases missing tests
   - Early returns not exercised

4. **Configuration variations**
   - Different config values not tested
   - Environment-specific code paths
   - Feature flags not toggled in tests

**Coverage improvement strategies:**

- Focus on new/changed code first
- Test error paths explicitly
- Use table-driven tests for multiple scenarios
- Mock external dependencies properly
- Test both positive and negative cases
- Include boundary value testing

## Repo-Specific Notes

**Terminals (Go):**
- Common: race conditions, transaction context
- Test: `go test -race -v ./...`

**Python services:**
- Test: `pytest -v`

**Node.js services:**
- Test: `npm test`

## Razorpay Test Architecture Patterns

**Before writing tests for Razorpay repos, analyze:**

1. **Testing frameworks in use:**
   - Go: testify/assert, testify/mock, sqlmock
   - Python: pytest, unittest, mock
   - Check existing test files for patterns

2. **Common Razorpay patterns:**
   - **Transaction contexts:** Use `tctx` for transactional operations
   - **Database testing:** Check for sqlmock, testcontainers, or test DBs
   - **Service mocks:** Look for mock generators or manual mocks
   - **Test utilities:** Check for shared test helpers

3. **Test organization:**
   - Unit tests: `*_test.go`, `test_*.py`, `*.test.js`
   - Integration tests: Separate directories or tags
   - Test data: fixtures/ or testdata/ directories

4. **Key checks before writing tests:**
   ```bash
   # Check CLAUDE.md for testing guidelines
   cat CLAUDE.md | grep -A 10 -i "test"

   # Find existing test utilities
   find . -name "testutil*" -o -name "*test_helper*"

   # Check mock patterns
   grep -r "mock\|Mock" **/*_test.go | head -5

   # Look for table-driven test examples
   grep -A 10 "tests := \[\]struct" **/*_test.go | head -20

   # Check for transaction context usage
   grep -r "tctx" **/*_test.go | head -5
   ```

5. **Razorpay Go test template:**
   ```go
   func TestFeatureName(t *testing.T) {
       tests := []struct {
           name    string
           setup   func() // Setup mocks, data
           input   Input
           want    Output
           wantErr bool
       }{
           {
               name: "successful case",
               setup: func() { /* setup mocks */ },
               input: validInput,
               want: expectedOutput,
           },
           {
               name: "error case",
               setup: func() { /* setup error mocks */ },
               input: invalidInput,
               wantErr: true,
           },
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               if tt.setup != nil {
                   tt.setup()
               }

               got, err := FeatureName(tt.input)

               if tt.wantErr {
                   assert.Error(t, err)
                   return
               }

               assert.NoError(t, err)
               assert.Equal(t, tt.want, got)
           })
       }
   }
   ```

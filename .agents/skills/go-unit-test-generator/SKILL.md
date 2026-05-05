---
name: go-unit-test-generator
description: >-
  Generates unit tests for Go code changes on a feature branch. Analyzes git diff, creates
  table-driven tests with GoMock/testify, updates affected existing tests, runs locally until
  >95% coverage passes (max 5 iterations), commits, pushes, and monitors CI. Use when user
  wants to generate unit tests for branch changes.
---

# Go Unit Test Generator

Autonomous end-to-end unit test lifecycle: analyze changes, generate tests, run locally, push, monitor CI.

## Activation

```
USING go-unit-test-generator SKILL — [action description]
```

## Invocation

```
/go-unit-test-generator
```

## Input Requirements

- **Go repository** on a feature branch (not master/main)
- **Changed Go files** in the current branch vs base branch

## Execution Order (MANDATORY)

```
Phase 0: Validate Context        → Check branch, tools, load project patterns
Phase 1: Change Analysis         → Parse diff, identify new/modified/deleted symbols
Phase 2: Existing Test Updater   → Fix broken existing tests surgically
Phase 3: New Test Writer         → Generate production-grade _test.go files
Phase 4: Local Test Loop         → Run tests + enforce >95% coverage (max 5 iterations)
Phase 5: Commit & Push           → Stage test files, commit, push (with user confirmation)
Phase 6: CI Handoff              → Print CI run URL, save metadata, EXIT (no polling)
```

---

## Phase 0: Validate Context

1. Confirm current branch is NOT master/main:
   ```bash
   BRANCH=$(git branch --show-current)
   if [ "$BRANCH" = "master" ] || [ "$BRANCH" = "main" ]; then echo "ABORT: on protected branch"; fi
   ```

2. Confirm tools are available: `go version`, `which mockgen`, `gh --version`

3. Load repo skill from `.agents/skills/` if present — read for architecture context

4. **Discover test file patterns using Claude AI** (not bash scripts):
   - Use Glob to find all `*_test.go` files in the repository
   - For each changed source file (e.g., `foo.go`), Claude intelligently identifies the corresponding test file in the **same folder**
   - Read up to 3-5 existing `*_test.go` files to learn:
     - Mock strategy (GoMock vs testify/mock vs manual)
     - Error package (`errorclass`, `errors`, custom)
     - Import conventions and test helpers
     - Setup/teardown patterns (`TestMain`, `setupTest`)
   
   **Why Claude, not bash scripts?**
   - Understands Go naming conventions (handles unconventional test file names)
   - Works across any directory structure (`pkg/`, `internal/`, `cmd/`, `service/`, custom)
   - Infers test file location from imports and code patterns
   - More robust than glob/sed pattern matching

5. Identify base branch: `git merge-base HEAD origin/master` or `origin/main`

**Output:** 
- ✅ Test files discovered for each changed source file
- ✅ Project test conventions summary (mock strategy, error package, helpers, patterns)
- ✅ Ready for Phase 1 change analysis

---

## Phase 1: Change Analysis

See [subskills/change-analysis.md](subskills/change-analysis.md) for detailed steps.

Claude reads each changed Go file directly and uses Go language understanding to produce a structured list of:
- New functions/methods (need new tests)
- Modified functions/methods (need updated tests)
- Deleted functions/methods (their tests must be removed/updated)

For each symbol, check if an existing `*_test.go` covers it.

**CRITICAL: Capture the `test_files` mapping from Phase 1 output and pass it to Phase 3.**
- Phase 1 outputs: `"test_files": {"internal/core/order.go": {"test_file": "internal/core/order_test.go", "exists": true/false}}`
- This mapping tells Phase 3 whether each test file exists
- **MUST be used by Phase 3** to decide: append (exists=true) vs create (exists=false)

---

## Phase 2: Existing Test Updater

See [subskills/existing-test-updater.md](subskills/existing-test-updater.md) for detailed steps.

For each modified/deleted symbol with an existing test:
- Read the test, determine if it still compiles and is semantically valid
- Fix: update function signatures, mock expectations, assertion values
- Delete: remove test cases for deleted code paths
- **Do NOT rewrite tests wholesale** — surgical edits only

---

## Phase 3: New Test Writer

**CRITICAL: Read the `test_files` mapping from Phase 1 output BEFORE writing tests.**

For EACH changed source file:
1. **Look up in test_files mapping:** Is there an existing `*_test.go`?
   - `"internal/core/order.go": {"test_file": "internal/core/order_test.go", "exists": true}` → **APPEND to existing file**
   - `"internal/core/order.go": {"test_file": "internal/core/order_test.go", "exists": false}` → **CREATE new file**
2. **Append mode (exists=true):** Read entire existing test file, find last test function, append new tests after it. **Preserve all existing code.**
3. **Create mode (exists=false):** Create new `order_test.go` file with package declaration, imports, and new test functions.
4. **Never create files with names like `order_test_new.go`** — always use the exact filename from the mapping.
5. **Idempotency check:** Before appending `func TestFoo`, run `grep -q "func TestFoo(" <test_file>`. If found, skip — the test already exists. This prevents duplicate test functions when the skill is re-run on the same branch.

See [subskills/test-writer.md](subskills/test-writer.md) for full specification.

Generate `*_test.go` files using production-grade patterns from [references/production-go-patterns.md](references/production-go-patterns.md):
- Table-driven tests with `setupMock` field
- GoMock or project's existing mock strategy
- `require` for fatal, `assert` for soft assertions
- Happy path + all error paths + nil/empty/boundary values
- Transaction context rule (CRITICAL): use `tctx` inside `db.Transaction`

---

## Phase 4: Local Test Execution Loop

See [subskills/local-runner.md](subskills/local-runner.md) for detailed steps.

**Step 1.5: AI-Powered Build Error Resolution**

Before running tests, check if code compiles. If build errors occur, use Claude's reasoning to solve them:

1. **Parse build error** to extract file, line, and missing identifier
2. **Read problematic code** + related source code
3. **Invoke Claude's AI reasoning** to understand root cause:
   - Type renamed or moved to different package?
   - Missing import?
   - Function signature changed?
   - Mock outdated or incomplete?
   - Test code using old API?
4. **Claude suggests & applies fix** intelligently
5. **Re-run build** to verify
6. **Retry up to 5 times** — if Claude still can't solve it, escalate with analysis

This is NOT pattern-matching — Claude reasons about the actual problem.

**Steps 1-4: Test Execution Loop**

Run `scripts/run-coverage.sh` — executes `go test` + parses coverage.

**Pass condition:** All tests green AND overall coverage >= 95%.

After tests pass and coverage meets the threshold, run a **test quality check** (see Step 3.5 in local-runner.md):
- Assertion density >= 2 meaningful assertions per test case
- No shallow-only tests (tests that only assert `NotNil`/`NoError` without checking actual values)
- Error path coverage: every distinct `return err` in source has a corresponding `wantErr` test case
- Quality check is **advisory** — it flags issues and attempts auto-fix but does not block Phase 5.

**Retry logic (max 5 iterations):**
1. Parse error output → identify failing test + root cause
2. Fix the specific failing test (max 2 fix attempts per test)
3. If unfixable after 2 attempts → `t.Skip` with detailed comment
4. Re-run entire suite
5. After 5 iterations without passing → surface failures to user, ask for guidance

**NEVER push if local tests fail.**

---

## Phase 5: Commit & Push

**MANDATORY GATE: Phase 4 MUST be completed successfully first!**

Before entering Phase 5, verify:
- ✅ All tests executed locally (Phase 4 Step 1 ran)
- ✅ All test packages report `ok` status (no `FAIL`)
- ✅ Overall coverage >= 95%
- ✅ No syntax errors in test code
- ✅ Build succeeded (Phase 4 Step 1.5 passed)

**If any of the above is NOT met: ABORT Phase 5 and report the issue to the user. DO NOT COMMIT OR PUSH.**

See [subskills/git-push.md](subskills/git-push.md) for detailed steps.

1. Stage only `*_test.go` and `mocks/*.go` files
2. Commit: `test: add/update unit tests for <package> (coverage: X%)`
3. **Show diff and ask user for confirmation before pushing**
4. Push to current branch

---

## Phase 6: CI Handoff (Fire and Forget)

**Do NOT poll CI. Do NOT sleep. Identify the run, save metadata, and EXIT.**

See [subskills/ci-monitor.md](subskills/ci-monitor.md) for detailed steps.

### Why No Polling

Razorpay CI pipelines take 60-70 minutes. Polling from inside a Claude session:
- Burns tokens doing nothing (~65 poll cycles for a single CI run)
- Is vulnerable to session disconnect, context compaction, and terminal closure
- Blocks the developer from using Claude for other work
- The previous `check-ci-status.sh` script had a 30-minute hard timeout — shorter than any real CI run

### Steps

1. **Verify GitHub authentication** (same pre-flight check as before — `$GH_TOKEN` or `gh auth status`)

2. **List ALL CI runs** for this commit (unit tests, SLIT, lint, security scan, etc.):
   ```bash
   sleep 10
   BRANCH=$(git branch --show-current)
   COMMIT_SHA=$(git rev-parse HEAD)
   ALL_RUNS=$(gh run list --branch "$BRANCH" --limit 10 \
     --json databaseId,url,headSha,status,name,workflowName \
     --jq "[.[] | select(.headSha | startswith(\"${COMMIT_SHA:0:7}\"))]")
   ```
   Present all runs as a table. Ask user which is the unit test run (auto-suggest if workflow name contains "unit"/"test"/"ut"). If only 1 run, use it automatically.

3. **Persist ALL run metadata** for the companion CI fix skill:
   ```bash
   cat > /tmp/gutg-ci-run.json <<EOF
   {
     "branch": "$BRANCH",
     "commit_sha": "$COMMIT_SHA",
     "push_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
     "primary_run_id": "$PRIMARY_RUN_ID",
     "runs": $ALL_RUNS_JSON,
     "test_files": $(git diff --name-only HEAD~1 HEAD | grep '_test.go' | jq -R -s 'split("\n") | map(select(. != ""))')
   }
   EOF
   ```

4. **Print summary showing ALL runs and EXIT:**
   ```
   ## Push Complete — CI Running

   **Branch:** <branch>
   **Commit:** <short_sha>

   | Workflow              | Run ID | Status      | URL   |
   |-----------------------|--------|-------------|-------|
   | Unit Tests (primary)  | 12345  | in_progress | <url> |
   | SLIT Integration      | 12346  | in_progress | <url> |
   | Lint & Build          | 12347  | in_progress | <url> |

   CI typically takes 60-70 minutes for Razorpay pipelines.

   **Next steps:**
   - If CI passes: you are done!
   - If CI fails: run `/go-unit-test-generator-ci-fix` to diagnose and fix

   Run metadata saved to /tmp/gutg-ci-run.json
   ```

5. **EXIT. The skill is complete.** Do not poll. Do not wait.

---

## Behavioral Rules

### Local-First Rule
Never push without all local tests passing at >= 95% coverage. Local validation prevents wasting CI runs.

### Surgical Edit Rule
Never rewrite a working existing test. Only fix what the branch changes broke. Preserve existing test structure, naming, and style.

### Confirmation Gate
Always show the staged diff and ask for user confirmation before `git push`. No silent pushes.

### No Blind Retries
Each retry iteration MUST act on new diagnostic signal from the test output. If the same fix is attempted twice without new evidence, STOP and escalate.

### Skip with Evidence
If a test cannot be fixed in 2 attempts:
```go
t.Skip("Could not fix after 2 attempts: <root cause explanation>")
```
Include a comment block explaining what was tried and why it failed.

### Escalation Limits
- **Local:** After 5 iterations without all-pass → escalate to user with diagnosis
- **CI:** Decoupled — use `/go-unit-test-generator-ci-fix` if CI fails (max 3 invocations per branch)

### Memory Optimization
Do not re-analyze the same test failure twice. If repetition is detected, switch strategy or escalate.

---

## Decision Rules

**Ask user when:**
- Coverage target seems unreachable for the changed code
- Existing test uses patterns not recognized
- Push confirmation required (always)
- 5 local iterations or 3 CI failures exhausted

**Act autonomously when:**
- Reading existing tests and conventions
- Generating new test files
- Running `go test` locally
- Parsing coverage output
- Fixing failing tests within retry limits

---

## State Tracking

Maintain a running state indicator throughout the workflow:

```
STATE: [Validating | Analyzing | Updating Tests | Writing Tests | Running Local | Pushing | Monitoring CI]
ITERATION: X/5 (local) | Y/3 (CI)
COVERAGE: X%
```

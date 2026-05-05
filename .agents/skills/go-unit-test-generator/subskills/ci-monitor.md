# Phase 6: CI Handoff

After Phase 5 pushes successfully, identify the CI run, persist metadata, and exit. **No polling. No sleeping. No waiting.**

## Pre-Flight: GitHub Authentication Check

Before querying CI runs, verify GitHub CLI authentication is configured. This check runs **ONCE**.

**Check in order:**

1. **Check environment variable first:**
   ```bash
   if [ -n "$GH_TOKEN" ]; then
       # Token found in environment, proceed
   else
       # Token not set, check stored credentials
       gh auth status
   fi
   ```

2. **If both fail (no GH_TOKEN and no stored credentials):**
   - **Problem:** Not authenticated to GitHub
   - **Resolution — provide options to user:**
     1. Set token: `export GH_TOKEN="your_personal_access_token"` (needs `repo` + `read:workflow` scopes)
     2. Run interactive login: `gh auth login`
   - **Action:** Do NOT proceed. Ask user to authenticate first, then restart Phase 6.

---

## Step 1: List ALL CI Runs for This Commit

After push, multiple workflows may trigger (unit tests, SLIT, lint, security scan, etc.). List **all of them** so the user knows which to monitor.

```bash
sleep 10
BRANCH=$(git branch --show-current)
COMMIT_SHA=$(git rev-parse HEAD)

# List ALL runs for this commit — not just one
ALL_RUNS=$(gh run list --branch "$BRANCH" --limit 10 \
  --json databaseId,url,headSha,status,name,workflowName,event \
  --jq "[.[] | select(.headSha | startswith(\"${COMMIT_SHA:0:7}\"))]")
```

**Present ALL runs to the user as a numbered list:**

```
## CI Runs Triggered for commit <short_sha>

| #  | Workflow                | Run ID    | Status      | URL                    |
|----|------------------------|-----------|-------------|------------------------|
| 1  | Unit Tests             | 12345     | in_progress | https://github.com/... |
| 2  | SLIT Integration Tests | 12346     | in_progress | https://github.com/... |
| 3  | Lint & Build           | 12347     | in_progress | https://github.com/... |
| 4  | Security Scan          | 12348     | queued      | https://github.com/... |

Which run(s) should I track for unit test results?
(Enter number, or "all" to track all runs)
```

**If only 1 run found:** Use it automatically, no need to ask.
**If no runs found within 30 seconds:**
1. Check if CI triggers are configured: `ls .github/workflows/*.yml`
2. Check for merge conflicts: `gh pr view --json mergeable`
3. Report to user: "CI not triggered — check workflow configuration"

**Auto-detection hint:** If a workflow name contains "unit", "test", or "ut" (case-insensitive), suggest it as the primary run. But always show all runs and let the user confirm.

## Step 2: Persist Run Metadata

Save **all runs** for the companion `/go-unit-test-generator-ci-fix` skill. The user-selected primary run is marked with `"primary": true`.

```bash
cat > /tmp/gutg-ci-run.json <<EOF
{
  "branch": "$BRANCH",
  "commit_sha": "$COMMIT_SHA",
  "push_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "primary_run_id": "$PRIMARY_RUN_ID",
  "runs": [
    {"run_id": "12345", "workflow": "Unit Tests", "url": "...", "primary": true},
    {"run_id": "12346", "workflow": "SLIT Integration Tests", "url": "...", "primary": false},
    {"run_id": "12347", "workflow": "Lint & Build", "url": "...", "primary": false}
  ],
  "test_files": $(git diff --name-only HEAD~1 HEAD | grep '_test.go' | jq -R -s 'split("\n") | map(select(. != ""))')
}
EOF
```

This way the ci-fix skill can:
- Default to the primary (unit test) run
- Let the user switch to any other run if needed

## Step 3: Report and Exit

Present the final summary showing ALL runs:

```
## Push Complete — CI Running

**Branch:** <branch>
**Commit:** <short_sha>

### CI Runs
| Workflow                | Run ID | Status      | URL           |
|------------------------|--------|-------------|---------------|
| Unit Tests (primary)   | 12345  | in_progress | <url>         |
| SLIT Integration Tests | 12346  | in_progress | <url>         |
| Lint & Build           | 12347  | in_progress | <url>         |

CI typically takes 60-70 minutes for Razorpay pipelines.

**Next steps:**
- If CI passes: you are done!
- If any run fails: run `/go-unit-test-generator-ci-fix` to diagnose and fix
  - It defaults to the primary (Unit Tests) run
  - Pass a specific run ID to fix a different workflow: `/go-unit-test-generator-ci-fix <run_id>`

Run metadata saved to /tmp/gutg-ci-run.json
```

**EXIT immediately. Do NOT poll. Do NOT sleep. The skill is complete.**

---

## Why No Polling

The previous design polled CI from inside the Claude session with `sleep 300` (5 min intervals) for up to 90 minutes per attempt, 3 attempts max = 4.5 hours potential agent time.

Problems with that approach:
- **Script timeout conflict:** `check-ci-status.sh` hard-exits at 30 min but CI takes 60-70 min
- **Session vulnerability:** Claude sessions can disconnect (laptop close, SSH drop, terminal restart)
- **Context compaction:** After hours of polling, earlier phase context (conventions, test_files mapping) gets evicted
- **Token waste:** ~65 poll cycles × token cost per cycle for zero useful work
- **Developer blocked:** Cannot use Claude for other work while agent is polling

The companion skill `/go-unit-test-generator-ci-fix` solves all of these: fresh session, fresh context, focused on the specific CI failure, and the developer chooses when to invoke it.

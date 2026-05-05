---
name: pr-action-analyzer
description: >-
  Analyze GitHub Actions CI check failures (Unit Tests, SLIT, or E2E/ETE) for a Pull Request.
  Use when a PR has a failing CI check and you need to investigate why — accepts a PR URL and
  action type (UT / SLIT / E2E), checks the check status, and for E2E failures: locates the
  Argo workflow URL from PR comments, downloads Argo log artifacts as text files, lists every
  failing E2E test, and identifies common failure patterns with root causes. Use when asked
  "why is UT/SLIT/E2E failing on PR X", "analyze E2E failures", "check CI status for this PR",
  "debug the failing action on this PR", or "what ETEs are failing in this PR".
argument-hint: "<PR_URL> [UT|SLIT|E2E]"
allowed-tools: Bash, Read, Write, AskUserQuestion
---

# PR Action Analyzer

Diagnose CI check failures on a GitHub PR. Supports three action types: **UT** (Unit Tests), **SLIT** (Service-Layer Integration Tests), and **E2E** (End-to-End / ETE tests). For E2E failures, performs deep Argo workflow log analysis.

## Activation Announcement

First output must be:
```
USING PR-ACTION-ANALYZER SKILL — Analyzing {UT|SLIT|E2E} for PR #{number}
```
For Mode B (direct Argo URL), use:
```
USING PR-ACTION-ANALYZER SKILL — Analyzing Argo workflow {workflow-name} directly
```

## Input Requirements

Two valid input modes:

**Mode A — PR URL** (full flow): provide a GitHub PR URL and action type. The skill fetches CI checks, locates the Argo URL from PR comments, then analyzes logs.
- PR URL: `https://github.com/razorpay/api/pull/1234`
- Action type: `UT`, `SLIT`, or `E2E` (ask if not provided)

**Mode B — Argo URL directly** (Argo-only flow): provide an Argo workflow URL and skip GitHub entirely. Jump straight to Step 6.
- Argo URL: `https://argo.dev.razorpay.in/workflows/argo-workflows/e2e-xxxxx`

Detect mode from the input: if the input contains `argo.dev.razorpay.in`, use Mode B and **skip Steps 1–5 entirely**.

## Hard Constraints

**Read-only.** Never post comments, approve, merge, close, or write to any PR. All GitHub API usage must be GET requests only.

**GitHub CLI required for Mode A only.** All GitHub data (PR checks, comments, run logs) MUST be fetched via `gh` CLI or `gh api`. Never use WebFetch, unauthenticated `curl`, or direct `api.github.com` URLs — private repos return a 404 HTML body that breaks parsing.

**Only check `gh` when using Mode A (PR URL input).** If the user provided an Argo URL directly (Mode B), skip this check entirely and go to Step 6.

For Mode A, before Step 1, verify `gh` is available and authenticated:
```bash
command -v gh >/dev/null 2>&1 || echo "GH_NOT_FOUND"
gh auth status 2>&1 || echo "GH_NOT_AUTHED"
```
- If `GH_NOT_FOUND`: stop and tell the user "`gh` CLI is not installed. Install it from https://cli.github.com then run `gh auth login`."
- If `GH_NOT_AUTHED` (gh exists but not logged in): stop and tell the user "GitHub CLI is not authenticated. Run `gh auth login` then re-run this skill."
- Otherwise: proceed to Step 1.

---

## Step 1 — Parse PR and Fetch All CI Checks

Extract `{org}`, `{repo}`, and `{pr_number}` from the PR URL.

```bash
# Fetch all check runs for the PR
gh pr checks <PR_URL> --json name,status,conclusion,startedAt,completedAt,link
```

Print a table of all checks found:

| Check Name | Status | Conclusion | Link |
|-----------|--------|------------|------|
| ...       | ...    | ...        | ...  |

---

## Step 2 — Locate the Target Action

Match the user's action type to a check using case-insensitive substring matching:

| User Input | Match Substrings |
|------------|-----------------|
| `UT`   | `unit`, `unittest`, `unit-test`, `ci-test`, `ci/test` |
| `SLIT` | `slit`, `service-layer`, `integration` |
| `E2E`  | `e2e`, `ete`, `end-to-end`, `end_to_end` |

- If **multiple checks match**: present them and ask the user to pick one.
- If **no check matches**: display all available check names and ask which to analyze.

---

## Step 3 — Report Action Status

| Conclusion | Action |
|------------|--------|
| `success` | Print: "✓ {check-name} is **PASSING**." Stop. |
| Pending / in progress | Print: "{check-name} is still running. Re-run after it completes." Stop. |
| `skipped` / `cancelled` | Print: "{check-name} was {conclusion} — not a test failure." Stop. |
| `failure` / `timed_out` / `action_required` | Continue to Step 4 (UT/SLIT) or Step 5 (E2E). |

---

## Step 4 — UT / SLIT: Fetch GitHub Actions Logs

**Only for UT and SLIT failures.** For E2E skip directly to Step 5.

Get the branch name for this PR, then find the matching run:

```bash
BRANCH=$(gh pr view <PR_URL> --json headRefName -q .headRefName)

gh run list --repo {org}/{repo} --branch "$BRANCH" \
  --json databaseId,name,status,conclusion,url --limit 10
```

Fetch failed job logs for the matching run:

```bash
gh run view {run_id} --repo {org}/{repo} --log-failed
```

Parse the log for:
- Failed test lines: `--- FAIL:`, `FAILED`, `Error:`, `AssertionError`, `FAIL\t`
- Error messages and stack traces immediately following each failure marker
- Exit summary (e.g., `FAIL github.com/razorpay/...`)

Print report:

```
## {UT|SLIT} Failure Report — PR #{pr_number}

**Check:** {check-name}
**Run:** {run_url}

### Failing Tests
| Test Name | Error |
|-----------|-------|
| {test}    | {first error line} |

### Summary
{N} test(s) failed. {Common theme if any.}
```

Stop after this report.

---

## Step 5 — E2E: Find Argo Workflow URL

Search ALL PR comments for an Argo workflow URL. **Always use `gh api` — never unauthenticated `curl` against `api.github.com`** (private repos return a 404 string body, causing `jq` to fail).

```bash
gh api repos/{org}/{repo}/issues/{pr_number}/comments --paginate \
  | jq '[
      .[] | select(.body | test("https://argo\\.dev\\.razorpay\\.in/workflows/"))
      | {id, user: .user.login, created_at, body}
    ] | sort_by(.created_at) | reverse'
```

Evaluate results:
- **No comments with Argo URL** → Print: "No Argo workflow URL found in PR comments." Stop.
- Take the **single most recent** comment.
- If it shows **only passing indicators** (green checks, "succeeded") → Print: "Latest E2E run is passing." Stop.
- If it shows **only coverage gate failures** (e.g. `| Code Change Coverage | ... | fail |`) with no step failures → Print: "Only coverage gate failed, not E2E steps. Add test coverage." Stop.
- Otherwise: extract the URL matching `https://argo\.dev\.razorpay\.in/workflows/[^\s"]+` and continue.

---

## Step 6 — E2E: Fetch Argo Workflow JSON

Parse the Argo URL:
- `https://argo.dev.razorpay.in/workflows/argo-workflows/e2e-x8ntl`
  → namespace: `argo-workflows`, workflow name: `e2e-x8ntl`

```bash
curl -s "https://argo.dev.razorpay.in/api/v1/workflows/{namespace}/{workflow-name}" \
  -o /tmp/workflow.json

# Verify
jq '.metadata.name // "FETCH FAILED"' /tmp/workflow.json
```

Extract key fields:

```bash
ARCHIVE_UID=$(jq -r '.metadata.uid' /tmp/workflow.json)
START=$(jq -r '.status.startedAt' /tmp/workflow.json)
END=$(jq -r '.status.finishedAt' /tmp/workflow.json)
```

List all failed Pod nodes (skip StepGroup/DAG orchestration nodes — they only fail because children failed):

```bash
jq '[
  .status.nodes | to_entries[]
  | select(.value.phase == "Failed" or .value.phase == "Error")
  | select(.value.type == "Pod")
  | {
      id:           .value.id,
      name:         (.value.displayName // .value.name),
      phase:        .value.phase,
      message:      (.value.message // ""),
      podName:      (.value.podName // ""),
      startedAt:    .value.startedAt,
      finishedAt:   .value.finishedAt,
      artifacts:    (.value.outputs.artifacts // [] | map(.name))
    }
] | sort_by(.startedAt)' /tmp/workflow.json > /tmp/failed-nodes.json

cat /tmp/failed-nodes.json
```

Print: `Found {N} failed E2E Pod nodes in workflow {workflow-name}.`

---

## Step 7 — E2E: Download Argo Logs as Text Files

Process each failed node **sequentially** (never parallel — mixing logs corrupts analysis).

For each node, check its `artifacts` list and pick a download strategy:

### Strategy A — JUnit artifact (`junit-test-results` present)

```bash
NODE_ID="{node.id}"

curl -s "https://argo.dev.razorpay.in/artifact-files/{namespace}/archived-workflows/${ARCHIVE_UID}/${NODE_ID}/outputs/junit-test-results" \
  -o /tmp/junit-{node-name}.txt

# Validate that the download contains actual JUnit XML content
if grep -q 'testname=\|<testsuites\|<testsuite' /tmp/junit-{node-name}.txt 2>/dev/null; then
  echo "→ /tmp/junit-{node-name}.txt (JUnit XML valid)"
else
  echo "⚠️  JUnit artifact empty or invalid — falling back to Strategy B for {node-name}"
  rm -f /tmp/junit-{node-name}.txt
  # [proceed with Strategy B commands below]
fi
```

> **Path details:** use `archived-workflows/{uid}/{nodeId}` — NOT `workflows/{name}`.
> Artifact name has **no `.xml` extension**: `junit-test-results`, not `junit-test-results.xml`.
> **Validation is mandatory.** The curl may succeed (HTTP 200) but return an empty body or an error
> JSON — always check the file content before treating it as JUnit. If validation fails, fall back
> to Strategy B immediately.

### Strategy B — Step logs (no JUnit artifact, or JUnit validation failed)

```bash
# Primary: main container
curl -s "https://argo.dev.razorpay.in/api/v1/workflows/{namespace}/{workflow-name}/log?podName={node.podName}&logOptions.container=main" \
  -o /tmp/e2e-log-{node-name}.txt

# Fallback: wait container (if main is empty)
if [ ! -s /tmp/e2e-log-{node-name}.txt ]; then
  curl -s "https://argo.dev.razorpay.in/api/v1/workflows/{namespace}/{workflow-name}/log?podName={node.podName}&logOptions.container=wait" \
    -o /tmp/e2e-log-{node-name}.txt
fi

echo "→ /tmp/e2e-log-{node-name}.txt"
```

### Strategy C — Archived workflow, empty logs

If both log endpoints return empty files: use the node's `message` field from `workflow.json` as the sole failure signal. Note: "Live logs unavailable — using workflow node message."

Print a download manifest:
```
Downloaded logs for {N} failed nodes:
  /tmp/junit-{name-1}.txt          (JUnit artifact)
  /tmp/e2e-log-{name-2}.txt        (Step log — main)
  ...
```

---

## Step 8 — E2E: Analyze Failures and Detect Patterns

Process each downloaded file sequentially.

**Root cause MUST come from the downloaded JUnit XML content and step log content.** Do NOT attempt to fetch BrowserStack URLs — they require authentication and cannot be accessed. If BrowserStack session URLs appear in the JUnit output, list them as reference links only (not as an analysis source).

**For JUnit files:**
```bash
# Failing test names
grep -oP '(?<=testname=")[^"]+' /tmp/junit-{name}.txt

# System-out: use -A 150 to capture full error context (assertion failures can appear deep in system-out)
grep -A 150 '<system-out>' /tmp/junit-{name}.txt | head -600

# Go test failure markers with full surrounding context
grep -B2 -A 40 '--- FAIL:' /tmp/junit-{name}.txt | head -600

# Assertion errors: actual vs expected values — the core failure reason
grep -iE '(Expected|Actual:|Got:|assert\.|require\.|StatusCode|status_code|response body|Error:|panic:)' \
  /tmp/junit-{name}.txt | grep -v '<!--' | head -100

# BrowserStack session URLs — list as reference links only (NOT used for analysis)
grep -oP 'https://observability\.browserstack\.com[^\s"<&]+' /tmp/junit-{name}.txt | sort -u
```
- Extract test names; strip `#01`/`#02` parameterized suffixes for grouping.
- Read failure details from `<system-out>`, not `<failure>` attributes (`<failure>` is often empty).
- **BrowserStack URLs**: list them per test as reference links in the report. Do not attempt to fetch them.

**For step log files:**
```bash
# Step 1: Find actual test failure markers (specific — avoids HTTP log noise)
grep -E '(--- FAIL:|FAIL\t|panic:|exit status [^0])' \
  /tmp/e2e-log-{name}.txt | head -50

# Step 2: For each --- FAIL: line found, capture the surrounding context
grep -B2 -A 20 '--- FAIL:' /tmp/e2e-log-{name}.txt | head -200

# Step 3: If no --- FAIL: markers found, look for infrastructure-level failures only
grep -E '(connection refused|dial tcp.*i/o timeout|no healthy upstream|upstream connect error)' \
  /tmp/e2e-log-{name}.txt | head -30
```

> **Do NOT use a broad `Error:` grep on step logs.** Step logs contain ALL container output
> including HTTP access logs, Kubernetes health checks, and proxy errors — grepping for `Error:`
> will match hundreds of unrelated lines (502/504 from k8s probes, etc.) and inflate counts.
> Use only the specific markers above.

**Build the complete failing test list:**

| # | Test Name | Node / Step | Error (first line) |
|---|-----------|-------------|-------------------|
| 1 | TestXxx   | {step-name} | {error}           |
| 2 | TestYyy   | {step-name} | {error}           |

**Common failure pattern detection:**

| Pattern | Signals |
|---------|---------|
| DB / network connectivity | `connection refused`, `dial tcp`, `EOF`, `timeout`, `i/o timeout` |
| Auth / credential failure | `401`, `403`, `unauthorized`, `token`, `authentication failed` |
| Service unavailable | `no healthy upstream`, `upstream connect error` (from step logs); `502`/`503`/`504` only if present in JUnit `<system-out>` assertion context |
| Nil pointer / data fixture | `nil pointer`, `not found`, `expected non-nil`, resource creation failure |
| Configuration / env var | `env var not set`, `config`, `splitz`, `feature flag`, missing property |
| Infra / deploy step failure | step name matches `deploy-to-devstack`, `clone-secrets` |

Group tests by shared pattern. A test can belong to multiple patterns.

---

## Step 9 — E2E: Generate Analysis Report

```
## E2E Failure Analysis — PR #{pr_number}

**Workflow:** {workflow-name}
**Argo URL:** {argo-url}
**Started:** {start UTC} / {start IST}
**Finished:** {end UTC} / {end IST}
**Failed Steps:** {N}

---

### All Failing E2E Tests ({total} total)

| # | Test Name | Step | Error Summary |
|---|-----------|------|---------------|
| 1 | {test}    | {step} | {one-line error} |
| 2 | {test}    | {step} | {one-line error} |

**BrowserStack Reference Links** (if found in JUnit output — requires auth to view):
- `{test-name}`: {bs-url}

---

### Common Failure Patterns

#### Pattern 1: {Name} — {N} tests affected
**Affected tests:** TestA, TestB, TestC
**Evidence:**
```
{first log line confirming the pattern}
```
**Root Cause:** {one sentence citing specific log evidence}
**Recommended Action:** {concrete fix}

#### Pattern 2: {Name} — {N} tests affected
...

---

### Downloaded Log Files

| File | Node | Strategy |
|------|------|---------|
| /tmp/junit-{name}.txt      | {node} | JUnit artifact |
| /tmp/e2e-log-{name}.txt    | {node} | Step log |

---

### Summary

- **Total failing tests:** {N}
- **Distinct failure patterns:** {M}
- **Primary cause:** {most impactful pattern}
```

---

## Step 10 — Copy Analysis to Clipboard for Slack

After generating the Step 9 report, write it as-is to a temp file and copy it to clipboard so it can be pasted directly into Slack. No reformatting — use the Step 9 output verbatim.

```bash
# Write Step 9 report directly to file (no reformatting)
cat > /tmp/e2e-analysis-slack.txt << 'SLACK_EOF'
{paste Step 9 report verbatim here}
SLACK_EOF

# Copy to clipboard (macOS / Linux)
cat /tmp/e2e-analysis-slack.txt | pbcopy 2>/dev/null \
  || xclip -selection clipboard < /tmp/e2e-analysis-slack.txt 2>/dev/null \
  || xsel --clipboard --input < /tmp/e2e-analysis-slack.txt 2>/dev/null \
  || echo "(clipboard copy unavailable — paste content above manually)"

echo "Analysis copied to clipboard — ready to paste into Slack."
echo "File saved at: /tmp/e2e-analysis-slack.txt"
```

---

## Step 11 — Emit Structured Output

After completing the analysis (Steps 4, 5, or 9), emit a structured output. Nexus always requests
a JSON response conforming to `ACTIVITY_LOG_SCHEMA` (fields: `title`, `details`).

**The `details` field must be set to exactly the report text you produced in Step 9 (or Step 4).
Do NOT summarize, restructure, or abbreviate it. Copy the full report text as a plain string.**

### E2E analysis completed (Step 9 done)

Read the report from the file written in Step 10:

```bash
cat /tmp/e2e-analysis-slack.txt
```

Then emit:

```json
{
  "title": "E2E: {N} tests failing — {workflow-name} [{primary_pattern_name}]",
  "details": "{output of cat /tmp/e2e-analysis-slack.txt — full text, no changes}"
}
```

### UT or SLIT analysis completed (Step 4 done)

```json
{
  "title": "{UT|SLIT}: {N} tests failing — PR #{pr_number}",
  "details": "{the full Step 4 report text you already printed, copied verbatim}"
}
```

### Check is passing, skipped, or cancelled (Step 3 early-stop)

```json
{
  "title": "{action_type}: {check-name} is {conclusion} — PR #{pr_number}",
  "details": "{check-name} is {passing|skipped|cancelled|running}."
}
```

---

## Step 12 — Cleanup Downloaded Log Files

After emitting structured output, delete all temporary files downloaded during the analysis:

```bash
rm -f /tmp/workflow.json /tmp/failed-nodes.json
rm -f /tmp/junit-*.txt
rm -f /tmp/e2e-log-*.txt
rm -f /tmp/main-logs-*.txt
rm -f /tmp/e2e-analysis-slack.txt
```

Print: `Cleaned up {N} temporary log file(s).`

---

## Quick Reference — Argo API

| Operation | URL Pattern |
|-----------|------------|
| Fetch workflow JSON | `https://argo.dev.razorpay.in/api/v1/workflows/{ns}/{name}` |
| Download JUnit artifact | `https://argo.dev.razorpay.in/artifact-files/{ns}/archived-workflows/{uid}/{nodeId}/outputs/junit-test-results` |
| Download step log (main) | `https://argo.dev.razorpay.in/api/v1/workflows/{ns}/{name}/log?podName={pod}&logOptions.container=main` |
| Download step log (wait) | `https://argo.dev.razorpay.in/api/v1/workflows/{ns}/{name}/log?podName={pod}&logOptions.container=wait` |

**No authentication required** for `argo.dev.razorpay.in`.

---
name: feature-poc-finder
description: Identifies the right people to contact for a feature by searching across GitHub (code/PR/commit history), DevRev (tickets), and Slack (conversations). Use when you need to know who owns or has worked on a particular feature, module, or system — before raising a PR, asking for a review, or debugging an unfamiliar area.
mcp_servers: [devrev-mcp, slack-mcp]
---

# Feature POC Finder

## Overview

Identifies the right people to contact for a feature by searching across GitHub (code/PR/commit history), DevRev (tickets), and Slack (conversations). Use this skill when you need to know who owns or has worked on a particular feature, module, or system — before raising a PR, asking for a review, or debugging an unfamiliar area.

## Prerequisites

The skill works best when the following integrations are configured, but **none are hard blockers** — the skill will proceed with whatever is available and tell you what was skipped.

**MCP Requirements:**
- DevRev MCP server must be configured and active
- Slack MCP server must be configured and active
- Required MCP tools:
  - `devrev → get_self` - Verify DevRev connection
  - `devrev → hybrid_search` - Search historical tickets by keyword
  - `slack_search_public_and_private` - Search Slack conversations by keyword

| Source | Setup | Impact if missing |
|--------|-------|-------------------|
| **GitHub CLI** (`gh`) | Run `gh auth login` in terminal | GitHub code/PR/commit search unavailable |
| **GitHub MCP** | Configure in Claude settings | Alternative to CLI; either one is sufficient |
| **DevRev MCP** | Run `/rzp-discover:connect-mcps` | Ticket history and assignees unavailable |
| **Slack MCP** | Run `/rzp-discover:connect-mcps` | Conversation-based signals unavailable |

If none are configured, the skill will inform you and exit gracefully rather than returning empty results silently.

> **Tip:** GitHub (CLI or MCP) provides the strongest signal — code authorship is the most durable indicator of ownership. Results are still useful with only GitHub configured.

> **Setup help:** See [references/setup-guide.md](references/setup-guide.md) for step-by-step instructions on configuring each source, including the Razorpay-specific Slack MCP IT approval process.

---

## Known Tradeoff: Partially Implemented Features

If a feature is **partially implemented** (e.g., you've already raised some PRs or commits exist under your name), this skill may surface **you** as a top POC. This is expected behavior — the skill reflects historical signal, not intent or future ownership.

In this case:
- Use your judgment to exclude yourself from the results
- Look at the second and third ranked contributors as the actual contacts
- If you're looking for a reviewer specifically, filter for people who have *reviewed* PRs in the area, not just authored them

---

## Step 0: Pre-flight Validation

Before doing anything else, run the following checks and report status to the user.

> **IMPORTANT:** All checks in this step must be executed **directly in the main session**. Do NOT delegate any of these checks to a Task agent or sub-agent. Task agents run in isolated environments and do not have access to MCP tools (Slack, DevRev) configured in the main session.

### GitHub CLI

```bash
gh auth status
```

- **Pass:** output contains `Logged in to github.com` → mark GitHub CLI as ✓
- **Fail:** command not found or not authenticated → tell the user:
  > "GitHub CLI is not authenticated. Run `gh auth login` in your terminal, then retry."
  > "GitHub search will be skipped until this is resolved."
  > "See [setup-guide.md](references/setup-guide.md) for installation and auth instructions."

**If GitHub CLI is ✓**, immediately ask the user to grant permission for all `gh` commands upfront before proceeding:

> "This skill will run several `gh` commands (code search, PR list, commit history) across one or more repos. To avoid repeated permission prompts, please select **'Allow for this session'** or **'Always allow'** when Claude asks to run `gh` commands."

Wait for the user to acknowledge before continuing to Step 1.

### DevRev MCP


Call `devrev → get_self` (no parameters).


- **Pass:** returns a valid user object → mark DevRev as ✓
- **Fail:** tool errors or returns nothing → tell the user:
  > "DevRev MCP is not connected. Run `/rzp-discover:connect-mcps` in Claude Code to set it up."
  > "Ticket history and assignee information will be unavailable."
  > "See [setup-guide.md](references/setup-guide.md) for manual setup instructions."

### Slack MCP

**Call directly from the main session — do NOT use a Task agent.** Probe Slack connectivity by attempting a minimal search. Try the following tools in order until one succeeds:
1. `slack_search_public_and_private` with `query: test`, `limit: 1`
2. `slack_search_messages` with `query: test`, `count: 1`

- **Pass:** any of the above responds (even with empty results) → mark Slack as ✓
- **Fail:** all attempts error or no Slack tool is found → tell the user:

  > "Slack MCP is not connected. Run `/rzp-discover:connect-mcps` in Claude Code to set it up."
  > "Conversation-based signals will be unavailable."
  > "See [setup-guide.md](references/setup-guide.md) for IT approval steps (Razorpay-specific)."

### Summary

Print a status line before proceeding:

> "Source availability: GitHub CLI ✓ | DevRev ✓ | Slack ✗"
> "Proceeding with available sources. Results may be incomplete."

**Do not block execution if one or more sources are unavailable.** Proceed to Step 1 with whatever is active. Only exit if **all three** sources are unavailable — in that case, tell the user there's nothing to search and ask them to configure at least one source.

---

## Step 1: Accept Feature Input

Ask the user for a description of the feature or area they're investigating. Accept either:
- Free-form text (e.g., "UPI mandate flows in checkout")
- A file path or module name (e.g., `app/Service/UTILS/EGrass/`)

**Extract 3–5 key technical search terms** from the input. Examples:
- "UPI mandate payment flow" → `upi`, `mandate`, `payment_flow`, `UpiMandate`
- "reconciliation handoff recon" → `handoffrecon`, `settlement_recon`, `reconcil`
- "checkout config for NTRP" → `NTRP`, `StandardCheckout`, `checkout_config`

These terms will be used as search queries across all sources.

**Also note:** if the user provides specific repo names, skip Step 2 entirely and use those repos directly.

---

## Step 2: Identify Target Repos

**If the user provided repo names:** use them directly. Skip this step.

**Otherwise:** use the **Skill tool** (not the Task tool) to run `/rzp-discover:brainstorm`. Pass the feature description as the argument and ask it to return:
- Relevant subgroup names (e.g., "checkout", "payments-processing-platform")
- Repo names within those subgroups (e.g., `checkout`, `api`, `pg-router`)

Use the top 2–3 repos for the search steps below. If discovery returns more than 5, ask the user to narrow down.

---

## Step 3: Search GitHub

**Use the Skill tool (not the Task tool) to run `/rzp-discover:github-cli`.**

For each target repo, search for:

1. **Code matches** — files containing the extracted keywords:
   ```bash
   gh search code "<keyword>" --repo razorpay/<repo> --limit 10
   ```

2. **PR authors** — PRs touching relevant files or mentioning the feature:
   ```bash
   gh pr list --repo razorpay/<repo> --search "<keyword>" --state all --limit 20 --json number,title,author,mergedAt
   ```

3. **Commit authors** — who last touched relevant files:
   ```bash
   gh api repos/razorpay/<repo>/commits?path=<file_path>&per_page=10
   ```

Collect from GitHub:
- GitHub usernames of PR authors
- GitHub usernames of recent commit authors on relevant files
- PR numbers and titles for reference

---

## Step 4: Search DevRev (if available)

Search for tickets/issues matching the extracted keywords using `devrev → hybrid_search`. For each keyword:
- Find open and recently closed tickets related to the feature
- Note the ticket ID, title, assignee, and any commenters

Run up to 3 parallel searches, one per keyword extracted in Step 1. Use `projection_type: summary` and limit 10 results.

Collect from DevRev:
- Ticket IDs (e.g., `TKT-1234`)
- Assignees and active commenters
- Links to relevant tickets

If DevRev is not configured, skip this step and note it in the output.

---

## Step 5: Search Slack (if available)

**Call directly from the main session — do NOT use a Task agent.** Run one call per extracted keyword using whichever Slack search tool is available:
- `slack_search_public_and_private` with `query: <keyword>`, `limit: 20`, `sort: relevance`
- OR `slack_search_messages` with `query: <keyword>`, `count: 20`, `sort_by: relevance`

Collect:
- Display names / Slack user IDs of people who posted relevant messages
- Channel names and thread timestamps for reference

If Slack MCP is not connected, skip this step and note it in the output.

---

## Step 6: Compile & Output Results

After Steps 3–5 complete, deduplicate people across sources.

**Merge rule:** if the same person appears under different identifiers (GitHub username vs. Slack display name vs. DevRev assignee), consolidate into one row. Use best-effort matching on name similarity; flag ambiguous cases.

Output the results as a Markdown table:

```
| Person           | GitHub (PRs / Commits)                     | Slack (Channels / Threads)        | DevRev (Tickets)       |
|------------------|--------------------------------------------|-----------------------------------|------------------------|
| @github-user     | PR #123 in checkout — "Add UPI mandate"    | #payments thread (2024-11-01)     | TKT-456 (assignee)     |
| @another-user    | Commit in api/UpiMandate.php (2024-10-15)  | —                                 | TKT-789 (commenter)    |
```

Below the table, add:
- **Top 3 recommended POCs** — people appearing in 2+ sources or as PR/ticket assignees
- **Key files/areas touched** — a brief list of the most relevant files found
- **Note if the requester appears in results** — flag it so they can self-exclude and look at the next candidates

---

## Graceful Degradation Notes

When sources are unavailable, tell the user:

> "Slack search was skipped because the Slack token isn't configured. Results can be more comprehensive with this source. Run `/rzp-discover:connect-mcps` to set it up."

> "DevRev search was skipped because the DevRev integration isn't configured. Ticket history and assignee information won't be available. Run `/rzp-discover:connect-mcps` to set it up."

> "GitHub search was skipped because neither `gh` CLI nor GitHub MCP is configured. Run `gh auth login` in your terminal or set up the GitHub MCP."

Results with only GitHub will still be useful — GitHub covers the most durable signal (code authorship and PR history).

---

## Key Design Rules

- **Parallel execution:** Steps 3 (GitHub) and 4 (DevRev) must be launched as parallel Task agents. Step 5 (Slack) uses a direct MCP call. All three should be initiated in a single message. Do not run them sequentially.
- **User-provided repos take priority:** If the user specifies repos, skip Step 2 (brainstorm) entirely.
- **Keyword quality matters:** Extract technical identifiers (function names, class names, route paths, file names) — not generic English words. Bad: `payment`. Good: `handoffrecon`, `EGrassHandoffRecon`, `settlementrecon`.
- **Deduplication is required:** Do not list the same person multiple times. Merge all references for a person into one row.
- **No hard blocks:** Never exit or refuse to run because a source is unavailable. Always proceed with what's configured.
- **Do not expose credentials or auth tokens** found in code. Redact them in output.
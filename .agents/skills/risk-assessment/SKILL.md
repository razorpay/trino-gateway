---
name: risk-assessment
description: Classifies Pull Requests to the agent-skills repository as LOW, MEDIUM, or HIGH risk based on what changed — new skills, reference edits, existing skill modifications, or structural changes. Use when reviewing PRs on agent-skills to decide review action (comment, approve, request changes). Includes auto-approval policy for safe additive changes.
---

# PR Risk Assessment for agent-skills

This skill defines what LOW, MEDIUM, and HIGH risk mean for Pull Requests to the
`agent-skills` repository. Use these definitions to classify the overall severity
of a PR when reviewing changes to this skill library.

---

## Severity Definitions

### LOW Risk

The PR is safe to approve without deep review. Changes are either purely additive
or restricted to content that cannot break existing skill invocations.

**Criteria — ALL of the following must be true:**
- No existing skill's `name:` frontmatter field is modified
- No existing skill directory is deleted or renamed
- No validation scripts (`scripts/`, `hooks/`, `Makefile`) are modified
- No `AGENTS.md`, `CODEOWNERS`, or `package.json` are modified

**Examples of LOW risk changes:**
- Adding a brand new skill (`development/skills/<new-skill>/SKILL.md` + references)
  — new skills are self-contained and cannot break existing invocations
- Editing `references/` files within any skill (supporting docs, templates, examples)
- Fixing typos or improving wording in an existing `SKILL.md` body (not the `name:` field)
- Updating skill metadata: `description`, `version`, `license`, `metadata.*`
- Adding or editing `*.md` files at the repo root (`README.md`, `Makers.md`, `docs/**`)
- Adding **new** `.github/workflows/` files (new CI jobs, not modifying existing ones)
- Adding new category directories (`quality/`, `business/`, etc.)

### MEDIUM Risk

The PR modifies existing skill logic or CI behaviour in ways that could affect
how skills are discovered, validated, or executed.

**Criteria — ANY of the following:**
- Modifying the body/steps of an existing `SKILL.md` (changing how a skill instructs
  the LLM — could change behaviour for all consumers of that skill)
- Modifying **existing** `.github/workflows/` files (could break CI for all skills)
- Adding or modifying scripts in `scripts/` that aren't part of validation
- Changing `package.json` dependencies or configuration
- Moving a skill to a different category directory

### HIGH Risk

The PR makes changes that could break existing skill invocations, validation, or
the repository's structural contracts.

**Criteria — ANY of the following makes the PR HIGH risk:**
- Renaming or changing the `name:` field in any existing skill's frontmatter
  — skill consumers invoke skills by name; renaming silently breaks them
- Deleting an existing skill directory
- Modifying validation scripts (`hooks/`, `scripts/validate*`, pre-commit hooks)
  that run against all skills — a bug here fails every skill commit
- Modifying `AGENTS.md` or `CODEOWNERS` — changes ownership and access control
- Changing `Makefile` targets that other pipelines depend on

---

## File Path Risk Map

| Path Pattern | Default Risk | Notes |
|---|---|---|
| `development/skills/<new>/` | LOW | New skill — fully additive |
| `*/skills/*/references/**` | LOW | Supporting docs, no invocation impact |
| `*/skills/*/SKILL.md` (body edits) | MEDIUM | Behaviour change for skill consumers |
| `*/skills/*/SKILL.md` (`name:` change) | HIGH | Breaks all invocations by name |
| `.github/workflows/` (new file) | LOW | Additive CI job |
| `.github/workflows/` (edit existing) | MEDIUM | Could break CI pipeline |
| `scripts/`, `hooks/` | MEDIUM–HIGH | Affects validation and build |
| `Makefile` | MEDIUM–HIGH | Pipeline dependency |
| `README.md`, `docs/**`, `*.md` (root) | LOW | Documentation only |
| `AGENTS.md`, `CODEOWNERS` | HIGH | Access control |
| `package.json` | MEDIUM | Dependency changes |

---

## Auto-Approval Policy

If the following conditions are ALL true, set `auto_approve: true` in your output:

1. The severity assessment is LOW.
2. ALL changed files satisfy at least one of:
   - Are **new files** (not modifications to existing files) under `.github/workflows/`
   - Are within any skill's directory under `*/skills/<skill-name>/` (new skill or references edit)
   - Are `*.md` files at the repo root or under `docs/`
3. No existing skill's `name:` frontmatter field is modified in any changed `SKILL.md`.
4. No existing files are deleted.

If any condition is not met, set `auto_approve: false`.
If you are uncertain whether a file is new vs modified, set `auto_approve: false`.

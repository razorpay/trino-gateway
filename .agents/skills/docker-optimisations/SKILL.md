---
name: docker-optimisations
description: Optimise Dockerfiles in repositories for better Docker layer caching. Reorders layers so static/rarely-changing layers are at the top and build-specific ARGs/ENVs are placed after the cache-busting source copy.
version: "1.0.0"
category: "CI/CD Optimisation"
author: "razorpay"
user_invocable: true
---

# Docker Optimisations Skill

Optimise Dockerfiles in a repository for best Docker layer caching by reordering layers so that static, rarely-changing instructions are at the top and build-specific values are placed after the cache-busting source code copy.

## Usage

```
/docker-optimisations optimise <service-name> [--branch <branch-name>]
```

**Examples:**
```
/docker-optimisations optimise shield
/docker-optimisations optimise pg-router
/docker-optimisations optimise 1cc-consumer-app --branch chore/dockerfile-caching
```

## How to interpret arguments

When this skill is invoked, parse the arguments as follows:

1. The argument after `optimise` (or `optimize`) is the **service/repo name**.
2. Treat the service name as a directory name under the current working directory or its parent.
3. If `--branch <branch-name>` is provided, use that as the git branch name. Otherwise, default to `docker-action-migration`.

---

## Reference Files

Before proceeding, read the following reference files from the `references/` directory next to this SKILL.md. These contain the detailed rules, patterns, and templates needed to execute this skill:

- **`references/anti-patterns.md`** — Anti-pattern definitions and detection checklist
- **`references/optimisation-rules.md`** — Transformation rules (Rules 1-6) with Dockerfile templates
- **`references/reference-example.md`** — pg-router Dockerfile as a known-good caching example
- **`references/git-and-pr.md`** — Git branch setup, commit message, PR template, and cleanup
- **`references/proto-commit-pinning.md`** — Proto fetch determinism fix (Makefile / scripts/variables.mk)

---

## Execution Steps

### Step 1: Locate or clone the repo

Search for the repo directory in this order:
1. Current working directory: `./<service-name>`
2. Sister directory: `../<service-name>`

If NOT found in either location, clone it using the GitHub CLI:
```bash
gh repo clone razorpay/<service-name> ./<service-name>
```

If the clone fails, report the error and stop.

### Step 2: Git branch setup

Follow the instructions in **`references/git-and-pr.md`** for branch setup. Use the user-provided branch name, or `docker-action-migration` if none was specified.

### Step 3: Find all Dockerfiles

Search recursively for all Dockerfiles in the repo:
- `**/Dockerfile`
- `**/Dockerfile.*`

Log all Dockerfiles found. If none exist, report "No Dockerfiles found" and stop.

### Step 4: Analyse each Dockerfile

Read each Dockerfile and check for anti-patterns described in **`references/anti-patterns.md`**. Use the detection checklist at the bottom of that file.

### Step 4.5: Check proto commit pinning

Check `Makefile` and `scripts/variables.mk` for proto fetch patterns using the rules in **`references/proto-commit-pinning.md`**.

```bash
grep -n "PROTO_BRANCH\|PROTO_COMMIT_SHA" Makefile scripts/variables.mk scripts/common.mk 2>/dev/null
```

If `PROTO_BRANCH` exists but `PROTO_COMMIT_SHA` is missing, or `git checkout` uses `origin/$(PROTO_BRANCH)`, apply the fix. This is a cache-determinism issue — proto fetched by branch busts the Docker layer cache on every proto commit.

### Step 5: Apply optimisations

For each Dockerfile that has anti-patterns, apply the transformation rules from **`references/optimisation-rules.md`** (Rules 1-6). Use the pg-router example in **`references/reference-example.md`** as the gold standard when in doubt.

Also apply the proto commit pinning fix if detected in Step 4.5.

### Step 6: Verify changes

After applying optimisations, verify each modified Dockerfile:

1. **Structure check**: Ensure the Dockerfile is syntactically valid (all FROM, RUN, COPY, ENV, ARG, WORKDIR, EXPOSE, ENTRYPOINT instructions are properly formed)
2. **Completeness check**: Ensure no instructions were accidentally lost during reordering
3. **Order check**: Verify that:
   - No build-specific ARG/ENV appears before `go mod download` in builder stages
   - No build-specific ARG/ENV appears before `apk add` in distribution stages
   - `COPY go.mod` / `COPY go.sum` appear before `go mod download`
   - `ADD . /src` appears after `go mod download`
   - `GIT_TOKEN` is used via `--mount=type=secret,id=GIT_TOKEN` with `GIT_ASKPASS`, not via `ARG GIT_TOKEN`

Log each file's verification result.

### Step 7: Commit and create/update PR

Follow the commit and PR instructions in **`references/git-and-pr.md`**.

### Step 8: Cleanup

Follow the cleanup instructions in **`references/git-and-pr.md`**.

---

## Important Rules

1. **NEVER delete or add new Dockerfile instructions** unless consolidating duplicates or removing redundant ones. Only REORDER existing instructions.
2. **Preserve all functionality.** The built image must be identical - only build speed changes.
3. **Preserve all COPY --from=builder paths, EXPOSE ports, ENTRYPOINT commands** exactly as they were.
4. **Convert `ARG GIT_TOKEN` to `--mount=type=secret`** (Rule 6). If `GIT_TOKEN` is already using `--mount=type=secret` with `GIT_ASKPASS`, preserve it as-is. If it uses `ARG GIT_TOKEN` with `git config` or `.netrc`, convert it to the `--mount=type=secret` + `GIT_ASKPASS` pattern so token value changes don't bust cache.
5. **Only modify Dockerfiles that have anti-patterns.** If a Dockerfile is already well-ordered, skip it.
6. **Use pg-router Dockerfiles as the reference** for correct ordering patterns. When in doubt, check `references/reference-example.md`.
7. **Log every change** with the file path and what was reordered.
8. **Handle non-Go Dockerfiles too.** The same principles apply: static package installs before dynamic ARGs, dependency caching before source copy. Adapt the Go-specific rules (go.mod, go mod download) to the equivalent for other languages (package.json/npm install, requirements.txt/pip install, etc.).
9. **Always check and fix proto commit pinning.** If the repo fetches proto files and uses `PROTO_BRANCH` without `PROTO_COMMIT_SHA`, apply the fix from `references/proto-commit-pinning.md`. Proto fetched by branch is non-deterministic and busts Docker layer cache. Do NOT auto-fix if `PROTO_BRANCH` is set to a non-master value — flag for manual review instead.

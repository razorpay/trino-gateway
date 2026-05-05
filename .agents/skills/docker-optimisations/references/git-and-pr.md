# Git Branch Setup, Commit, and PR

## Branch name

Use the branch name provided by the user via `--branch`. If not provided, default to `ci-optimisations`.

**If the user supplies a branch name, use it exactly as provided** — do not modify, prefix, or sanitise it. The branch is shared across repos in the same batch; always use the same resolved name for every repo in the run.

All references to `<branch-name>` below should be replaced with the resolved branch name.

## Step 2: Git branch setup

Navigate to the repo and set up the branch:

```bash
cd <repo-path>
```

Check if the branch `<branch-name>` already exists (locally or remotely):

```bash
git fetch origin
git branch -a | grep <branch-name>
```

**If the branch exists remotely but not locally:**
```bash
git checkout -b <branch-name> origin/<branch-name>
git pull origin <branch-name>
```

**If the branch exists locally:**
```bash
git checkout <branch-name>
git pull origin <branch-name> 2>/dev/null || true
```

**If the branch does NOT exist at all:**
```bash
git checkout master && git pull origin master
git checkout -b <branch-name>
```

## Step 7: Commit and create/update PR

Stage and commit the changes:

```bash
git add <modified-dockerfile-paths>
git commit -m "$(cat <<'EOF'
chore: optimise Dockerfiles for better layer caching

- Reordered layers to maximise Docker build cache hits
- Moved build-specific ARG/ENV after source copy (cache-busting boundary)
- Consolidated duplicate package installations
- Removed redundant go mod download calls
- Split static operations from dynamic ones in distribution stage

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

Push the branch:
```bash
git push origin <branch-name>
```

Check if a PR already exists for this branch:
```bash
gh pr list --head <branch-name> --state open
```

**If a PR already exists:** The push is sufficient, the PR will auto-update.

**If no PR exists:** Create one:
```bash
gh pr create \
  --title "chore: optimise Dockerfiles for better Docker layer caching" \
  --body "$(cat <<'EOF'
## Summary
- Optimised Dockerfile layer ordering for better Docker build cache utilisation
- Moved build-specific `ARG`/`ENV` instructions (`BRANCH_NAME`, `CODE_COVERAGE`, `GIT_COMMIT_HASH`) after the cache-busting source copy
- Consolidated duplicate `RUN apk add` calls into single instructions
- Split static operations from dynamic ones so static layers remain cached
- Added `ENV CGO_ENABLED 0` and `ENV GOPRIVATE` as static layers at top of builder stages

## Why
Previously, changing build args like `BRANCH_NAME` or `CODE_COVERAGE` would invalidate the Docker cache for ALL subsequent layers, causing expensive operations like `apk add` and `go mod download` to re-run on every build. With this fix, those layers remain cached as long as `go.mod`/`go.sum` haven't changed.

## Changes
- Reordered builder stage: static ENV -> apk install -> go.mod/sum copy -> go mod download -> source copy -> build-specific ARGs -> build
- Reordered distribution stage: static ENV -> apk install -> mkdir -> build-specific ARGs -> commit.txt -> COPY binaries
- Removed redundant debug `RUN echo` lines from cached sections
- Removed duplicate `go mod download` calls

## Test plan
- [ ] Verify Dockerfile syntax is valid
- [ ] Confirm Docker builds succeed with the reordered layers
- [ ] Verify layer caching works (second build should skip apk/go mod download if go.mod unchanged)

Generated with Claude Code docker-optimisations skill
EOF
)"
```

Report the PR URL.

## Step 8: Cleanup

Switch back to the original branch:
```bash
git checkout master
```

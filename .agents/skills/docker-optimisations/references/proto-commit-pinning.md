# Proto Commit Pinning Reference

Proto fetched by branch is non-deterministic — Docker layer cache is busted every time the branch tip advances. Pinning to a commit SHA makes the fetch deterministic and cache-friendly. This is the same cache-determinism goal as Dockerfile layer reordering.

## Detection

Check root `Makefile` and `scripts/variables.mk` for proto variables:

```bash
grep -n "PROTO_BRANCH\|PROTO_COMMIT_SHA" Makefile scripts/variables.mk scripts/common.mk 2>/dev/null
```

**Apply the fix if**:
- `PROTO_BRANCH` exists but `PROTO_COMMIT_SHA` is missing
- `git checkout` uses `origin/$(PROTO_BRANCH)` instead of `$(PROTO_COMMIT_SHA)`

**Skip if**:
- `PROTO_BRANCH` is set to a non-`master` value (e.g. a feature branch) → flag for manual review, do not auto-fix
- No `PROTO_BRANCH` anywhere → proto may be fetched differently, skip

---

## Correct Pattern — root Makefile

```makefile
# Pin to a specific proto commit for deterministic, cache-friendly builds
PROTO_COMMIT_SHA ?= 3bbd3a85ceab352c5610b2d1816a723b2a5870b7
# Accept a branch override for dev testing
PROTO_BRANCH ?= master

proto-fetch: ## Fetch proto files from remote repo
    @echo "\n + Fetching proto files from commit: $(PROTO_COMMIT_SHA) \n"
    @mkdir $(PROTO_ROOT) && \
    cd $(PROTO_ROOT) && \
    git init --quiet && \
    git config core.sparseCheckout true && \
    cp $(CURDIR)/scripts/proto_modules .git/info/sparse-checkout && \
    git remote add origin $(PROTO_GIT_URL) && \
    git fetch origin $(PROTO_BRANCH) --quiet && \
    git checkout $(PROTO_COMMIT_SHA) --quiet
```

## Correct Pattern — scripts/variables.mk variant (e.g. ads-server, ads-tracker)

```makefile
# scripts/variables.mk
PROTO_COMMIT_SHA := 3bbd3a85ceab352c5610b2d1816a723b2a5870b7
PROTO_BRANCH ?= master
```

The proto-fetch target in `scripts/common.mk` uses `$(PROTO_COMMIT_SHA)` for checkout and `$(PROTO_BRANCH)` for fetch.

---

## Key Points

| Element | Value | Why |
|---|---|---|
| `PROTO_COMMIT_SHA ?=` | `3bbd3a85ceab352c5610b2d1816a723b2a5870b7` | Pinned SHA — Docker cache keyed on this value |
| `PROTO_BRANCH ?=` | `master` | Dev override: `make proto-fetch PROTO_BRANCH=my-branch PROTO_COMMIT_SHA=<sha>` |
| `git fetch origin $(PROTO_BRANCH)` | fetches the branch ref | Makes the SHA reachable in a shallow clone |
| `git checkout $(PROTO_COMMIT_SHA)` | checks out exact commit | Deterministic regardless of branch tip |

---

## Wrong Patterns to Fix

```makefile
# ❌ checkout by branch — non-deterministic, busts Docker cache on every proto commit
git checkout origin/$(PROTO_BRANCH) --quiet

# ❌ PROTO_COMMIT_SHA missing entirely
PROTO_BRANCH ?= master
git checkout origin/$(PROTO_BRANCH) --quiet

# ❌ fetch by SHA directly — SHA not reachable in shallow clone
git fetch origin $(PROTO_COMMIT_SHA) --quiet && \
git checkout $(PROTO_COMMIT_SHA) --quiet

# ❌ hardcoded master — no dev override possible
git fetch origin master --quiet && \
git checkout origin/master --quiet
```

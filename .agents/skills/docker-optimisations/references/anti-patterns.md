# Anti-Patterns That Hurt Docker Layer Caching

## Anti-Pattern 1: Build-specific ARG/ENV before static layers (CRITICAL)

`ARG` and `ENV` instructions for values that change per build (e.g., `BRANCH_NAME`, `CODE_COVERAGE`, `GIT_COMMIT_HASH`, `IMAGE_TAG`, `BUILD_NUMBER`) placed BEFORE static layers like `RUN apk add`, `COPY go.mod`, or `RUN go mod download`.

When an `ARG` value changes, Docker invalidates the cache for that layer AND all subsequent layers. So placing changing ARGs at the top causes expensive operations like package installation and dependency downloads to re-run on every build.

## Anti-Pattern 2: Duplicate package installation

Multiple `RUN apk add` (or `apt-get install`) calls that install overlapping packages. These should be consolidated into a single `RUN` instruction.

## Anti-Pattern 3: Missing static environment variables

Go-based builder stages missing `ENV CGO_ENABLED 0` and `ENV GOPRIVATE github.com/razorpay/*` at the top. These are static values that never change and should be set early.

## Anti-Pattern 4: Redundant go mod download after source copy

A `go mod download` after `ADD . /src/` is redundant if there's already one before the source copy (which is the correct caching pattern). The dependencies are already downloaded.

## Anti-Pattern 5: GIT_TOKEN passed as ARG instead of Docker secret

Using `ARG GIT_TOKEN` to pass the Git token as a build argument means that any change to the token value invalidates the cache for that layer and all subsequent layers (including `go mod download`). Instead, `GIT_TOKEN` should be passed via `--mount=type=secret,id=GIT_TOKEN` so the token value is never baked into any layer and changes to it do not bust the cache.

```dockerfile
# BAD - token value change busts cache for go mod download
ARG GIT_TOKEN
RUN git config --global url."https://${GIT_TOKEN}@github.com/".insteadOf "https://github.com/" && \
    go mod download

# GOOD - token mounted as secret, value changes don't affect caching
RUN --mount=type=secret,id=GIT_TOKEN set -eux && \
    printf '#!/bin/sh\nexec cat /run/secrets/GIT_TOKEN\n' > /tmp/askpass.sh && \
    chmod +x /tmp/askpass.sh && \
    export GIT_ASKPASS=/tmp/askpass.sh && \
    go mod download
```

Also watch for `.netrc`-based patterns that use `ARG GIT_TOKEN`:
```dockerfile
# BAD - ARG value change busts cache
ARG GIT_TOKEN
RUN echo "machine github.com login ${GIT_TOKEN}" > ~/.netrc && \
    go mod download

# GOOD - use --mount=type=secret instead
RUN --mount=type=secret,id=GIT_TOKEN set -eux && \
    printf '#!/bin/sh\nexec cat /run/secrets/GIT_TOKEN\n' > /tmp/askpass.sh && \
    chmod +x /tmp/askpass.sh && \
    export GIT_ASKPASS=/tmp/askpass.sh && \
    go mod download
```

## Anti-Pattern 6: Dynamic operations mixed with static operations

Operations that depend on build args (like `echo "$GIT_COMMIT_HASH" > commit.txt`) combined in the same `RUN` instruction with static operations (like `mkdir -p /app/public`). These should be split so the static mkdir can be cached.

## Detection Checklist

Use this checklist when analysing each Dockerfile:

- [ ] Are `ARG BRANCH_NAME` / `ARG CODE_COVERAGE` / `ARG GIT_COMMIT_HASH` placed BEFORE `apk add` or `go mod download`?
- [ ] Are there `ENV` instructions referencing build args before static layers?
- [ ] Are there `RUN echo "..."` debug lines using dynamic envs in the cached section?
- [ ] Are there multiple `RUN apk add` (or `apt-get install`) calls with overlapping packages?
- [ ] Is `go mod download` missing the go.mod/go.sum-first pattern?
- [ ] Is there a redundant `go mod download` after `ADD . /src/`?
- [ ] Are `mkdir` and `echo "$GIT_COMMIT_HASH"` in the same RUN in the distribution stage?
- [ ] Is `ENV CGO_ENABLED 0` / `ENV GOPRIVATE` missing from Go builder stages?
- [ ] Is `GIT_TOKEN` passed via `ARG GIT_TOKEN` instead of `--mount=type=secret,id=GIT_TOKEN`?
- [ ] Is there a `RUN git config ... ${GIT_TOKEN}` or `RUN echo ... ${GIT_TOKEN} > ~/.netrc` that bakes the token into a layer?
- [ ] Are there Dockerfile instructions with accidental leading whitespace?

# Optimisation Rules

For each Dockerfile that has anti-patterns, apply the following transformations. The reference pattern is based on the pg-router Dockerfiles.

## Rule 1: Builder stage ordering (Go services)

The correct order for a Go service builder stage is:

```dockerfile
# Stage 1 - Compilation build stage
######################################
FROM <base-image> as builder

# 1. Static environment variables (NEVER change)
ENV CGO_ENABLED 0
ENV GOPRIVATE github.com/razorpay/*

# 2. Static working directory setup
ENV SRC_DIR=<source-dir>
WORKDIR $SRC_DIR

# 3. Static package installation (consolidated into ONE call)
RUN apk add --update --no-cache --repository https://dl-4.alpinelinux.org/alpine/latest-stable/community/ <all-packages> && update-ca-certificates 2>/dev/null || true

# 4. Dependency files ONLY (cached unless go.mod/go.sum change)
# Avoid copying the whole source code first since that will invalidate
# cache for all further layers
COPY go.mod .
COPY go.sum .
# OR: ADD go.mod go.sum $SRC_DIR/

# 5. Download dependencies (cached unless go.mod/go.sum change)
# GIT_TOKEN MUST be mounted as a secret so token value changes don't bust cache
RUN --mount=type=secret,id=GIT_TOKEN set -eux && \
    printf '#!/bin/sh\nexec cat /run/secrets/GIT_TOKEN\n' > /tmp/askpass.sh && \
    chmod +x /tmp/askpass.sh && \
    export GIT_ASKPASS=/tmp/askpass.sh && \
    go mod download
## Don't change the above lines since these lines are cached with cache image

# 6. Copy ALL source code (BUSTS CACHE - everything below re-runs)
# Copy rest of the source code
ADD . $SRC_DIR

# 7. Build-specific ARGs/ENVs (placed AFTER source copy since cache is already busted)
ARG BRANCH_NAME
ARG CODE_COVERAGE
ENV CODE_COVERAGE=${CODE_COVERAGE}
ENV BRANCH_NAME=${BRANCH_NAME}

# 8. Build commands
RUN <build commands>
```

**Key principles:**
- Static ENV values (`CGO_ENABLED`, `GOPRIVATE`, `SRC_DIR`) go at the very top
- Package installation (`apk add`) comes before any COPY/ADD
- `go.mod` + `go.sum` are copied BEFORE the full source, so `go mod download` is cached
- `ADD . /src/` is the cache-busting boundary
- All build-specific `ARG`/`ENV` (`BRANCH_NAME`, `CODE_COVERAGE`, etc.) go AFTER the source copy
- Remove any `RUN echo "..."` debug lines that use dynamic env vars from the cached section

## Rule 2: Distribution/runtime stage ordering

The correct order for the distribution/runtime stage is:

```dockerfile
# Stage 2 - Binary build stage
######################################
FROM <base-image>

# 1. Static environment variables
ENV SRC_DIR=/app
WORKDIR $SRC_DIR

# 2. Static package installation (cached)
RUN apk add --no-cache <runtime-packages>

# 3. Static directory creation (cached, no dependency on ARGs)
RUN set -eux && \
  mkdir -p /app/conf /app/dockerconf /app/migrations /app/public /app/coverage && \
  chmod a+rw /app/coverage

# 4. Build-specific ARGs/ENVs (change per build)
ARG GIT_COMMIT_HASH
ARG BRANCH_NAME
ARG CODE_COVERAGE
ENV CODE_COVERAGE=${CODE_COVERAGE}
ENV BRANCH_NAME=${BRANCH_NAME}
ENV GIT_COMMIT_HASH=${GIT_COMMIT_HASH}

# 5. Operations that depend on build args
RUN echo "$GIT_COMMIT_HASH" > /app/public/commit.txt

# 6. Copy binaries from builder
COPY --from=builder <source> <dest>

# 7. Static runtime config
ENV GOCOVERDIR=$SRC_DIR/coverage
ENV PATH=$PATH:/root/.local/bin

EXPOSE <ports>
ENTRYPOINT [<entrypoint>]
```

**Key principles:**
- Static package installation and directory creation go BEFORE changing ARGs
- Split `mkdir` from `echo "$GIT_COMMIT_HASH"` so mkdir is cached
- `COPY --from=builder` goes after ARG/ENV setup
- Remove `RUN echo "CODE_COVERAGE is set to: ..."` debug lines (they bust cache and provide no runtime value)

## Rule 3: Consolidate duplicate package installations

If there are multiple `RUN apk add` calls with overlapping packages, merge them into one:

```dockerfile
# BEFORE (bad - two separate calls):
RUN apk add --update --no-cache make git unzip file
RUN apk update && apk add --update --no-cache ca-certificates make git && update-ca-certificates 2>/dev/null || true

# AFTER (good - one consolidated call):
RUN apk add --update --no-cache --repository https://dl-4.alpinelinux.org/alpine/latest-stable/community/ make git unzip file ca-certificates && update-ca-certificates 2>/dev/null || true
```

## Rule 4: Remove redundant go mod download

If there are two `RUN go mod download` calls - one before and one after `ADD . /src/` - remove the one after `ADD`.

## Rule 5: Fix indentation issues

Dockerfile instructions should not have leading whitespace. Fix any indented `ADD`, `RUN`, `COPY` instructions that have accidental leading spaces.

## Rule 6: Convert ARG GIT_TOKEN to --mount=type=secret

If `GIT_TOKEN` is passed via `ARG GIT_TOKEN` and used in `RUN` instructions (e.g., `git config`, `.netrc` setup, or directly in URLs), convert it to use `--mount=type=secret,id=GIT_TOKEN` with `GIT_ASKPASS`. This ensures that changing the token value does not invalidate the Docker cache for the `go mod download` layer.

```dockerfile
# BEFORE (any of these patterns):
ARG GIT_TOKEN
RUN git config --global url."https://${GIT_TOKEN}@github.com/".insteadOf "https://github.com/" && \
    go mod download

# OR:
ARG GIT_TOKEN
RUN echo "machine github.com login ${GIT_TOKEN}" > ~/.netrc && \
    go mod download

# OR:
ARG GIT_TOKEN
RUN printf "machine github.com\n  login %s\n" "${GIT_TOKEN}" >> /root/.netrc && \
    go mod download

# AFTER (all should become):
RUN --mount=type=secret,id=GIT_TOKEN set -eux && \
    printf '#!/bin/sh\nexec cat /run/secrets/GIT_TOKEN\n' > /tmp/askpass.sh && \
    chmod +x /tmp/askpass.sh && \
    export GIT_ASKPASS=/tmp/askpass.sh && \
    go mod download
```

**Key points:**
- Remove the `ARG GIT_TOKEN` line entirely (the token is no longer a build arg)
- Remove any `RUN git config ...` or `RUN echo ... > ~/.netrc` lines that used the token
- The `--mount=type=secret` makes the token available at `/run/secrets/GIT_TOKEN` only during the RUN instruction, without baking it into any layer
- `GIT_ASKPASS` is the standard Git mechanism for providing credentials; it calls the script which reads the mounted secret
- Since the secret is mounted (not an ARG), its value is not part of the layer cache key, so token rotations don't bust the cache
- If `GIT_USERNAME` was also used as an ARG only for auth, it can be removed too

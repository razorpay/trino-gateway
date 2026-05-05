# Reference: pg-router Dockerfile.api (Good Caching Example)

## Builder stage

```dockerfile
FROM <base> as builder

ENV CGO_ENABLED 0
ENV GOPRIVATE github.com/razorpay/*

RUN mkdir /src
WORKDIR /src

RUN apk add --update --no-cache --repository http://dl-4.alpinelinux.org/alpine/latest-stable/community/ make openssh ca-certificates git && update-ca-certificates 2>/dev/null || true

COPY go.mod .
COPY go.sum .

RUN --mount=type=secret,id=GIT_TOKEN set -eux && \
    printf '#!/bin/sh\nexec cat /run/secrets/GIT_TOKEN\n' > /tmp/askpass.sh && \
    chmod +x /tmp/askpass.sh && \
    export GIT_ASKPASS=/tmp/askpass.sh && \
    go mod download

ADD . /src/

RUN make easyjson

ARG CODE_COVERAGE
ENV CODE_COVERAGE=${CODE_COVERAGE}

RUN make go-build-api
```

## Distribution stage

```dockerfile
FROM <base>
ARG GIT_COMMIT_HASH

ARG BRANCH_NAME
ARG CODE_COVERAGE
ENV CODE_COVERAGE=${CODE_COVERAGE}
ENV BRANCH_NAME=${BRANCH_NAME}

COPY --from=builder /src/bin/api /app/
COPY --from=builder /src/config/ /app/config/
COPY build/docker/probe.sh /app/
COPY build/docker/entrypoint.sh /app/

ENV WORKDIR=/app
ENV DUMB_INIT_SETSID=0
WORKDIR /app

RUN apk add --update --no-cache dumb-init su-exec ca-certificates curl
ENV GIT_COMMIT_HASH=${GIT_COMMIT_HASH}
RUN mkdir -p /app/public /app/coverage && \
    chmod a+rw $WORKDIR/coverage && \
    echo "$GIT_COMMIT_HASH" > /app/public/commit.txt

ENV GOCOVERDIR=$WORKDIR/coverage
ENV PATH=$PATH:/root/.local/bin

EXPOSE 9400

RUN chmod +x entrypoint.sh probe.sh
ENTRYPOINT ["/app/entrypoint.sh", "api"]
```

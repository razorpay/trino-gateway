ARG TARGET_PLATFORM
FROM --platform=linux/${TARGET_PLATFORM} golang:1.22.3-alpine3.20

ARG GIT_COMMIT_HASH
ENV GIT_COMMIT_HASH=${GIT_COMMIT_HASH}
ENV TRINO-GATEWAY_APP_GITCOMMITHASH=${GIT_COMMIT_HASH}

WORKDIR /app


RUN apk update \
    && apk add --no-cache bash make protobuf protobuf-dev git gzip curl build-base

COPY ./ /app

RUN go mod download \
    && make setup build
# RUN go mod vendor

COPY ./build/docker/prod/probe.sh /app/probe.sh

ENTRYPOINT /app/build/docker/prod/entrypoint.sh

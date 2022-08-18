FROM golang:1.18.5-alpine3.16

ARG GIT_COMMIT_HASH
ENV GIT_COMMIT_HASH=${GIT_COMMIT_HASH}
ENV TRINO-GATEWAY_APP_GITCOMMITHASH=${GIT_COMMIT_HASH}

WORKDIR /app


RUN apk update \
    && apk add --no-cache bash make protobuf protobuf-dev git gzip curl

COPY ./ /app

RUN go mod download \
    && make setup build build-frontend
# RUN go mod vendor

COPY ./build/docker/prod/probe.sh /app/probe.sh

ENTRYPOINT /app/build/docker/prod/entrypoint.sh

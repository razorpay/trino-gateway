FROM golang:1.17.1-alpine3.14

ARG GIT_COMMIT_HASH
ENV GIT_COMMIT_HASH=${GIT_COMMIT_HASH}

WORKDIR /app


RUN apk update \
    && apk add --no-cache bash make protobuf git gzip curl

COPY ./ /app

RUN go install github.com/golang/protobuf/protoc-gen-go@v1.5.2 \
    && cp /go/bin/protoc-gen-go /usr/bin/ \
    && go install github.com/twitchtv/twirp/protoc-gen-twirp@v8.1.0+incompatible \
    && cp /go/bin/protoc-gen-twirp /usr/bin/ \
    && go install github.com/elliots/protoc-gen-twirp_swagger@v0.0.0-20210801014231-a541d6eeae24 \
    && cp /go/bin/protoc-gen-twirp_swagger /usr/bin \
    && go install github.com/gopherjs/gopherjs@1.17.0+go1.17.1

RUN go mod download \
    && make build build-frontend \
    && go mod vendor

COPY ./build/docker/prod/probe.sh /app/probe.sh

ENTRYPOINT /app/build/docker/prod/entrypoint.sh

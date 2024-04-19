FROM golang:1.19.13-alpine3.18

WORKDIR /app


RUN apk update \
    && apk add --no-cache bash make protobuf protobuf-dev git gzip curl build-base

# COPY ./ /app
# RUN go mod download

RUN go install github.com/githubnemo/CompileDaemon@v1.4.0

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

# RUN go mod vendor
RUN go mod download

ENTRYPOINT /app/build/docker/dev/entrypoint.sh

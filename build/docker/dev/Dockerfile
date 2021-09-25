FROM golang:1.17.1-alpine3.14

WORKDIR /app


RUN apk update \
    && apk add --no-cache bash make protobuf git

# COPY ./ /app
# RUN go mod download

RUN go install github.com/githubnemo/CompileDaemon@v1.4.0

RUN go install github.com/golang/protobuf/protoc-gen-go@v1.5.2 \
    && cp /go/bin/protoc-gen-go /usr/bin/

RUN go install github.com/twitchtv/twirp/protoc-gen-twirp@v8.1.0+incompatible \
    && cp /go/bin/protoc-gen-twirp /usr/bin/

RUN go install github.com/elliots/protoc-gen-twirp_swagger@v0.0.0-20210801014231-a541d6eeae24 \
    && cp /go/bin/protoc-gen-twirp_swagger /usr/bin

RUN go install github.com/gopherjs/gopherjs@1.17.0+go1.17.1

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

# RUN go mod vendor
RUN go mod download

ENTRYPOINT /app/build/docker/dev/entrypoint.sh

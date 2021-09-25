FROM golang:1.16.7-alpine3.13

WORKDIR /app


RUN apk update && apk add bash
# COPY ./ /app
# RUN go mod download

RUN go get github.com/githubnemo/CompileDaemon

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

RUN go mod download

RUN go get github.com/gopherjs/gopherjs@1.16.4+go1.16.7

RUN apk add make

ENTRYPOINT CompileDaemon --build="go build cmd/gateway/main.go -o gateway" --command=./gateway

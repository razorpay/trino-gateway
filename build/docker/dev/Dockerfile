FROM golang:1.17.1-alpine3.14

WORKDIR /app


RUN apk update && apk add bash make

# COPY ./ /app
# RUN go mod download

RUN go install github.com/githubnemo/CompileDaemon@v1.4.0
RUN go install github.com/gopherjs/gopherjs@1.17.0+go1.17.1

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

RUN go mod download

ENTRYPOINT CompileDaemon --build="go build cmd/gateway/main.go -o gateway" --command=./gateway

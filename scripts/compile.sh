#!/bin/bash

protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto
protoc --go_out=. --twirp_swagger_out=./swaggerui/ ./rpc/gateway/service.proto

go mod tidy

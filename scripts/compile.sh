#!/bin/bash

go mod vendor
go mod download

protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto
protoc --go_out=. --twirp_swagger_out=./third_party/swaggerui/ ./rpc/gateway/service.proto

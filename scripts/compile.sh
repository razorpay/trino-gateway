#!/bin/bash

go mod download

protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto
protoc --go_out=. --twirp_swagger_out=./swaggerui/ ./rpc/gateway/service.proto

gopherjs build ./internal/frontend/main --output "./web/frontend/js/frontend.js" --verbose

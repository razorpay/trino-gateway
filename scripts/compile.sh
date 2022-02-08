#!/bin/bash

go mod download

protoc -I . \
    -I "$GOPATH/pkg/mod/github.com/grpc-ecosystem/grpc-gateway/v2@$(go list -m -mod=mod -u github.com/grpc-ecosystem/grpc-gateway/v2 | awk '{print $2}')" \
    --openapiv2_opt logtostderr=true \
    --openapiv2_opt generate_unbound_methods=true \
    --openapiv2_out ./third_party/swaggerui \
    --twirp_out=. \
    --go_out=. \
    rpc/gateway/service.proto

go mod vendor

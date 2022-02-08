#!/bin/bash

go mod download

go install github.com/golang/protobuf/protoc-gen-go
go install github.com/gopherjs/gopherjs
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
go install github.com/twitchtv/twirp/protoc-gen-twirp

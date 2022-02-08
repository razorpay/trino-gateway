//go:build tools
// +build tools

package tools

import (
	// _ "github.com/elliots/protoc-gen-twirp_swagger"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/gopherjs/gopherjs"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/twitchtv/twirp/protoc-gen-twirp"
)

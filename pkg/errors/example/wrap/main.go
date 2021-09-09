package main

import (
	goErr "errors"
	"fmt"

	"github.com/razorpay/trino-gateway/pkg/errors"
)

var (
	// creating a custom error type with base type as bad request
	CustomError = errors.NewClass("custom", errors.SeverityCritical, errors.BadRequest)

	GatewayError = errors.NewClass("gateway_error", errors.SeverityCritical, nil)

	IdentifierCodeApp  errors.IdentifierCode = "PGA000001"
	IdentifierCodeCard errors.IdentifierCode = "PGC000001"

	ErrorCodeCustom  errors.ErrorCode = "custom"
	ErrorCodeGateway errors.ErrorCode = "gateway"
)

func init() {
	// bind the custom error code with the public struct
	// this will be used while serializing the error
	// for public response
	//
	//  if not set then default public response will be sent for the code
	errors.RegisterMultiple(map[errors.IdentifierCode]errors.IPublic{
		IdentifierCodeApp: &errors.Public{
			Code:        "custom_error",
			Description: "this is a custom error",
		},
		IdentifierCodeCard: &errors.Public{
			Code:        "gateway_error",
			Description: "this is a gateway error",
		},
	}, map[errors.IdentifierCode]errors.ErrorCode{
		IdentifierCodeApp:  ErrorCodeCustom,
		IdentifierCodeCard: ErrorCodeGateway,
	})
}

func main() {
	err := doSomething()

	// check if the error is wrapped
	// and extended on top of CustomError
	if errors.Is(err, CustomError) {
		fmt.Println("it's a custom error")
	} else {
		fmt.Println("not a custom error")
	}
}

func doSomething() errors.IError {
	err := externalPkg()

	iErr := CustomError.
		New(IdentifierCodeApp).
		Wrap(err)

	return GatewayError.New(IdentifierCodeCard).Wrap(iErr)
}

func externalPkg() error {
	return goErr.New("sample error")
}

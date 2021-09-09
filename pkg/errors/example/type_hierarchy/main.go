package main

import (
	"fmt"

	"github.com/razorpay/trino-gateway/pkg/errors"
)

var (
	// creating a custom error type with base type as bad request
	CustomError = errors.NewClass("custom", errors.SeverityMedium, errors.BadRequest)

	ErrorCodeCustom errors.ErrorCode = "custom"

	IdentifierCodeDefault errors.IdentifierCode = "default"
)

func init() {
	// bind the custom error code with the public struct
	// this will be used while serializing the error
	// for public response
	errors.Register(IdentifierCodeDefault, &errors.Public{
		Code:        "custom_error",
		Description: "this is a custom error",
	}, ErrorCodeCustom)
}

func main() {
	err := doSomething()

	// check if the error is of type
	// if the error has a class hierarchy
	// if any of the class match the type then it'll be true
	if errors.Is(err, errors.BadRequest) {
		fmt.Println("it's a bad request")
	} else {
		fmt.Println("unknown error")
	}
}

func doSomething() errors.IError {
	return CustomError.
		New(IdentifierCodeDefault).
		WithInternalMetadata(map[string]string{
			"key": "value",
		}).WithPublicMetadata(
		// these fields will be appended to the public error defined
		map[string]string{
			"key": "value",
		})
}

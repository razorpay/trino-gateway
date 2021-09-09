package main

import (
	"fmt"

	"github.com/razorpay/trino-gateway/pkg/errors"
)

var (
	IdentifierCodeDefault errors.IdentifierCode = "default"

	ErrorCodeDefault errors.ErrorCode = "default"
)

func init() {
	// bind the custom error code with the public struct
	// this will be used while serializing the error
	// for public response
	errors.Register(IdentifierCodeDefault, &errors.Public{
		Code:        "bad_request",
		Description: "invalid request sent",
		Field:       "",
		Source:      "",
		Step:        "",
		Reason:      "",
		Metadata:    nil,
	}, ErrorCodeDefault)
}

func main() {
	// Short hand
	// it creates error of type Base
	err := errors.New("something is wrong")

	// check if the error is of type
	if errors.Is(err, errors.Base) {
		fmt.Println("this is a base error")
	}

	// Using error types
	err = doSomething()

	// check if the error is of type
	if errors.Is(err, errors.BadRequest) {
		fmt.Println("it's a bad request")
	} else {
		fmt.Println("unknown error")
	}

	// base error is parent for every error class
	if errors.Is(err, errors.Base) {
		fmt.Println("bas request error is also a base error")
	}
}

func doSomething() errors.IError {
	return errors.BadRequest.
		New(errors.IdentifierDefault).
		WithInternalMetadata(map[string]string{
			"key": "value",
		})
}

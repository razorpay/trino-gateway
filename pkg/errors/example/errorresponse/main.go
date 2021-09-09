package main

import (
	"encoding/json"
	"fmt"

	errorv1 "github.com/razorpay/trino-gateway/pkg/errors/response/common/error/v1"

	"github.com/razorpay/trino-gateway/pkg/errors"
)

var (
	ErrorCodeBadRequest errors.ErrorCode = "bad_request"

	IdentifierCodeDefault errors.IdentifierCode = "default"
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
	}, ErrorCodeBadRequest)
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

	// Error response will be converted to this format
	// in case you need only the public response to be generated
	// then sent the internal as nil and vise versa is also true
	res := &errorv1.Response{
		Error:    &errorv1.Error{},
		Internal: &errorv1.Internal{},
	}

	// this will convert the error into the target struct
	if errors.As(err, res) {
		fmt.Println("error is serialized to given format")
		val, _ := json.Marshal(res)
		fmt.Println(string(val))
	}
}

func doSomething() errors.IError {
	return errors.BadRequest.
		New(IdentifierCodeDefault).
		WithInternalMetadata(map[string]string{
			"key": "value",
		})
}

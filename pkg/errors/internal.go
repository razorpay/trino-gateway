package errors

import (
	errorv1 "github.com/razorpay/trino-gateway/pkg/errors/response/common/error/v1"
)

// IInternal defined the interface definition for internal error details
// this interface will be used by the application
// and the details it holds, won't be exposed outside the applications
type IInternal interface {
	error

	// Code will return the error eCode
	Code() ErrorCode

	IdentifierCode() IdentifierCode

	// Cause will return the cause of the error if there any
	Cause() error

	// MetaData will give the meta data of the error
	Metadata() map[string]string

	// StackTrace gives the error stack trance
	StackTrace() StackTrace

	// As will convert the error into error response format specified by the target
	As(target interface{}) bool
}

// internal struct holds the internal data of the application
type internal struct {
	// code internal error code
	code ErrorCode

	// identifierCode unique error identifier across the services
	identifierCode IdentifierCode

	// metadata error meta data for internal application use
	// we are keeping the data as map[string]string to
	// strictly type the data. This will be really helpful
	// when we log the data and we can index the keys for faster access
	// ref: https://docs.google.com/document/d/1qWTN0-nQ4sCm9ufwt7VEWgzRi7LYA3wneWUsqv69gbQ/edit#heading=h.yvoi31eiuw9c
	metadata map[string]string

	// cause root cause of current error
	cause error

	// stack error stack trace from its origin
	stack *stack
}

// newInternal will create a new instance internal struct
// and fill that with given error eCode
func newInternal(identifierCode IdentifierCode) *internal {
	return &internal{
		code:           identifierCode.GetInternalCode(),
		identifierCode: identifierCode,
		metadata:       make(map[string]string),
		stack:          callers(1),
	}
}

// Code will return the error eCode
func (i *internal) Code() ErrorCode {
	return i.code
}

// IdentifierCode gives unique error identifier code
func (i *internal) IdentifierCode() IdentifierCode {
	return i.identifierCode
}

// Cause will return the cause of the error
// if there any
func (i *internal) Cause() error {
	return i.cause
}

// Error will give the error as string
// if there is description then returns the eCode
func (i *internal) Error() string {
	if i.cause == nil {
		return i.code.String()
	}

	return i.cause.Error()
}

// MetaData will give the meta data of the error
func (i *internal) Metadata() map[string]string {
	return i.metadata
}

// StackTrace gives the error stack trance
func (i *internal) StackTrace() StackTrace {
	return i.stack.StackTrace()
}

// As will convert the error into error response format specified by the target
func (i *internal) As(target interface{}) bool {
	var status bool

	switch target.(type) {
	case *errorv1.Internal:
		res := target.(*errorv1.Internal)
		res.Code = i.Code().String()
		res.Description = i.Error()

		status = true
	}

	return status
}

// withMetadata appends the meta data given as with available meta details
func (i *internal) withMetadata(fields map[string]string) {
	for key, val := range fields {
		i.metadata[key] = val
	}
}

// withCause will add error description detail
func (i *internal) withCause(cause error) {
	i.cause = cause
}

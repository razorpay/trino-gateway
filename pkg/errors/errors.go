package errors

import (
	"context"
	"fmt"
	"strings"

	errorv1 "github.com/razorpay/trino-gateway/pkg/errors/response/common/error/v1"
)

// Base is the base error class for all other classes
var Base = class{
	eType: TypeBase,
}

type IError interface {
	error

	// Class returns the error class
	Class() IClass

	// Internal returns the internal error
	Internal() IInternal

	// Public returns the Public error
	// with details defined in error code repo
	Public() IPublic

	// Is checks if source and destination error are of the same class
	// this wont check the wrapped error recursively
	// if we have to check all the wrapped errors then use errors.Is method
	Is(err error) bool

	// As will transform the error into target interface
	// if the valid target is passed
	As(target interface{}) bool

	// Wrap holds the cause of the error
	Wrap(err error) IError

	// Unwrap will return the cause of error
	Unwrap() error

	// WithPublicMetadata updates the public error details meta data information
	// this will only update the keys if the key is not already present
	WithPublicMetadata(fields map[string]string) IError

	// WithInternalMetadata appends the given fields with internal error fields
	// this will update the data if the key is not present
	WithInternalMetadata(fields map[string]string) IError

	// WithPublicField updates the field key's value in public error response
	WithPublicField(field string) IError

	// WithPublicDescriptionParams updates public error description place holders
	// with the given params in order. If the number of params does not match
	// the number of placeholders then it'll not replace the placeholders
	WithPublicDescriptionParams(publicDescParams ...interface{}) IError

	// Report runs ReporterHook(s)
	Report(ctx context.Context) IError
}

// ReporterHook is used to perform custom actions using IError.
type ReporterHook func(context.Context, IError)

var hooks []ReporterHook

func Initialize(h ...ReporterHook) {
	hooks = h
}

// New creates a new base error with given error message
// this error will have the severity critical
func New(code string) IError {
	return Base.New(IdentifierCode(code))
}

// Error struct holds the error details
// this consists of three sections
// 1. class: defines the type of error
// 2. public: holds the data which can be exposed in Public response
// 3. internal: holds the internal error details for logging and debugging purpose
type Error struct {
	class    class
	public   *Public
	internal *internal
}

// Error constructs the error message with available data
// this error is specifically for internal consumption
func (e Error) Error() string {
	msg := e.class.Type().String() + ": " + e.internal.Error()

	return msg
}

// Class returns the error class
func (e Error) Class() IClass {
	return e.class
}

// Internal returns the internal error
func (e Error) Internal() IInternal {
	return e.internal
}

// Public returns the Public error
func (e Error) Public() IPublic {
	return e.public
}

// WithPublicMetadata updates the public error details meta data information
// this will only update the keys if the key is not already present
func (e Error) WithPublicMetadata(fields map[string]string) IError {
	_ = e.public.withMetadata(fields)

	return e
}

// WithPublicDescriptionParams updates public error description place holders
// with the given params in order. If the number of params does not match
// the number of placeholders then it'll not replace the placeholders
func (e Error) WithPublicDescriptionParams(publicDescParams ...interface{}) IError {
	// if the number of placeholders do match the count of params passed
	// then only replace the placeholders
	if strings.Count(e.public.Description, "%s") == len(publicDescParams) {
		e.public.Description = fmt.Sprintf(e.public.Description, publicDescParams...)
	}

	return e
}

// WithInternalMetadata appends the given fields with internal error fields
// this will update the data if the key is not present
func (e Error) WithInternalMetadata(fields map[string]string) IError {
	e.internal.withMetadata(fields)
	return e
}

// WithPublicField updates the field key's value in public error response
func (e Error) WithPublicField(field string) IError {
	e.public.Field = field
	return e
}

// Wrap holds the cause of the error
func (e Error) Wrap(cause error) IError {
	e.internal.withCause(cause)
	return e
}

// Unwrap will return the cause of error
func (e Error) Unwrap() error {
	return e.internal.Cause()
}

// Is checks if source and destination error are of the same class
// this wont check the wrapped error recursively
// if we have to check all the wrapped errors then use errors.Is method
func (e Error) Is(err error) bool {
	iErr, ok := err.(IError)

	// If the target is not if error type
	// then check if its a class type
	if !ok {
		return e.IsOfClass(err)
	}

	return e.Class().Type() == iErr.Class().Type()
}

// As will transform the error into target interface
// if the valid target is passed
func (e Error) As(target interface{}) bool {
	switch target.(type) {
	case *errorv1.Response:
		res := target.(*errorv1.Response)
		if res.Error != nil && !e.public.As(res.Error) {
			return false
		}

		if res.Internal != nil && !e.internal.As(res.Internal) {
			return false
		}

		return true

	default:
		return false
	}
}

// IsOfClass checks if the error belongs to the class
// where the target type is class
func (e Error) IsOfClass(err error) bool {
	errClass, ok := err.(class)
	// check if the target is of class type
	// if not then the comparison can not be done
	if !ok {
		return false
	}

	// match the error class with given target class
	// considering all the error class hierarchy
	return e.Class().Is(errClass)
}

// Report runs ReporterHook(s)
func (e Error) Report(ctx context.Context) IError {
	for _, hook := range hooks {
		hook(ctx, e)
	}

	return e
}

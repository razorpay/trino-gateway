package errors

import (
	errorv1 "github.com/razorpay/trino-gateway/pkg/errors/response/common/error/v1"
)

// IPublic defined the interface for error details
// which can be exposed out side the application (in response)
type IPublic interface {
	error

	// GetCode returns the error eCode
	GetCode() string

	// GetField returns the field name which caused the error
	GetField() string

	// GetSource returns the source of the error.
	// This would be the name of the module which caused this error
	GetSource() string

	// GetStep returns step when the error occurred
	GetStep() string

	// GetReason returns the error reason
	GetReason() string

	// GetMetadata returns the meta data of error
	GetMetadata() map[string]string

	// As converts the public error into acceptable
	// format defined by target
	As(target interface{}) bool
}

// Public struct holds the details which can be exposed
// outside the application in response to the API call
type Public struct {
	// Code error code which has to to be exposed to the client
	Code string

	// Description error details
	Description string

	// Field will be populated in case the error occurred
	// because of because of particular attribute
	Field string

	// Source source identifier of the error
	// this can also be identify high level flow
	Source string

	// Step stage at which the error occurred in the flow
	Step string

	// Reason details on why this error has occurred
	Reason string

	// Metadata any addition data with respect to the error
	Metadata map[string]string
}

// newPublicFromCode creates a Public error from internal eCode
// this will lookup the error from the map provided
func newPublicFromCode(identifierCode IdentifierCode) *Public {
	err := identifierCode.GetPublicError()
	meta := err.GetMetadata()

	// if there is not meta defined in the public error
	// then allocate memory to store it
	if len(meta) == 0 {
		meta = make(map[string]string)
	}

	return &Public{
		Code:        err.GetCode(),
		Description: err.Error(),
		Field:       err.GetField(),
		Source:      err.GetSource(),
		Step:        err.GetStep(),
		Reason:      err.GetReason(),
		Metadata:    meta,
	}
}

// Error returns the error as string
func (p *Public) Error() string {
	return p.Description
}

// GetCode returns the error eCode
func (p *Public) GetCode() string {
	return p.Code
}

// GetField returns the field name which caused the error
func (p *Public) GetField() string {
	return p.Field
}

// GetSource returns the source of the error.
// This would be the name of the module which caused this error
func (p *Public) GetSource() string {
	return p.Source
}

// GetStep returns step when the error occurred
func (p *Public) GetStep() string {
	return p.Step
}

// GetReason returns the error reason
func (p *Public) GetReason() string {
	return p.Reason
}

// GetMetadata returns the meta data of error
func (p *Public) GetMetadata() map[string]string {
	return p.Metadata
}

// As converts the public error into acceptable
// format defined by target
func (p *Public) As(target interface{}) bool {
	var status bool

	switch target.(type) {
	case *errorv1.Error:
		res := target.(*errorv1.Error)
		res.Code = p.Code
		res.Step = p.Step
		res.Field = p.Field
		res.Source = p.Source
		res.Reason = p.Reason
		res.Metadata = p.Metadata
		res.Description = p.Description

		status = true
	}

	return status
}

// withMetadata appends given meta data with Public meta data
func (p *Public) withMetadata(fields map[string]string) *Public {
	for key, val := range fields {
		p.Metadata[key] = val
	}

	return p
}

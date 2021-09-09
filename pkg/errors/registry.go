package errors

import "sync"

const (
	// TypeBase is the type identifier for the base error
	TypeBase ErrorType = "base"

	// TypeBadRequest is the type identifier for bad request error
	TypeBadRequest ErrorType = "bad_request"

	// TypeRecoverable is something that you know beforehand can happen and take certain measures
	TypeRecoverable ErrorType = "recoverable"

	// TypeValidationFailure is the type identifier for validation error
	TypeValidationFailure ErrorType = "validation_failure"

	// Basic error codes
	CodeDefault ErrorCode = "default"

	// Default identifier code
	IdentifierDefault IdentifierCode = "default"
)

// ErrorType is a custom defined data type
// it holds the error type identifier for the error class
type ErrorType string

// String gives the error type in string
func (et ErrorType) String() string {
	return string(et)
}

// ErrorCode is a custom defined data type
// it holds the error code identifier for an error
type ErrorCode string

// String gives the error type in string
func (ec ErrorCode) String() string {
	return string(ec)
}

// IdentifierCode is a unique code defined to
// get error mapping against it
type IdentifierCode string

// String gives the unique identifier code in string
func (ic IdentifierCode) String() string {
	return string(ic)
}

// GetInternalCode gives the error code associated with the identifier code
func (ic IdentifierCode) GetInternalCode() ErrorCode {
	return mapping.GetInternalCode(ic)
}

func (ic IdentifierCode) GetPublicError() IPublic {
	return mapping.Get(ic)
}

var (
	// BadRequest is a error class for all the bad request type errors
	// for example validation error is type of bad request error
	BadRequest = NewClass(TypeBadRequest, SeverityCritical, nil)

	// Recoverable is a type of error that indicates a condition
	// where the hope is that subsequent processing will be able to recover from it.
	Recoverable = NewClass(TypeRecoverable, SeverityCritical, nil)

	// ValidationFailure this is a derived error from bad request
	// signifies the validation failure in the requests
	ValidationFailure = NewClass(TypeValidationFailure, SeverityCritical, BadRequest)
)

// defaultErrorMap map of internal eCode to public error
var defaultErrorMap = map[IdentifierCode]IPublic{
	IdentifierDefault: &Public{
		Code:        "SERVER_ERROR",
		Description: "The server encountered an error. The incident has been reported to admins.",
	},
}

// defaultInternalErrorMap map of identifier code to internal code
var defaultInternalErrorMap = map[IdentifierCode]ErrorCode{
	IdentifierDefault: CodeDefault,
}

// errorMap holds the error map of internal eCode to public error
type errorMap struct {
	sync.Mutex
	mappingList             map[IdentifierCode]IPublic
	internalCodeMappingList map[IdentifierCode]ErrorCode
}

// mapping holds the error code to public error mapping
var mapping = &errorMap{
	mappingList:             defaultErrorMap,
	internalCodeMappingList: defaultInternalErrorMap,
}

// Get will return the public error for the given internal eCode
// if the eCode does not exist then it'll return the default error
// associated with the default identifier code
func (em *errorMap) Get(identifierCode IdentifierCode) IPublic {
	if detail, ok := em.mappingList[identifierCode]; ok {
		return detail
	}

	return em.mappingList[IdentifierDefault]
}

func (em *errorMap) GetInternalCode(identifierCode IdentifierCode) ErrorCode {
	if errorCode, ok := em.internalCodeMappingList[identifierCode]; ok {
		return errorCode
	}

	return CodeDefault
}

// Register will update the internal eCode to public details map
// *note: if the same eCode is passed, it'll replace the value
func Register(identifierCode IdentifierCode, detail IPublic, internalCode ErrorCode) {
	mapping.Lock()
	defer mapping.Unlock()

	mapping.mappingList[identifierCode] = detail
	mapping.internalCodeMappingList[identifierCode] = internalCode
}

// RegisterMultiple will take the map of internal error eCode
// and associated multiple error details by calling
// Register on every element of the map
func RegisterMultiple(mapping map[IdentifierCode]IPublic, internalMapping map[IdentifierCode]ErrorCode) {
	for k, v := range mapping {
		Register(k, v, internalMapping[k])
	}
}

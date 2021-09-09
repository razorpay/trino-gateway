# Errors
This package implements errors behaviors which are commonly used across all the service in razorpay.
It'll also adhere with the razorpay standard response format and intend to provide the common error definition
to standardise the error usage across the applications.

## Features
1. Supports error code to be loaded from `razorpay/error-mapping-module` repo
2. Supports razorpay stander public response
3. Uses common proto definition for error response
4. Supports error hierarchy and easy comparisons between error
5. Supports custom error response (whether to send internal error or public error or both)
6. Supports reporter hooks which can be used to perform custom actions using errors.

## Getting Started
Note: Please refer to examples folder to see all the use cases

### Import the mutex package
```
import "github.com/razorpay/trino-gateway/pkg/errors"
```

### Error interface

```go
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

  // WithPublicMetaData updates the public error details meta data information
  // this will only update the keys if the key is not already present
  WithPublicMetadata(fields map[string]string) IError

  // WithInternalMetadata appends the given fields with internal error fields
  // this will update the data if the key is not present
  WithInternalMetadata(fields map[string]string) IError

  // WithPublicDescriptionParams updates public error description place holders
  // with the given params in order. If the number of params does not match
  // the number of placeholders then it'll not replace the placeholders
  WithPublicDescriptionParams(publicDescParams ...interface{}) IError

  // Report runs ReporterHook(s)
  Report(ctx context.Context) IError
}

// IClass interface for error class
type IClass interface {
  // Type returns the class eType
  Type() ErrorType

  // Parent will return the base error class
  Parent() IClass

  // severity will return the severity level of error class
  Severity() ISeverity

  // Error makes the class compatible with error interface
  Error() string

  // New returns an Error instance for a specific class.
  New(identifierCode IdentifierCode) IError

  // Is checked if the given class matches any of the class hierarchy
  Is(target IClass) bool
}

// ISeverity severity interface
type ISeverity interface {
  // Is returns true if the both the severity have the same level
  Is(sev ISeverity) bool

  // String stringer implementation of severity
  String() string

  // Level given the integer value of level
  // lower the value higher the criticality
  Level() int
}

// IInternal defined the interface definition for internal error details
// this interface will be used by the application
// and the details it holds, won't be exposed outside the applications
type IInternal interface {
  error

  // Code will return the error eCode
  Code() ErrorCode

  // IdentifierCode gives unique error identifier code
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
```

## Usage

### Initializing error codes
We are keeping all the error codes in a centralized repo `razorpay/error-mapping-module`.
So to load the error details mentioned in this repo, we should initialize the error package

```go
 err := errors.InitMapping(mapper, []string{"service"})
```
This will load all the errors available for the provided service list.

Here `mapper` is a struct which implements the interface
```go
// IErrorMapLoader interface which has to be implemented by error code mappign library
// to read the mapping data and return the data in required format
type IErrorMapLoader interface {
	ReadFilesIntoStruct(services []string) ([]IMapper, error)
}

// IMapper represents the error mapping data which should be loaded into the error package
type IMapper interface {
  GetIdentifierCode() string
  GetInternalErrorCode() string
  GetPublicErrorCode() string
  GetErrorDescription() string
  GetReason() string
  GetFailureType() string
  GetSource() string
  GetStep() string
  GetNextBestAction() string
  GetRecoverable() string
  GetLink() string
}
```

Note: In case if the client tries to use any error code which is not part of the above module
then error public response will be
```
  Code:        "SERVER_ERROR",
  Description: "The server encountered an error. The incident has been reported to admins.",
```

### Initializing Reporter Hooks
```go
errors.Initialize(hooks)
```
Note: These hooks can be used to log errors, reporting them to sentry, push prometheus metrics, etc. See reference implementation in capital-scorecard repo.
### Creating error class

Class defined type of error and also we can categorize it by using the class hierarchy.
We can define error class like
```go
  // BaseRequest class is a parent error
  // when there is no parent class provided then package will by default consider
  // BaseClass as the parent
  BadRequestError = errors.NewClass("bad_request", errors.SeverityLow, nil)

  // ValidationError class is an extended error whose parent is BaseRequest class
  // this will help categorise the error better
  // in this case fields like severity, type can have different value than the parent class
  ValidationError = errors.NewClass("validation_error", errors.SeverityLow, BadRequestError)
```

### Creating an error

Error can be created using class.
```go
// IdentifierCode is the unique code which has been loaded from the 'error-mapping-module'
// in case the code is not valid (not loaded) then the internal error code will be set to 'default'
// and default public error response will be sent back
err := BadRequestError.New(IdentifierCode)
```

Wrap an error to propagate further. This can also be used for modifying goland error into IError interface
```go
err = err.Wrap(errors.New())
```

If required, internal metadata can be added with the error to provide more information to the error.
This should be consumed only by the internal services. Internal metadata can be updated by calling
```go
err = err.WithInternalMetadata(map[string]string{
	"key": "value"
})
```

If required, public metadata can be added with the error to provide more information to the error.
This will be exposed externally in error response. Public metadata can be updated by calling
```go
err = err.WithPublicMetadata(map[string]string{
	"key": "value"
})
```

If the public response contains the placeholder to support dynamic response
Then we can pass the params to replace the placeholder defined in the public error description
example usecase: in case of validation error we have to send the field name for which the validation has failed
```go
err = err.WithPublicDescriptionParams(parram1, param2, ...)
```

#### Shorthand

This package also provides a flexibility to create error without defining a class.
```go
 err := errors.New(IdentifierCode)
```

In this case the BaseClass will be used as error class.

### Error comparison

Errors can be compared using class identifier.
This means we can compare if an error is belongs to a class or of different error

#### Comparing error with class
```go
   // this will return true if the error or any error in the chain belongs to the error class BadRequestError
   // this will also consider the class hierarchy
   errors.Is(err, BadRequestError)

   // this will return true if the err is belongs to the error class BadRequestError
   // this will also consider the class hierarchy
   err.Is(BadRequestError)
```

#### Comparing error with error
```go
   // this will return true if the error or any error in the chain belongs to the error whose class type matches with err1's class
   errors.Is(err, err1)

   // this will return true if the err is belongs to the error whose class type matches with err1's class
   err.Is(err1)
```

### Error serialization
Serializing an error to a response format is achieved by `As` method

```go
errors.As(err, target)
```
or
```go
err.As(targer)
```

In this `target` is a defined structure which is understood by the error package.
This response structure is generated using proto definition [here](https://github.com/razorpay/proto/tree/master/common/error).

Example:
1. Serialize only Public response
```go
target := &errorv1.Response {
   Error: &errorv1.Error{},
}
```

2. Serialize both public and internal error
```go
target := &errorv1.Response {
   Error:    &errorv1.Error{},
   Internal: &errorv1.Internal{},
}
```

## Contribution Guide
* Error serialization only works on defined type.
  So in case there is any change in proto make sure its updated in the error package as well.

* Once the Error proto change are accommodated, make sure to commit the proto generated code.
  To generate the code run `make proto-refresh`.

* Please keep a tab on proto version before publishing the change.

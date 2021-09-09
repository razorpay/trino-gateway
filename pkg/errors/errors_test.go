package errors_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/razorpay/trino-gateway/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	defaultClass = errors.NewClass("default", errors.SeverityCritical, nil)
	testClass    = errors.NewClass("test", errors.SeverityCritical, defaultClass)

	identifierCodeCustomError = errors.IdentifierCode("custom_placeholder")
	customErrorPublicResponse = &errors.Public{
		Code:        "custom_public_code_1",
		Description: "custom description is %s",
		Metadata: map[string]string{
			"key": "value",
		},
	}
)

func init() {
	errors.RegisterMultiple(
		map[errors.IdentifierCode]errors.IPublic{
			identifierCodeCustomError: customErrorPublicResponse,
		},
		map[errors.IdentifierCode]errors.ErrorCode{
			identifierCodeCustomError: "custom_error",
		})
}

func TestError_New(t *testing.T) {
	tests := []struct {
		name     string
		err      errors.IError
		validate func(err errors.IError)
	}{
		{
			name: "basic error without eCode",
			err:  testClass.New(errors.IdentifierDefault),
			validate: func(err errors.IError) {
				assert.Equal(t, errors.IdentifierDefault, err.Internal().IdentifierCode())
				assert.Equal(t, errors.ErrorType("test"), err.Class().Type())
				assert.Equal(t, &errors.Public{
					Code:        "SERVER_ERROR",
					Description: "The server encountered an error. The incident has been reported to admins.",
					Metadata:    map[string]string{},
				}, err.Public())
				assert.Equal(t, "default", err.Internal().Error())
				assert.Equal(t, "test", testClass.Error())
			},
		},
		{
			name: "custom public description with invalid params",
			err:  testClass.New(identifierCodeCustomError),
			validate: func(err errors.IError) {
				assert.Equal(t, identifierCodeCustomError, err.Internal().IdentifierCode())
				assert.Equal(t, errors.ErrorType("test"), err.Class().Type())
				assert.Equal(t, customErrorPublicResponse, err.Public())
				assert.Equal(t, "custom_error", err.Internal().Error())
				assert.Equal(t, "test", testClass.Error())
			},
		},
		{
			name: "custom public description with params",
			err:  testClass.New(identifierCodeCustomError).WithPublicDescriptionParams("test error"),
			validate: func(err errors.IError) {
				assert.Equal(t, identifierCodeCustomError, err.Internal().IdentifierCode())
				assert.Equal(t, errors.ErrorType("test"), err.Class().Type())
				assert.Equal(t, "custom description is test error", err.Public().Error())
				assert.Equal(t, "custom_error", err.Internal().Error())
				assert.Equal(t, "test", testClass.Error())
			},
		},
		{
			name: "basic error with unmapped public detail",
			err:  defaultClass.New("default_error_code"),
			validate: func(err errors.IError) {
				assert.Equal(t, "default", err.Internal().Code().String())
				assert.Equal(t, "default", err.Class().Type().String())
				assert.Equal(t, &errors.Public{
					Code:        "SERVER_ERROR",
					Description: "The server encountered an error. The incident has been reported to admins.",
					Metadata:    map[string]string{},
				}, err.Public())
				assert.Equal(t, "default", err.Internal().Error())
			},
		},
		{
			name: "basic error with mapped public error",
			err:  defaultClass.New(errors.IdentifierDefault),
			validate: func(err errors.IError) {
				assert.Equal(t, errors.IdentifierDefault, err.Internal().IdentifierCode())
				assert.Equal(t, "default", err.Class().Type().String())
				assert.Equal(t, &errors.Public{
					Code:        "SERVER_ERROR",
					Description: "The server encountered an error. The incident has been reported to admins.",
					Metadata:    map[string]string{},
				}, err.Public())
				assert.Equal(t, "default: default", err.Error())
			},
		},
		{
			name: "with public details",
			err: defaultClass.New(errors.IdentifierDefault).
				WithPublicMetadata(map[string]string{
					"key": "value",
				}),
			validate: func(err errors.IError) {
				assert.Equal(t, errors.IdentifierDefault, err.Internal().IdentifierCode())
				assert.Equal(t, "default", err.Class().Type().String())
				assert.Equal(t, &errors.Public{
					Code:        "SERVER_ERROR",
					Description: "The server encountered an error. The incident has been reported to admins.",
					Metadata: map[string]string{
						"key": "value",
					},
				}, err.Public())
			},
		},
		{
			name: "with internal details",
			err: defaultClass.New("default_error_code").
				Wrap(errors.New("some error")).
				WithInternalMetadata(map[string]string{
					"key": "value",
				}),
			validate: func(err errors.IError) {
				assert.Equal(t, errors.ErrorCode("default"), err.Internal().Code())
				assert.Equal(t, errors.ErrorType("default"), err.Class().Type())
				assert.Equal(t, "base: default", err.Unwrap().Error())
				assert.Equal(t, map[string]string{
					"key": "value",
				}, err.Internal().Metadata())
				assert.Equal(t, "base: default", err.Internal().Error())
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.validate(testCase.err)
		})
	}
}

func TestError_Is(t *testing.T) {
	tests := []struct {
		name  string
		err1  error
		err2  error
		match bool
	}{
		{
			name:  "different errors",
			err1:  defaultClass.New(errors.IdentifierDefault),
			err2:  testClass.New(errors.IdentifierDefault),
			match: false,
		},
		{
			name:  "same errors",
			err1:  defaultClass.New(errors.IdentifierDefault),
			err2:  defaultClass.New("bad_request_code"),
			match: true,
		},
		{
			name:  "wrapped errors",
			err1:  testClass.New("bad_request_code").Wrap(defaultClass.New("some_code")),
			err2:  defaultClass.New(errors.IdentifierDefault),
			match: true,
		},
		{
			name:  "wrapped invalid errors",
			err1:  testClass.New("bad_request_code"),
			err2:  fmt.Errorf("some error"),
			match: false,
		},
		{
			name:  "is of class",
			err1:  testClass.New("bad_request_code").Wrap(defaultClass.New("some_code")),
			err2:  defaultClass,
			match: true,
		},
		{
			name:  "is of class hierarchy",
			err1:  testClass.New("bad_request_code").Wrap(defaultClass.New("some_code")),
			err2:  defaultClass,
			match: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			res := errors.Is(test.err1, test.err2)
			assert.Equal(t, test.match, res)
		})
	}
}

func TestError_Report(t *testing.T) {
	b := false
	errors.Initialize(func(ctx context.Context, iError errors.IError) {
		b = true
	})
	_ = testClass.New("bad_request_code").Wrap(defaultClass.New("some_code")).Report(context.Background())
	assert.True(t, b)
}

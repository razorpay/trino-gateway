package errors_test

import (
	"testing"

	"github.com/razorpay/trino-gateway/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	DefaultClass = errors.NewClass("default", errors.SeverityCritical, nil)

	ExtendedClass = errors.NewClass("extended", errors.SeverityMedium, DefaultClass)
)

func TestClass_New(t *testing.T) {
	tests := []struct {
		name     string
		err      errors.IError
		validate func(err errors.IError)
	}{
		{
			name: "new base class error",
			err:  DefaultClass.New("error_code"),
			validate: func(err errors.IError) {
				assert.Equal(t, DefaultClass.Error(), err.Class().Error())
				assert.Equal(t, DefaultClass.Type(), err.Class().Type())
				assert.Equal(t, errors.Base, err.Class().Parent())
				assert.Equal(t, "default: default", err.Error())
				assert.True(t, DefaultClass.Severity().Is(errors.SeverityCritical))
				assert.Equal(t, errors.SeverityCritical.Level(), DefaultClass.Severity().Level())
				assert.Equal(t, errors.SeverityCritical.String(), DefaultClass.Severity().String())
			},
		},
		{
			name: "extended error class",
			err:  ExtendedClass.New("error_code"),
			validate: func(err errors.IError) {
				assert.Equal(t, ExtendedClass.Error(), err.Class().Error())
				assert.Equal(t, ExtendedClass.Type(), err.Class().Type())
				assert.Equal(t, DefaultClass, err.Class().Parent())
				assert.Equal(t, "extended: default", err.Error())
				assert.True(t, ExtendedClass.Severity().Is(errors.SeverityMedium))
				assert.Equal(t, errors.SeverityMedium.Level(), ExtendedClass.Severity().Level())
				assert.Equal(t, errors.SeverityMedium.String(), ExtendedClass.Severity().String())
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.validate(testCase.err)
		})
	}
}

func TestClass_Is(t *testing.T) {
	tests := []struct {
		name        string
		srcClass    errors.IClass
		targetClass errors.IClass
		match       bool
	}{
		{
			name:        "same class",
			srcClass:    DefaultClass,
			targetClass: DefaultClass,
			match:       true,
		},
		{
			name:        "class mismatch",
			srcClass:    errors.NewClass("custom", errors.SeverityCritical, nil),
			targetClass: DefaultClass,
			match:       false,
		},
		{
			name:        "class match the hierarchy",
			srcClass:    ExtendedClass,
			targetClass: DefaultClass,
			match:       true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.match, testCase.srcClass.Is(testCase.targetClass))
		})
	}
}

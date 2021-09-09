package errors_test

import (
	"encoding/json"
	wrapper "errors"
	"testing"

	"github.com/razorpay/trino-gateway/pkg/errors"
	errorv1 "github.com/razorpay/trino-gateway/pkg/errors/response/common/error/v1"
	"github.com/stretchr/testify/assert"
)

func TestUnwrap(t *testing.T) {
	eType := errors.NewClass("sample", errors.SeverityCritical, nil)

	cause := errors.New("test message")
	err := eType.New("sample_code").Wrap(cause)

	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestAs(t *testing.T) {
	err := errors.New("sample error")
	testCases := []struct {
		name          string
		task          func() (bool, *errorv1.Response)
		result        bool
		updatedTarget string
	}{
		{
			name: "v1 complete error response",
			task: func() (bool, *errorv1.Response) {
				res := &errorv1.Response{
					Error:    &errorv1.Error{},
					Internal: &errorv1.Internal{},
				}

				status := errors.As(err, res)
				return status, res
			},
			result:        true,
			updatedTarget: `{"error":{"code":"SERVER_ERROR","description":"The server encountered an error. The incident has been reported to admins."},"internal":{"code":"default","description":"default"}}`,
		},
		{
			name: "v1 error response without internal",
			task: func() (bool, *errorv1.Response) {
				res := &errorv1.Response{
					Error: &errorv1.Error{},
				}

				status := errors.As(err, res)
				return status, res
			},
			result:        true,
			updatedTarget: `{"error":{"code":"SERVER_ERROR","description":"The server encountered an error. The incident has been reported to admins."}}`,
		},
		{
			name: "v1 error response without public",
			task: func() (bool, *errorv1.Response) {
				res := &errorv1.Response{
					Error: &errorv1.Error{},
				}

				status := errors.As(err, res)
				return status, res
			},
			result:        true,
			updatedTarget: `{"error":{"code":"SERVER_ERROR","description":"The server encountered an error. The incident has been reported to admins."}}`,
		},
		{
			name: "invalid target",
			task: func() (bool, *errorv1.Response) {
				res := &errorv1.Error{}

				status := errors.As(err, res)
				return status, nil
			},
			result: false,
		},
		{
			name: "default error conversion",
			task: func() (bool, *errorv1.Response) {
				res := &errorv1.Error{}

				status := errors.As(wrapper.New("some error"), res)
				return status, nil
			},
			result:        true,
			updatedTarget: "null",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, res := testCase.task()
			assert.Equal(t, testCase.result, result)
			if testCase.result {
				bytes, e := json.Marshal(res)
				assert.Nil(t, e)
				assert.Equal(t, testCase.updatedTarget, string(bytes))
			}
		})
	}
}

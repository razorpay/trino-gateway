package errors_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack_StackTrace(t *testing.T) {
	err := defaultClass.New("default_error_code")

	trace := err.Internal().StackTrace()

	assert.Equal(t, 3, len(trace))
	assert.Regexp(t,
		`^.+trino-gateway/pkg/errors_test.TestStack_StackTrace .+/trino-gateway/pkg/errors/stack_test.go:\d+`,
		trace[0].String())
}

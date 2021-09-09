package errors_test

import (
	"testing"

	"github.com/razorpay/trino-gateway/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestError_Register(t *testing.T) {
	public := errors.Public{
		Code:        "custom_public_code",
		Description: "custom description",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	errors.Register("custom_code", &public, "custom_internal_code")
	err := defaultClass.New("custom_code")

	assert.Equal(t, &public, err.Public())
}

func TestError_RegisterMultiple(t *testing.T) {
	public1 := errors.Public{
		Code:        "custom_public_code_1",
		Description: "custom description",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	public2 := errors.Public{
		Code:        "custom_public_code_2",
		Description: "custom description",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	errors.RegisterMultiple(
		map[errors.IdentifierCode]errors.IPublic{
			"test_map_1": &public1,
			"test_map_2": &public2,
		},
		map[errors.IdentifierCode]errors.ErrorCode{
			"test_map_1": "test_internal_error_1",
			"test_map_2": "test_internal_error_2",
		})

	err1 := defaultClass.New("test_map_1")
	err2 := defaultClass.New("test_map_2")

	assert.Equal(t, err1.Internal().IdentifierCode().String(), "test_map_1")
	assert.Equal(t, err2.Internal().IdentifierCode().String(), "test_map_2")

	assert.Equal(t, &public1, err1.Public())
	assert.Equal(t, &public2, err2.Public())
}

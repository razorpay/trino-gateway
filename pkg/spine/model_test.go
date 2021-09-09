package spine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/razorpay/trino-gateway/pkg/errors"
	"github.com/razorpay/trino-gateway/pkg/spine"
)

type Test struct {
	spine.Model
}

func (m *Test) Validate() errors.IError {
	return m.Model.Validate()
}

func TestModel_ValidateError(t *testing.T) {
	m := &Test{
		spine.Model{
			ID:        "1",
			CreatedAt: int64(123),
			UpdatedAt: int64(123),
		},
	}

	err := m.Validate()
	assert.NotNil(t, err)
	assert.Equal(t,
		"validation_failure: created_at: not a valid input; id: not a valid input; updated_at: not a valid input.",
		err.Error())
}

func TestModel_ValidateSuccess(t *testing.T) {
	m := &Test{
		spine.Model{
			ID:        "test123testing",
			CreatedAt: 1587133053,
		},
	}

	err := m.Validate()
	assert.Nil(t, err)

	assert.Equal(t, "test123testing", m.GetID())
}

type SoftTest struct {
	spine.SoftDeletableModel
}

func (s *SoftTest) Validate() errors.IError {
	return s.SoftDeletableModel.Validate()
}

func TestSoftDeletableModel_ValidateError(t *testing.T) {

	m := &SoftTest{
		SoftDeletableModel: spine.SoftDeletableModel{
			Model: spine.Model{
				ID:        "123",
				CreatedAt: 158713305,
			},
		},
	}

	err := m.Validate()
	assert.NotNil(t, err)
	assert.Equal(t,
		"validation_failure: created_at: not a valid input; id: not a valid input.",
		err.Error())
}

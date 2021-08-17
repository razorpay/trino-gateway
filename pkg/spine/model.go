package spine

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/razorpay/trino-gateway/pkg/spine/datatype"
)

const (
	AttributeID        = "id"
	AttributeCreatedAt = "created_at"
	AttributeUpdatedAt = "updated_at"
	AttributeDeletedAt = "deleted_at"
)

type Model struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type IModel interface {
	TableName() string
	EntityName() string
	GetID() string
	Validate() error
	SetDefaults() error
}

// Validate validates base Model.
func (m *Model) Validate() error {
	return GetValidationError(
		validation.ValidateStruct(
			m,
			validation.Field(&m.ID, validation.By(datatype.IsRZPID)),
			validation.Field(&m.CreatedAt, validation.By(datatype.IsTimestamp)),
			validation.Field(&m.UpdatedAt, validation.By(datatype.IsTimestamp)),
		),
	)
}

// GetID gets identifier of entity.
func (m *Model) GetID() string {
	return m.ID
}

// GetCreatedAt gets created time of entity.
func (m *Model) GetCreatedAt() int64 {
	return m.CreatedAt
}

// GetUpdatedAt gets last updated time of entity.
func (m *Model) GetUpdatedAt() int64 {
	return m.UpdatedAt
}

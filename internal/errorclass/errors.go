package errorclass

import (
	"github.com/razorpay/trino-gateway/pkg/errors"
)

var (
	ErrValidationFailure = errors.NewClass("validation_failure", errors.BadRequestValidationFailureException)
)
